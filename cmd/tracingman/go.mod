module github.com/ttys3/tracing-go/cmd/tracingman

go 1.21

replace github.com/ttys3/tracing-go => ../../

require (
	github.com/gofrs/uuid v4.4.0+incompatible
	github.com/ttys3/tracing-go v0.0.0-00010101000000-000000000000
	go.opentelemetry.io/otel/trace v1.23.1
)

require (
	github.com/cenkalti/backoff/v4 v4.2.1 // indirect
	github.com/go-logr/logr v1.4.1 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/golang/protobuf v1.5.3 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.17.1 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.18.0 // indirect
	go.opentelemetry.io/otel v1.23.1 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.17.0 // indirect
	go.opentelemetry.io/otel/metric v1.23.1 // indirect
	go.opentelemetry.io/otel/sdk v1.17.0 // indirect
	go.opentelemetry.io/proto/otlp v1.0.0 // indirect
	golang.org/x/net v0.17.0 // indirect
	golang.org/x/sys v0.13.0 // indirect
	golang.org/x/text v0.13.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20230822172742-b8732ec3820d // indirect
	google.golang.org/grpc v1.57.1 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
