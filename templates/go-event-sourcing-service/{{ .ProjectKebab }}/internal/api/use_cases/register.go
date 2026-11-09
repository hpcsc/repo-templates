package use_cases

import (
{{- if .Scaffold.PushSubscriber }}
	"context"
{{- end }}
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/middleware"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
	apiAccount "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/account"
{{- if .Scaffold.ProcessManager }}
	apiPayment "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/payment"
{{- end }}
{{- if .Scaffold.PushSubscriber }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/websocket"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/root"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/use_cases/webhook"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
{{- if .Scaffold.PushSubscriber }}
	checkpointStore "github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/checkpoint/store"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	coreUseCases "github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/balance"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/payment/summary"
{{- end }}
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(router chi.Router, cfg *api.Config, pool *pgxpool.Pool, logger *slog.Logger) (func(), error) {
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.TokenPath)
	if err != nil {
		return nil, err
	}

{{- if .Scaffold.PushSubscriber }}
	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	wsHub := websocket.NewHub(logger)
	checkpoints := checkpointStore.NewPostgres(pool)
	go startPushSubscriber(context.Background(), pool, reg, wsHub, checkpoints, logger)
{{- end }}

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range allRoutes(pool{{- if .Scaffold.PushSubscriber }}, wsHub{{- end }}) {
		if r.IsPublic() {
			publicRoutes = append(publicRoutes, r)
		} else if r.IsProtected() {
			protectedRoutes = append(protectedRoutes, r)
		}
	}

	router.Route("/api", func(apiRouter chi.Router) {
		for _, r := range publicRoutes {
			apiRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
			logger.Info(fmt.Sprintf("registered public route %s %s", r.Method, r.Pattern))
		}

		apiRouter.Group(func(protectedRouter chi.Router) {
			protectedRouter.Use(authMiddleware)

			for _, r := range protectedRoutes {
				protectedRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
				logger.Info(fmt.Sprintf("registered protected route %s %s", r.Method, r.Pattern))
			}
		})
	})

	return {{- if .Scaffold.PushSubscriber }} func() { wsHub.Shutdown(context.Background()) } {{- else }} nil {{- end }}, nil
}

func allRoutes(pool *pgxpool.Pool{{- if .Scaffold.PushSubscriber }}, wsHub *websocket.Hub{{- end }}) []*route.Route {
	reg := registry.New()
	coreUseCases.RegisterEvents(reg)

	eventStream := stream.NewPostgres(pool, reg)
	bus := command.NewBus(command.WithHistory(eventStream))
	coreUseCases.RegisterHandlers(bus, eventStream)

	routables := []route.Routable{
		root.NewHandler(),
		apiAccount.NewHandler(bus, balance.NewQuery(pool)),
{{- if .Scaffold.ProcessManager }}
		apiPayment.NewHandler(bus, summary.NewQuery(pool)),
		webhook.NewHandler(bus),
{{- end }}
{{- if .Scaffold.PushSubscriber }}
		websocket.NewHandler(wsHub, slog.Default()),
{{- end }}
	}

	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}