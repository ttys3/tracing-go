package tracing

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

func TestSpanStartNoPanic(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer TracerProviderShutdown(ctx)

	createTestSpan(ctx)
}

func TestOtlpSpanExport(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer TracerProviderShutdown(ctx)
	InitOtlpTracerProvider(ctx, WithOtelGrpcEndpoint("tempo.service.dc1.consul:4317"), WithSerivceName("otel-tracing.test.TestOtlpSpanExport"), WithServiceVersion("1.0.0"))

	createTestSpan(ctx)
}

func createTestSpan(ctx context.Context) {
	ctx, span := SpanStart(ctx, "test.MySpanName")
	defer span.End()

	func() {
		ctx, span := SpanStart(ctx, "test.MySubWork01")
		defer span.End()
		time.Sleep(time.Millisecond * 480)

		func() {
			_, span := SpanStart(ctx, "test.MySubSubWork02")
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

	sp := NewSpanFromB3(ctx, header)
	sp.SetName("hello")
	sp.RecordError(fmt.Errorf("oops"))
	sp.SetAttributes(attribute.String("hello", "tracing"))
	sp.End()
}

func TestSpanFromB3PropagatorHeader(t *testing.T) {
	ctx := context.WithValue(context.Background(), "key001", "val007")
	defer TracerProviderShutdown(ctx)

	// otel-collector.service.dc1.consul
	InitOtlpTracerProvider(ctx, WithOtelGrpcEndpoint("otel-collector.service.dc1.consul:4317"), WithSerivceName("otel-tracing.test.TestSpanFromB3PropagatorHeader"), WithServiceVersion("1.0.0"))

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
		_, sp = SpanStart(ctx, "SpanFromB3PropagatorHeader")
	}

	sp.RecordError(fmt.Errorf("oops"))
	sp.SetStatus(codes.Error, "we got an error")
	sp.SetAttributes(attribute.String("hello", "tracing"))
	defer sp.End()
	createTestSpan(ctx)
}
