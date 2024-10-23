package runner

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/hashicorp/go-multierror"
	"github.com/showwin/speedtest-go/speedtest"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type SpeedTestConfig struct {
	TestInterval         time.Duration
	HealthAndMetricsPort int
}

func StartSpeedTestRunner(ctx context.Context, log *slog.Logger, config SpeedTestConfig) (shutdownFunc func(ctx context.Context) error, err error) {
	metrics, metricsHandler, metricsShutdownFunc, err := initMetrics()
	if err != nil {
		return nil, err
	}

	serverShutdownFunc, err := startHealthAndMetricsServer(ctx, log, metricsHandler, config.HealthAndMetricsPort)
	if err != nil {
		return nil, err
	}

	startSpeedTests(ctx, log, config, metrics)

	shutdownFunc = func(shutdownCtx context.Context) error {
		return multierror.Append(
			serverShutdownFunc(shutdownCtx),
			metricsShutdownFunc(shutdownCtx),
		).ErrorOrNil()
	}

	return shutdownFunc, nil
}

func startHealthAndMetricsServer(ctx context.Context, log *slog.Logger, metricsHandler http.Handler, healthAndMetricsPort int) (shutdownFunc func(ctx context.Context) error, err error) {
	router := mux.NewRouter()

	// Add the health check handlers
	// These are used by systems like kubernetes to check if the container is still alive and well
	// For this particular service, we're not actually serving data to anyone, so readiness and liveness are both identical
	// Liveness determines if the service is able to make forward progress at all
	// AKA, is the process hung
	// So we just return 200, no matter what. If the checking service is able to get our
	// response, then our server stack is at least able to make forward progress

	router.HandleFunc("/readiness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("ready")); err != nil {
			log.Error("Failed to write readiness success to client", "err", err)
		}
	}).Methods(http.MethodGet).Name("readiness")
	router.HandleFunc("/liveness", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("alive")); err != nil {
			log.Error("Failed to write liveness success to client", "err", err)
		}
	}).Methods(http.MethodGet).Name("liveness")

	// Add the main metrics route
	router.Handle("/metrics", metricsHandler)

	srv := &http.Server{
		Addr:        fmt.Sprintf(":%d", healthAndMetricsPort),
		Handler:     router,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("Failed to start listening server", "err", err)
		}
	}()

	log.Info("HTTP Server Started", "port", healthAndMetricsPort)

	shutdownFunc = func(ctx context.Context) error {
		return multierror.Append(
			srv.Shutdown(ctx),
			srv.Close(),
		).ErrorOrNil()
	}

	return shutdownFunc, nil
}

func startSpeedTests(ctx context.Context, log *slog.Logger, config SpeedTestConfig, metrics *speedTestMetrics) {
	// Infinite loop
	go func() {
		for {
			// Break out of the loop when the context is cancelled
			if ctx.Err() != nil {
				return
			}

			startTestTime := time.Now()
			runSpeedTest(ctx, log, metrics)
			testDuration := time.Since(startTestTime)

			sleepDuration := config.TestInterval - testDuration
			time.Sleep(sleepDuration)
		}
	}()
}

func runSpeedTest(ctx context.Context, log *slog.Logger, metrics *speedTestMetrics) {
	metrics.testCount.Add(ctx, 1)

	client := speedtest.New()

	user, err := client.FetchUserInfo()
	if err != nil {
		log.Error("Failed to fetch user info", "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}

	serverList, err := client.FetchServers()
	if err != nil {
		log.Error("Failed to fetch server list", "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}

	targets, err := serverList.FindServer([]int{})
	if err != nil {
		log.Error("Failed to find appropriate server", "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}

	if len(targets) == 0 {
		log.Error("Failed to find appropriate server", "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}

	// We *should* only get 1 back. But just pick it blindly anyway
	target := targets[0]

	attributes := attribute.NewSet(
		userInfoKey.String(user.String()),
		serverHostKey.String(target.Host),
	)

	// Test the server with ping, upload, and download
	err = target.PingTest(func(latency time.Duration) {})
	if err != nil {
		log.Error("Failed to test ping", "user_info", user, "server", target.Host, "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}
	metrics.latency.Record(ctx, target.Latency.Seconds(), metric.WithAttributeSet(attributes))

	err = target.UploadTest()
	if err != nil {
		log.Error("Failed to test upload speed", "user_info", user, "server", target.Host, "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}
	metrics.uploadSpeed.Record(ctx, float64(target.ULSpeed), metric.WithAttributeSet(attributes))

	err = target.DownloadTest()
	if err != nil {
		log.Error("Failed to test download speed", "user_info", user, "server", target.Host, "error", err)

		metrics.testStatus.Record(ctx, 0)
		return
	}
	metrics.downloadSpeed.Record(ctx, float64(target.DLSpeed), metric.WithAttributeSet(attributes))

	// Signal a complete test
	metrics.testStatus.Record(ctx, 1)
}
