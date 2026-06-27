package http

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "reportaya-api/http"

// TracingMiddleware creates an OTel span for every incoming request and injects
// the trace_id into Fiber locals so LoggerMiddleware can include it in logs.
// It also propagates W3C traceparent headers from upstream callers.
func TracingMiddleware() fiber.Handler {
	tracer := otel.Tracer(tracerName)
	propagator := otel.GetTextMapPropagator()

	return func(c *fiber.Ctx) error {
		// Extract trace context from incoming headers (W3C traceparent).
		carrier := propagation.MapCarrier{}
		c.Request().Header.VisitAll(func(key, val []byte) {
			carrier[string(key)] = string(val)
		})
		ctx := propagator.Extract(c.Context(), carrier)

		spanName := fmt.Sprintf("%s %s", c.Method(), c.Path())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Method()),
				attribute.String("http.url", c.OriginalURL()),
				attribute.String("http.route", c.Route().Path),
			),
		)
		defer span.End()

		// Make trace/span IDs available to logger and handlers.
		spanCtx := span.SpanContext()
		if spanCtx.IsValid() {
			c.Locals("trace_id", spanCtx.TraceID().String())
			c.Locals("span_id", spanCtx.SpanID().String())
			// Propagate outbound traceparent header so downstream services correlate.
			propagator.Inject(ctx, carrier)
			for k, v := range carrier {
				c.Set(k, v)
			}
		}

		// Store OTel context so handlers can start child spans.
		c.SetUserContext(ctx)

		chainErr := c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Response().StatusCode()))
		if chainErr != nil {
			span.RecordError(chainErr)
		}
		return chainErr
	}
}
