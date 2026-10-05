package usecase

import (
	"fmt"
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/middleware"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/route"
	apiAccount "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/usecase/account"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/api/usecase/root"
{{- if .Scaffold.ProcessManager }}
	apiTransfer "github.com/hpcsc/{{ .ProjectKebab }}/internal/api/usecase/transfer"
{{- end }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/command"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/registry"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/domain/event/stream"
	coreUsecase "github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/account/balance"
{{- if .Scaffold.ProcessManager }}
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/usecase/transfer/process"
{{- end }}
	"github.com/jackc/pgx/v5/pgxpool"
)

func Register(router chi.Router, cfg *api.Config, pool *pgxpool.Pool, logger *slog.Logger) error {
	authMiddleware, err := middleware.NewAuthMiddleware(cfg.TokenPath)
	if err != nil {
		return err
	}

	var publicRoutes, protectedRoutes []*route.Route
	for _, r := range allRoutes(pool) {
		if r.IsPublic() {
			publicRoutes = append(publicRoutes, r)
		} else if r.IsProtected() {
			protectedRoutes = append(protectedRoutes, r)
		}
	}

	for _, r := range publicRoutes {
		router.MethodFunc(r.Method, r.Pattern, r.Handler)
		logger.Info(fmt.Sprintf("registered public route %s %s", r.Method, r.Pattern))
	}

	router.Group(func(protectedRouter chi.Router) {
		protectedRouter.Use(authMiddleware)

		for _, r := range protectedRoutes {
			protectedRouter.MethodFunc(r.Method, r.Pattern, r.Handler)
			logger.Info(fmt.Sprintf("registered protected route %s %s", r.Method, r.Pattern))
		}
	})

	return nil
}

func allRoutes(pool *pgxpool.Pool) []*route.Route {
	reg := registry.New()
	coreUsecase.RegisterEvents(reg)

	bus := command.NewBus()
	coreUsecase.RegisterHandlers(bus, stream.NewPostgres(pool, reg))

	routables := []route.Routable{
		root.NewHandler(),
		apiAccount.NewHandler(bus, balance.NewQuery(pool)),
{{- if .Scaffold.ProcessManager }}
		apiTransfer.NewHandler(bus, process.NewQuery(pool)),
{{- end }}
	}

	var routes []*route.Route
	for _, r := range routables {
		routes = append(routes, r.Routes()...)
	}
	return routes
}
