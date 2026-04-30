package telemetry

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

const scopeName = "codex-online/gateway"

// Init sets up OpenTelemetry with stdout exporters and returns a shutdown function.
func Init(_ context.Context) (shutdown func(context.Context) error, err error) {
	var shutdowns []func(context.Context) error

	combined := func(ctx context.Context) error {
		var errs []error
		for _, fn := range shutdowns {
			if e := fn(ctx); e != nil {
				errs = append(errs, e)
			}
		}
		return errors.Join(errs...)
	}

	// Traces.
	traceExp, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return combined, err
	}
	tp := sdktrace.NewTracerProvider(sdktrace.WithBatcher(traceExp))
	otel.SetTracerProvider(tp)
	shutdowns = append(shutdowns, tp.Shutdown)

	// Metrics.
	metricExp, err := stdoutmetric.New()
	if err != nil {
		return combined, err
	}
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExp, sdkmetric.WithInterval(10*time.Second))),
	)
	otel.SetMeterProvider(mp)
	shutdowns = append(shutdowns, mp.Shutdown)

	return combined, nil
}

// Tracer returns the gateway tracer.
func Tracer() trace.Tracer {
	return otel.Tracer(scopeName)
}

// Meter returns the gateway meter.
func Meter() metric.Meter {
	return otel.Meter(scopeName)
}
