package tracing

import (
	"go.opentelemetry.io/otel/attribute"
)

type options struct {
	otelGrpcEndpoint      string
	serviceName           string
	serviceVersion        string
	deploymentEnvironment string
	attributes            []attribute.KeyValue // keyValue attribute pairs
	errorHandler          func(err error)
	stdoutTrace           bool
}

type setOptionFunc func(*options)

type Option interface {
	apply(*options)
}

func (f setOptionFunc) apply(o *options) {
	f(o)
}

func WithOtelGrpcEndpoint(endpoint string) Option {
	return setOptionFunc(func(o *options) {
		o.otelGrpcEndpoint = endpoint
	})
}

func WithSerivceName(name string) Option {
	return setOptionFunc(func(o *options) {
		o.serviceName = name
	})
}

func WithServiceVersion(version string) Option {
	return setOptionFunc(func(o *options) {
		o.serviceVersion = version
	})
}

func WithDeploymentEnvironment(env string) Option {
	return setOptionFunc(func(o *options) {
		o.deploymentEnvironment = env
	})
}

func WithAttributes(attrs []string) Option {
	return setOptionFunc(func(o *options) {
		if len(attrs) < 2 {
			return
		}
		if len(attrs)%2 != 0 {
			return
		}
		attributes := make([]attribute.KeyValue, 0)
		for i := 0; i < len(attrs)-1; i += 2 {
			key := attrs[i]
			val := attrs[i+1]
			attributes = append(attributes, attribute.String(key, val))
		}
		o.attributes = attributes
	})
}

func WithErrorHandler(handler func(err error)) Option {
	return setOptionFunc(func(o *options) {
		o.errorHandler = handler
	})
}

func WithStdoutTrace() Option {
	return setOptionFunc(func(o *options) {
		o.stdoutTrace = true
	})
}
