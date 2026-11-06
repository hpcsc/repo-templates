//go:build unit

package credential_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/hpcsc/go-cli-project/internal/credential"
	"github.com/stretchr/testify/require"
)

func isolate(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", home)
	t.Setenv("GO_CLI_PROJECT_{{ .Computed.variable }}", "")
{{- if .Computed.stored }}
	t.Setenv("GO_CLI_PROJECT_API_URL", "https://api.example.test")
	t.Setenv("GO_CLI_PROJECT_CREDENTIAL_STORE", "file")
{{- end }}
	return filepath.Join(home, "go-cli-project")
}

func writeConfig(t *testing.T, dir, text string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o700))
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(text), 0o600))
	return path
}

func TestCredential(t *testing.T) {
	t.Run("resolve", func(t *testing.T) {
		t.Run("takes the {{ .Computed.label }} from GO_CLI_PROJECT_{{ .Computed.variable }} before the credentialCommand", func(t *testing.T) {
			writeConfig(t, isolate(t), "credentialCommand: echo from-command\n")
			t.Setenv("GO_CLI_PROJECT_{{ .Computed.variable }}", "from-environment")

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "from-environment", found.Value)
			require.Equal(t, "GO_CLI_PROJECT_{{ .Computed.variable }}", found.Source)
		})

		t.Run("runs the credentialCommand in the user config when the environment has no {{ .Computed.label }}", func(t *testing.T) {
			path := writeConfig(t, isolate(t), "credentialCommand: echo from-command\n")

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "from-command", found.Value)
			require.Equal(t, "the credentialCommand in "+path, found.Source)
		})

		t.Run("ignores the other keys in the user config", func(t *testing.T) {
			writeConfig(t, isolate(t), "theme: dark\ncredentialCommand: echo from-command\n")

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "from-command", found.Value)
		})

		t.Run("a credentialCommand that fails is an error that names the config file", func(t *testing.T) {
			path := writeConfig(t, isolate(t), "credentialCommand: exit 3\n")

			_, err := credential.Resolve(t.Context())

			require.ErrorContains(t, err, "run the credentialCommand in "+path)
		})

		t.Run("a credentialCommand that prints nothing is an error", func(t *testing.T) {
			writeConfig(t, isolate(t), "credentialCommand: printf ''\n")

			_, err := credential.Resolve(t.Context())

			require.ErrorContains(t, err, "printed no {{ .Computed.label }}")
		})

		t.Run("with no source, the error says how to give the {{ .Computed.label }}", func(t *testing.T) {
			dir := isolate(t)

			_, err := credential.Resolve(t.Context())

			require.ErrorIs(t, err, credential.ErrMissing)
			require.ErrorContains(t, err, "GO_CLI_PROJECT_{{ .Computed.variable }}")
			require.ErrorContains(t, err, filepath.Join(dir, "config.yaml"))
{{- if .Computed.stored }}
			require.ErrorContains(t, err, `"go-cli-project auth login"`)
{{- end }}
		})
{{- if and .Computed.stored (not .Computed.oauth) }}

		t.Run("takes the stored API key before the credentialCommand", func(t *testing.T) {
			dir := isolate(t)
			writeConfig(t, dir, "credentialCommand: echo from-command\n")
			store, err := credential.NewStore()
			require.NoError(t, err)
			_, err = store.Set("api.example.test", "from-store")
			require.NoError(t, err)

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "from-store", found.Value)
			require.Equal(t, filepath.Join(dir, "credentials.json"), found.Source)
		})

		t.Run("does not give an API key that is stored for another host", func(t *testing.T) {
			isolate(t)
			store, err := credential.NewStore()
			require.NoError(t, err)
			_, err = store.Set("other.example.test", "for-another-host")
			require.NoError(t, err)

			_, err = credential.Resolve(t.Context())

			require.ErrorIs(t, err, credential.ErrMissing)
		})

		t.Run("without an API URL, the error names the constant to set", func(t *testing.T) {
			isolate(t)
			t.Setenv("GO_CLI_PROJECT_API_URL", "")

			_, err := credential.Resolve(t.Context())

			require.ErrorContains(t, err, "set apiURL in internal/credential/api_url.go")
		})
{{- end }}
	})
}
