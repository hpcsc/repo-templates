module github.com/hpcsc/{{ .ProjectKebab }}

go {{ .Scaffold.GoVersion }}

require (
	github.com/caarlos0/env/v6 v6.10.1
	github.com/go-chi/chi/v5 v5.3.2
	github.com/google/uuid v1.6.0
	github.com/gookit/validate v1.6.0
	github.com/jackc/pgx/v5 v5.11.0
	github.com/stretchr/testify v1.12.1
	github.com/unrolled/render v1.8.1
)

require (
	github.com/fsnotify/fsnotify v1.10.0 // indirect
	github.com/gookit/filter v1.2.3 // indirect
	github.com/gookit/goutil v0.7.6 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/sync v0.21.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.39.0 // indirect
)
