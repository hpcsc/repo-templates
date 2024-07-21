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
	"github.com/hpcsc/{{.name}}/internal/config"
	"github.com/hpcsc/{{.name}}/internal/usecase"
	apiMiddleware "github.com/hpcsc/{{.name}}/internal/middleware"
	"github.com/hpcsc/{{.name}}/internal/middleware/request"
)

func New(name string, cfg *config.Config, logger *slog.Logger) (*Server, error) {
	handler, err := newHandler(name, cfg, logger)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg: cfg,
		httpServer: &http.Server{
			Addr:    fmt.Sprintf(":%s", cfg.Port),
			Handler: handler,
		},
		logger: logger,
	}, nil
}

func newHandler(name string, cfg *config.Config, logger *slog.Logger) (http.Handler, error) {
	r := chi.NewRouter()

	r.Use(apiMiddleware.RequestLogger(logger, request.NewRealClock(), nil))

	r.Use(middleware.Recoverer)

	if err := usecase.Register(r, cfg, logger); err != nil {
		return nil, err
	}

	return r, nil
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
