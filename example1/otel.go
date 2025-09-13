package main

import (
	"context"
	"errors"
	"log"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/stdout/stdoutmetric"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

// setupOTelSDK bootstraps tracing and metrics and returns a shutdown
// function to flush exporters on exit.
func setupOTelSDK() (func(), error) {
	var shutdownFns []func(context.Context) error

	shutdown := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var err error
		for _, fn := range shutdownFns {
			err = errors.Join(err, fn(ctx))
		}
		if err != nil {
			log.Printf("OpenTelemetry shutdown encountered errors: %v", err)
		}
	}

	// Identify this service so exporters show a proper name/version.
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String("dice-service"),
			semconv.ServiceVersionKey.String("0.1.0"),
		),
	)
	if err != nil {
		return shutdown, err
	}

	// Set context propagator for distributed tracing.
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{},
			propagation.Baggage{},
		),
	)

	// ----- Tracing -----
	traceExporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return shutdown, err
	}
	tp := trace.NewTracerProvider(
		trace.WithResource(res),
		trace.WithBatcher(traceExporter, trace.WithBatchTimeout(time.Second)),
	)
	otel.SetTracerProvider(tp)
	shutdownFns = append(shutdownFns, tp.Shutdown)

	// ----- Metrics -----
	metricExporter, err := stdoutmetric.New()
	if err != nil {
		return shutdown, err
	}
	mp := metric.NewMeterProvider(
		metric.WithResource(res),
		metric.WithReader(metric.NewPeriodicReader(
			metricExporter,
			metric.WithInterval(3*time.Second),
		)),
	)
	otel.SetMeterProvider(mp)
	shutdownFns = append(shutdownFns, mp.Shutdown)

	return shutdown, nil
}
