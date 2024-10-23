package runner

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/go-multierror"
	promreg "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/prometheus"
	api "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
)

const (
	meterName = "ghcr.io/richiesams/speedtest-exporter"

	userInfoKey   = attribute.Key("user.info")
	serverHostKey = attribute.Key("server.host")
)

type speedTestMetrics struct {
	testCount     api.Int64Counter
	testStatus    api.Int64Gauge
	latency       api.Float64Gauge
	uploadSpeed   api.Float64Gauge
	downloadSpeed api.Float64Gauge
}

func initMetrics() (metrics *speedTestMetrics, metricsHandler http.Handler, shutdownFunc func(ctx context.Context) error, err error) {
	registry := promreg.NewRegistry()

	// The exporter embeds a default OpenTelemetry Reader and
	// implements prometheus.Collector, allowing it to be used as
	// both a Reader and Collector.
	exporter, err := prometheus.New(
		prometheus.WithRegisterer(registry),
		prometheus.WithNamespace("speedtest.exporter"),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	provider := metric.NewMeterProvider(metric.WithReader(exporter))
	meter := provider.Meter(meterName)

	metrics = &speedTestMetrics{}

	metrics.testCount, err = meter.Int64Counter(
		"test.count",
		api.WithDescription("The number of tests run"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create test_counter - %w", err)
	}

	metrics.testStatus, err = meter.Int64Gauge(
		"test.status",
		api.WithDescription("The status of a test. 0 == failed. 1 == success"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create test_status - %w", err)
	}

	metrics.latency, err = meter.Float64Gauge(
		"latency",
		api.WithDescription("The latency to the server"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create latency - %w", err)
	}

	metrics.uploadSpeed, err = meter.Float64Gauge(
		"upload.speed",
		api.WithDescription("The speed of the upload"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create upload_speed - %w", err)
	}

	metrics.downloadSpeed, err = meter.Float64Gauge(
		"download.speed",
		api.WithDescription("The speed of the download"),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create download_speed - %w", err)
	}

	handler := promhttp.InstrumentMetricHandler(
		registry, promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry}),
	)

	shutdownFunc = func(ctx context.Context) error {
		return multierror.Append(
			provider.Shutdown(ctx),
			exporter.Shutdown(ctx),
		).ErrorOrNil()
	}

	return metrics, handler, shutdownFunc, nil
}
