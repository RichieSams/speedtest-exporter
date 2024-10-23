package cmd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/richiesams/speedtest-exporter/runner"

	"github.com/spf13/cobra"
)

const (
	viperPort         = "port"
	viperOTELEndpoint = "otel_endpoint"
)

func CreateRootCommand() (*cobra.Command, *slog.Logger, error) {
	// Create a logger
	logLevel := slog.LevelVar{}
	logLevel.Set(slog.LevelInfo)

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: &logLevel,
	}))

	testIntervalStr := "30s"
	opts := runner.SpeedTestConfig{
		HealthAndMetricsPort: 8080,
	}

	cmd := &cobra.Command{
		Use:   "speedtest-exporter",
		Short: "Run the speed test exporter service",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			var err error
			opts.TestInterval, err = time.ParseDuration(testIntervalStr)
			if err != nil {
				return fmt.Errorf("failed to parse test-interval as a duration - %w", err)
			}
			if opts.TestInterval <= 0 {
				return fmt.Errorf("--test-interval must be a positive integer")
			}

			cmd.SilenceUsage = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			shutdownFunc, err := runner.StartSpeedTestRunner(ctx, log, opts)
			if err != nil {
				return fmt.Errorf("failed to start server - %w", err)
			}

			// The root context is plumbed up to SIGINT and SIGTERM
			// So we can just wait on that
			<-ctx.Done()

			timeout := 10
			log.Info("Server Stopping... Waiting for in-progress requests to finish", "timeout", timeout)
			shutdownCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
			defer cancel()

			if err := shutdownFunc(shutdownCtx); err != nil {
				log.Error("Shutdown failed", "err", err)
			}

			log.Info("Server Exited")

			return nil
		},
	}

	validLogLevelEnums := []string{
		"debug",
		"info",
		"warn",
		"error",
	}
	cmd.PersistentFlags().String("log-level", "", fmt.Sprintf("The log level to use: [%s]", strings.Join(validLogLevelEnums, ", ")))
	if err := cmd.RegisterFlagCompletionFunc("log-level", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return validLogLevelEnums, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		return nil, log, fmt.Errorf("Failed to register flag completion function - %w", err)
	}

	cmd.Flags().IntVarP(&opts.HealthAndMetricsPort, "port", "p", opts.HealthAndMetricsPort, "The port to use for the health and metrics endpoints")
	cmd.Flags().StringVar(&testIntervalStr, "test-interval", testIntervalStr, "How often the test should run")

	return cmd, log, nil
}
