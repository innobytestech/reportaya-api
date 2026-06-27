package observability

import (
	"context"
	"os"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
	"go.opentelemetry.io/otel/trace/noop"
)

// TracerProvider wraps the SDK provider and exposes Shutdown.
type TracerProvider struct {
	inner   *sdktrace.TracerProvider
	Tracer  trace.Tracer
	Enabled bool
}

// InitTracing sets up the global OTel tracer. Returns a provider that must be
// shut down before the process exits. If OTEL_EXPORTER_OTLP_ENDPOINT is empty,
// a noop tracer is used and no external connection is made.
func InitTracing(ctx context.Context, serviceName, serviceVersion string) (*TracerProvider, error) {
	endpoint := strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
	if endpoint == "" {
		noopProvider := noop.NewTracerProvider()
		otel.SetTracerProvider(noopProvider)
		return &TracerProvider{
			Tracer:  noopProvider.Tracer(serviceName),
			Enabled: false,
		}, nil
	}

	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(endpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(serviceVersion),
		),
	)
	if err != nil {
		return nil, err
	}

	provider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(provider)

	return &TracerProvider{
		inner:   provider,
		Tracer:  provider.Tracer(serviceName),
		Enabled: true,
	}, nil
}

// Shutdown flushes pending spans and closes the exporter.
func (tp *TracerProvider) Shutdown(ctx context.Context) error {
	if tp.inner == nil {
		return nil
	}
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	return tp.inner.Shutdown(shutdownCtx)
}
