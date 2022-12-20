package tests

import (
	"context"
	"fmt"
	"github.com/ttys3/tracing-go"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func getTestOtlpEp() string {
	ep := "otel-agent.observability.svc.cluster.local:4317"
	if tmp := os.Getenv("OTEL_ENDPOINT"); tmp != "" {
		ep = tmp
	}
	return ep
}

func TestSpanStartNoPanic(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer tracing.TracerProviderShutdown(ctx)

	createTestSpan(ctx)
}

func TestOtlpSpanExport(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer tracing.TracerProviderShutdown(ctx)

	ep := getTestOtlpEp()

	tracing.InitProvider(ctx,
		tracing.WithOtelGrpcEndpoint(ep),
		tracing.WithSerivceName("otel-tracing.test.TestOtlpSpanExport"),
		tracing.WithServiceVersion("1.0.0"),
	)

	createTestSpan(ctx)
}

func createTestSpan(ctx context.Context) {
	ctx, span := tracing.Start(ctx, "test.MySpanName")
	defer span.End()
	log.Printf("begin func, trace_id=%v", tracing.TraceID(ctx))

	func() {
		ctx, span := tracing.Start(ctx, "test.MySubWork01")
		defer span.End()
		time.Sleep(time.Millisecond * 480)

		func() {
			_, span := tracing.Start(ctx, "test.MySubSubWork02")
			defer span.End()
			time.Sleep(time.Millisecond * 120)
		}()
	}()
}

// see go.opentelemetry.io/contrib/propagators/b3@v1.4.0/b3_data_test.go
// go.opentelemetry.io/contrib/propagators/b3@v1.4.0/b3_integration_test.go
const (
	traceIDStr = "4bf92f3577b34da6a3ce929d0e0e4736"
	spanIDStr  = "00f067aa0ba902b7"
)

const (
	b3Context      = "b3"
	b3Flags        = "x-b3-flags"
	b3TraceID      = "x-b3-traceid"
	b3SpanID       = "x-b3-spanid"
	b3Sampled      = "x-b3-sampled"
	b3ParentSpanID = "x-b3-parentspanid"
)

func TestNewSpanFromB3(t *testing.T) {
	ctx := context.Background()

	header := make(http.Header)
	for _, v := range []struct {
		Key string
		Val string
	}{
		{b3TraceID, traceIDStr},
		{b3SpanID, spanIDStr},
		{b3Sampled, "true"},
	} {
		header.Set(v.Key, v.Val)
	}

	sp := tracing.NewSpanFromB3(ctx, header)
	sp.SetName("hello")
	sp.RecordError(fmt.Errorf("oops"))
	sp.SetAttributes(attribute.String("hello", "tracing"))
	sp.End()
}

func TestSpanFromB3PropagatorHeader(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer tracing.TracerProviderShutdown(ctx)

	// otel-collector.service.dc1.consul
	tracing.InitProvider(ctx,
		tracing.WithOtelGrpcEndpoint(getTestOtlpEp()),
		tracing.WithSerivceName("otel-tracing.test.TestSpanFromB3PropagatorHeader"),
		tracing.WithServiceVersion("1.0.0"))

	u := uuid.Must(uuid.NewV4())
	traceID := strings.ReplaceAll(u.String(), "-", "")
	t.Logf("traceID=%v", traceID)

	propagator := b3.New()
	header := make(http.Header)
	for _, v := range []struct {
		Key string
		Val string
	}{
		{b3TraceID, traceID},
		{b3SpanID, spanIDStr},
		{b3Sampled, "true"},
	} {
		header.Set(v.Key, v.Val)
	}
	ctx = propagator.Extract(ctx, propagation.HeaderCarrier(header))

	// warning: trace.SpanFromContext(ctx) will got noopSpan
	sp := trace.SpanFromContext(ctx)
	sp.SetName("SpanFromB3PropagatorHeader")
	if !sp.IsRecording() {
		_, sp = tracing.Start(ctx, "SpanFromB3PropagatorHeader")
	}

	sp.RecordError(fmt.Errorf("oops"))
	sp.SetStatus(codes.Error, "we got an error")
	sp.SetAttributes(attribute.String("hello", "tracing"))
	defer sp.End()
	createTestSpan(ctx)
}

func TestWithoutCancel(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")

	ctx, cancel := context.WithCancel(ctx)

	shutdown, err := tracing.InitStdoutTracerProvider()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		shutdown(context.Background())
	})

	ctx, span := tracing.StartWithoutCancel(ctx, "test.MySpanName")
	defer span.End()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
	out:
		for {
			select {
			case <-ctx.Done():
				t.Logf("ctx.Done(), err=%v", ctx.Err())
			case <-time.After(time.Second * 3):
				t.Logf("begin tracing func, trace_id=%v", tracing.TraceID(ctx))

				func() {
					ctx, span := tracing.Start(ctx, "test.MySubWork01")
					defer span.End()
					time.Sleep(time.Millisecond * 480)

					func() {
						ctx, span := tracing.Start(ctx, "test.MySubSubWork02")
						defer span.End()
						t.Logf("ctx string=%v", ctx)
						time.Sleep(time.Millisecond * 120)
					}()
				}()

				t.Cleanup(func() {
					tracing.TracerProviderShutdown(ctx)
				})
				break out
			}
		}
	}()

	go func() {
		time.Sleep(time.Second * 2)
		cancel()
	}()

	wg.Wait()
}
