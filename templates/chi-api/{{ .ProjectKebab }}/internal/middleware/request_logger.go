package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/hpcsc/{{.ProjectKebab}}/internal/middleware/request"
)

const longRequestThreshold = 500 * time.Millisecond

func RequestLogger(l *slog.Logger, clock request.Clock, ignoreRoutes []string) func(next http.Handler) http.Handler {
	shouldIgnore := request.NewRouteMatcher(ignoreRoutes)

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			t1 := clock.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			entry := request.NewLogEntry(r, l)
			defer func() {
				statusCode := ww.Status()
				duration := clock.Since(t1)
				if statusCode >= 500 || !shouldIgnore(r.URL.Path) || duration > longRequestThreshold {
					entry.Write(statusCode, 0, ww.Header(), duration, nil)
				}
			}()
			next.ServeHTTP(ww, middleware.WithLogEntry(r, entry))
		}
		return http.HandlerFunc(fn)
	}
}
