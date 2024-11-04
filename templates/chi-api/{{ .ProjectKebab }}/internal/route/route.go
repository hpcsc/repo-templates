package route

import "net/http"

type Type string

const (
	PublicRoute    Type = "public"
	ProtectedRoute Type = "protected"
)

type Route struct {
	Type    Type
	Method  string
	Pattern string
	Handler http.HandlerFunc
}

func (r *Route) IsPublic() bool {
	return r.Type == PublicRoute
}

func (r *Route) IsProtected() bool {
	return r.Type == ProtectedRoute
}

func Public(method string, pattern string, handler http.HandlerFunc) *Route {
	return &Route{
		Type:    PublicRoute,
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	}
}

func Protected(method string, pattern string, handler http.HandlerFunc) *Route {
	return &Route{
		Type:    ProtectedRoute,
		Method:  method,
		Pattern: pattern,
		Handler: handler,
	}
}
