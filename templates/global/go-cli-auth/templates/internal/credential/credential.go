package credential

import (
	"context"
	"errors"
	"fmt"
	"os"
{{- if .Computed.oauth }}
	"time"
{{- end }}
)

const (
	name                = "go-cli-project"
	environmentVariable = "GO_CLI_PROJECT_{{ .Computed.variable }}"
)

var ErrMissing = errors.New("no {{ .Computed.label }}")

type Credential struct {
	Value  string
	Source string
{{- if .Computed.oauth }}
	Account string
	Expiry  time.Time
{{- end }}
}

func Resolve(ctx context.Context) (Credential, error) {
	if value := os.Getenv(environmentVariable); value != "" {
		return Credential{Value: value, Source: environmentVariable}, nil
	}
{{- if .Computed.stored }}
	stored, err := {{ if .Computed.oauth }}storedTokens(ctx){{ else }}storedKey(){{ end }}
	if err == nil {
		return stored, nil
	}
	if !errors.Is(err, ErrNotStored) {
		return Credential{}, err
	}
{{- end }}
	path, err := configPath()
	if err != nil {
		return Credential{}, err
	}
	command, err := credentialCommand(path)
	if err != nil {
		return Credential{}, err
	}
	if command != "" {
		return runCommand(ctx, command, path)
	}
	return Credential{}, fmt.Errorf(`%w: {{ if .Computed.stored }}run "%s auth login", {{ end }}set %s, or set credentialCommand in %s`, ErrMissing, {{ if .Computed.stored }}name, {{ end }}environmentVariable, path)
}
{{- if and .Computed.stored (not .Computed.oauth) }}

func storedKey() (Credential, error) {
	host, err := APIHost()
	if err != nil {
		return Credential{}, err
	}
	store, err := NewStore()
	if err != nil {
		return Credential{}, err
	}
	return store.Get(host)
}
{{- end }}
