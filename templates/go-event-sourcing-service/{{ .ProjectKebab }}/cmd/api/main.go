package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/server"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/common/config"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load[api.Config]()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	pool, err := pgxpool.New(context.Background(), cfg.DB.ConnectionString())
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
	defer pool.Close()

	srv, err := server.New("api", cfg, pool, logger)
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	withCancelCtx, cancelServer := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))

		srv.Shutdown()

		cancelServer()
	}()

	srv.Start()

	<-withCancelCtx.Done()

	logger.Info("exit")
}
