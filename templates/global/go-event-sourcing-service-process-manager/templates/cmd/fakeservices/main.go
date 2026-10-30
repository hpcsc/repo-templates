package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/hpcsc/go-event-sourcing-service-project/internal/common/config"
	"github.com/hpcsc/go-event-sourcing-service-project/internal/fakeservices"
)

type Config struct {
	Port         string        `env:"PORT" envDefault:"4444"`
	TokenPath    string        `env:"TOKEN_PATH" envDefault:"/var/run/token"`
	WebhookDelay time.Duration `env:"WEBHOOK_DELAY" envDefault:"1s"`
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load[Config]()
	if err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}

	token, err := os.ReadFile(cfg.TokenPath)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to read token at %s: %v", cfg.TokenPath, err))
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Port),
		Handler: fakeservices.NewHandler(strings.TrimSpace(string(token)), cfg.WebhookDelay, logger),
	}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
		s := <-sig
		logger.Info(fmt.Sprintf("received %v signal", s))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
	}()

	logger.Info(fmt.Sprintf("fake outside systems listening at %v", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error(fmt.Sprintf("failed to start server: %v", err))
		os.Exit(1)
	}
	logger.Info("exit")
}
