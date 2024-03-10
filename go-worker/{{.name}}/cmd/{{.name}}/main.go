package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hpcsc/{{.name}}/internal/background"
	"github.com/hpcsc/{{.name}}/internal/job"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	Version = "main"
)

func main() {
	logger := log.With().Str("version", Version).Logger()

	ctx, cancel := context.WithCancel(context.Background())
	setupShutdownSignalHandler(logger, cancel)

	runner := background.NewRunner(logger)
	runner.Run(ctx, job.NewMainJob())

	runner.Wait()
	logger.Info().Msg("exit")
}

func setupShutdownSignalHandler(logger zerolog.Logger, cancel context.CancelFunc) {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info().Msgf("received %v signal", s)

		// signal to all other goroutines to start doing their cleanup
		cancel()
	}()
}
