package config

import (
	"context"
	"net/http"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

var tp *sdktrace.TracerProvider

func setupTraceProvider(ctx context.Context) error {
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(Config.OpenTelemetryGrpcEndpoint),
		otlptracegrpc.WithInsecure(),
	)

	if err != nil {
		return err
	}

	// Create trace provider
	tp = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceName("goliac"),
		)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(tp)

	return nil
}

func ShutdownTraceProvider() error {
	if tp != nil {
		return tp.Shutdown(context.Background())
	}
	return nil
}

type OtelMiddleware struct {
	tracer trace.Tracer
}

func NewOtelMiddleware() *OtelMiddleware {
	tracer := otel.Tracer("negroni-otel")
	return &OtelMiddleware{
		tracer: tracer,
	}
}

func (o *OtelMiddleware) ServeHTTP(rw http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
	// Start a span
	ctx := r.Context()
	_, span := o.tracer.Start(ctx, r.URL.Path)
	defer span.End()

	// Call the next middleware
	next(rw, r)
}
