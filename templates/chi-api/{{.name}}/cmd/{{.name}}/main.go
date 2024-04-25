package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hpcsc/{{.name}}/internal/config"
	"github.com/hpcsc/{{.name}}/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	srv := server.New("{{.name}}", cfg, logger)
	withCancelCtx, cancelServer := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))

		srv.Shutdown()

		// Notify main goroutine that shutdown is done
		cancelServer()
	}()

	srv.Start()

	// Wait for server context to be stopped
	<-withCancelCtx.Done()

	logger.Info("exit")
}
