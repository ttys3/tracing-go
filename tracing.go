package tracing

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/propagators/b3"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	// ServiceInstance Tencent Cloud TKE APM only, for view application metrics by pod IP
	ServiceInstance = attribute.Key("service.instance")
)

type TpShutdownFunc func(ctx context.Context) error

func InitProvider(ctx context.Context, opts ...Option) (TpShutdownFunc, error) {
	opt := applayOptions(opts...)
	if opt.otelGrpcEndpoint != "" {
		return InitOtlpTracerProvider(ctx, opt)
	}
	return InitStdoutTracerProvider()
}

type otelErrorHandler struct {
	handler func(err error)
}

func (e *otelErrorHandler) Handle(err error) {
	if e.handler != nil {
		e.handler(err)
		return
	}
	slog.Error("[tracing] send telemetry data failed", err, "provider", "otlp")
}

var emptyTpShutdownFunc = func(_ context.Context) error {
	return nil
}

func applayOptions(opts ...Option) *options {
	options := &options{
		otelGrpcEndpoint: "",
		serviceName:      "no-name",
		serviceVersion:   "0.0.0",
	}
	for _, o := range opts {
		o.apply(options)
	}
	return options
}

// InitOtlpTracerProvider init a tracer provider with otlp exporter with B3 propagator
func InitOtlpTracerProvider(ctx context.Context, opt *options) (TpShutdownFunc, error) {

	otel.SetErrorHandler(&otelErrorHandler{handler: opt.errorHandler})

	expOptions := []otlptracegrpc.Option{
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(opt.otelGrpcEndpoint),
	}

	grpcConnectionTimeout := 3 * time.Second
	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, grpcConnectionTimeout)
	defer cancel()

	traceExp, err := otlptracegrpc.New(ctx, expOptions...)
	if err != nil {
		return emptyTpShutdownFunc, fmt.Errorf("failed to create the collector trace exporter (%w)", err)
	}

	ns := os.Getenv("INSTANCE_NAMESPACE")

	serviceName := opt.serviceName

	// Tencent Cloud TKE APM compatible service name, to avoid duplicated application like `fo`o and `foo.ns`
	if ns != "" && !strings.ContainsRune(serviceName, '.') {
		serviceName += "." + ns
	}

	attrs := []attribute.KeyValue{
		semconv.ServiceNameKey.String(serviceName),
		semconv.ServiceVersionKey.String(opt.serviceVersion),
	}

	if opt.deploymentEnvironment != "" {
		attrs = append(attrs, semconv.DeploymentEnvironmentKey.String(opt.deploymentEnvironment))
	}

	if ns != "" {
		attrs = append(attrs, semconv.ServiceNamespaceKey.String(ns))
	}

	// env from k8s configmap status.podIP
	if ip := os.Getenv("INSTANCE_IP"); ip != "" {
		attrs = append(attrs, ServiceInstance.String(ip))
	}

	attrs = append(attrs, opt.attributes...)

	res, err := resource.New(ctx,
		resource.WithAttributes(attrs...),
	)
	if err != nil {
		return emptyTpShutdownFunc, fmt.Errorf("failed to create resource (%w)", err)
	}

	// sdktrace.WithBatcher(traceExp,
	// sdktrace.WithBatchTimeout(5*time.Second),
	// sdktrace.WithMaxExportBatchSize(10)),
	batchProcessor := sdktrace.NewBatchSpanProcessor(traceExp,
		sdktrace.WithBatchTimeout(5*time.Second),
		sdktrace.WithMaxExportBatchSize(10),
	)
	spanProcessor := batchProcessor

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(1))),
		sdktrace.WithSpanProcessor(spanProcessor),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)

	propagator := b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader))
	otel.SetTextMapPropagator(propagator)

	return tp.Shutdown, nil
}

// InitStdoutTracerProvider is only for unit tests
func InitStdoutTracerProvider() (TpShutdownFunc, error) {
	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		log.Printf("new stdoutrace failed, err=%v", err)
		return emptyTpShutdownFunc, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		b3.New(b3.WithInjectEncoding(b3.B3MultipleHeader)),
		propagation.TraceContext{},
		propagation.Baggage{}))

	return tp.Shutdown, nil
}

func TracerProviderShutdown(ctx context.Context) error {
	if tp, ok := otel.GetTracerProvider().(*sdktrace.TracerProvider); ok {
		log.Printf("shutdown otel tp")
		return tp.Shutdown(ctx)
	}
	return nil
}

// Start creates a span and a context.Context containing the newly-created span.
// If the context.Context provided in `ctx` contains a Span then the newly-created
// Span will be a child of that span, otherwise it will be a root span. This behavior
// can be overridden by providing `WithNewRoot()` as a SpanOption, causing the
// newly-created Span to be a root span even if `ctx` contains a Span.
func Start(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (ctxWithSpan context.Context, newSpan trace.Span) {
	// when we do unit test, we have not called main, the `tracer` is nil, which will cause panic
	// nolint: forbidigo
	ctxWithSpan, newSpan = otel.Tracer("github.com/ttys3/tracing").Start(ctx, spanName, opts...)
	return
}

func StartWithoutCancel(ctx context.Context, spanName string, opts ...trace.SpanStartOption) (ctxWithSpan context.Context, newSpan trace.Span) {
	return Start(WithoutCancel(ctx), spanName, opts...)
}

func TraceID(ctx context.Context) string {
	if span := trace.SpanContextFromContext(ctx); span.HasTraceID() {
		return span.TraceID().String()
	}
	return ""
}

func SpanID(ctx context.Context) string {
	if span := trace.SpanContextFromContext(ctx); span.HasSpanID() {
		return span.SpanID().String()
	}
	return ""
}

func Span(ctx context.Context) trace.Span {
	// SpanFromContext will return a `noopSpan` if ctx is not a valid span
	return trace.SpanFromContext(ctx)
}

// CtxWithSpan wrap a span with parent context
func CtxWithSpan(parent context.Context, span trace.Span) context.Context {
	return trace.ContextWithSpan(parent, span)
}

func NewSpanFromB3(ctx context.Context, header http.Header) trace.Span {
	propagator := b3.New()
	ctx = propagator.Extract(ctx, propagation.HeaderCarrier(header))
	sp := trace.SpanFromContext(ctx)
	return sp
}
