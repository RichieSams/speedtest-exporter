package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/richiesams/speedtest-exporter/cmd"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// On signal, cancel the context
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-done
		cancel()
	}()

	rootCmd, log, err := cmd.CreateRootCommand()
	if err != nil {
		log.Error("Failed to initialize", "err", err)
		os.Exit(1)
	}

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}
