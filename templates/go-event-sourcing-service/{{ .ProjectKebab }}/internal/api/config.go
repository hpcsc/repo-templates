package api

import "github.com/hpcsc/{{ .ProjectKebab }}/internal/common/config"

type Config struct {
	Port           string   `env:"PORT" envDefault:"3333"`
	TokenPath      string   `env:"TOKEN_PATH" envDefault:"/var/run/token"`
	AllowedOrigins []string `env:"ALLOWED_ORIGINS" envSeparator:","`
	DB             config.DB
{{- if .Scaffold.PushNATS }}
	NATS           config.NATS
{{- end }}
}
