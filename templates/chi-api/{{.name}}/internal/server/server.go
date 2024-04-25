package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v2"
	"github.com/hpcsc/{{.name}}/internal/config"
	"github.com/hpcsc/{{.name}}/internal/usecase"
)

func New(name string, cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: newHandler(name),
		},
		logger: logger,
	}
}

func newHandler(name string) http.Handler {
	r := chi.NewRouter()

	r.Use(httplog.RequestLogger(httplog.NewLogger(name, httplog.Options{
		JSON:           true,
		LogLevel:       slog.LevelInfo,
		Concise:        true,
		RequestHeaders: true,
	})))

	r.Use(middleware.Recoverer)

	usecase.Register(r)

	return r
}

type Server struct {
	cfg        *config.Config
	httpServer *http.Server
	logger     *slog.Logger
}

func (s *Server) Start() {
	s.logger.Info(fmt.Sprintf("listening at %v", s.httpServer.Addr))
	if err := s.httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		s.logger.Error(fmt.Sprintf("failed to start server: %v", err))
	}
}

func (s *Server) Shutdown() {
	// Shutdown signal with grace period of 30 seconds
	withTimeoutCtx, cancelTimeout := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelTimeout()

	go func() {
		<-withTimeoutCtx.Done()
		if errors.Is(withTimeoutCtx.Err(), context.DeadlineExceeded) {
			s.logger.Error("graceful shutdown timed out")
		}
	}()

	s.httpServer.SetKeepAlivesEnabled(false)
	if err := s.httpServer.Shutdown(withTimeoutCtx); err != nil {
		s.logger.Error(fmt.Sprintf("failed to gracefully shutdown server: %v", err))
	} else {
		s.logger.Info("server shutdown")
	}
}
