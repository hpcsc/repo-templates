//go:build unit

package credential_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/hpcsc/go-cli-project/internal/credential"
	"github.com/stretchr/testify/require"
)

func TestTokens(t *testing.T) {
	t.Run("resolve", func(t *testing.T) {
		t.Run("takes the stored token before the credentialCommand", func(t *testing.T) {
			dir := isolateWithIssuer(t, newFakeIssuer(t))
			writeConfig(t, dir, "credentialCommand: echo from-command\n")
			loginInBrowser(t)

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "access-1", found.Value)
			require.Equal(t, filepath.Join(dir, "credentials.json"), found.Source)
			require.Equal(t, "ada@example.com", found.Account)
		})

		t.Run("takes GO_CLI_PROJECT_TOKEN before the stored token", func(t *testing.T) {
			isolateWithIssuer(t, newFakeIssuer(t))
			loginInBrowser(t)
			t.Setenv("GO_CLI_PROJECT_TOKEN", "from-environment")

			found, err := credential.Resolve(t.Context())

			require.NoError(t, err)
			require.Equal(t, "from-environment", found.Value)
		})

		t.Run("does not give a token that is stored for another API host", func(t *testing.T) {
			isolateWithIssuer(t, newFakeIssuer(t))
			loginInBrowser(t)
			t.Setenv("GO_CLI_PROJECT_API_URL", "https://other.example.test")

			_, err := credential.Resolve(t.Context())

			require.ErrorIs(t, err, credential.ErrMissing)
		})
	})

	t.Run("refresh", func(t *testing.T) {
		t.Run("gets a new access token when the stored one expires, and stores it", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.expiresIn = 1 })
			loginInBrowser(t)
			issuer.change(func(f *fakeIssuer) { f.expiresIn = 3600 })

			refreshed, err := credential.Resolve(t.Context())
			require.NoError(t, err)
			again, err := credential.Resolve(t.Context())
			require.NoError(t, err)

			require.Equal(t, "access-2", refreshed.Value)
			require.Equal(t, "access-2", again.Value)
		})

		t.Run("asks for a new login when the issuer refuses the refresh", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.expiresIn = 1 })
			loginInBrowser(t)
			issuer.change(func(f *fakeIssuer) { f.refusesRefresh = true })

			_, err := credential.Resolve(t.Context())

			require.ErrorContains(t, err, `run "go-cli-project auth login"`)
		})
	})

	t.Run("logout", func(t *testing.T) {
		t.Run("revokes the refresh token and removes the tokens", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			loginInBrowser(t)

			removed, err := credential.Logout(t.Context(), &bytes.Buffer{})

			require.NoError(t, err)
			require.True(t, removed)
			issuer.read(func(f *fakeIssuer) { require.Equal(t, []string{"refresh-1"}, f.revokedTokens) })
			_, err = credential.Resolve(t.Context())
			require.ErrorIs(t, err, credential.ErrMissing)
		})

		t.Run("removes the tokens when the issuer cannot revoke them", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			loginInBrowser(t)
			issuer.change(func(f *fakeIssuer) { f.refusesRevoke = true })
			var warnings bytes.Buffer

			removed, err := credential.Logout(t.Context(), &warnings)

			require.NoError(t, err)
			require.True(t, removed)
			require.Contains(t, warnings.String(), "The issuer did not revoke the token")
			_, err = credential.Resolve(t.Context())
			require.ErrorIs(t, err, credential.ErrMissing)
		})

		t.Run("reports false when nobody is logged in", func(t *testing.T) {
			isolateWithIssuer(t, newFakeIssuer(t))

			removed, err := credential.Logout(t.Context(), &bytes.Buffer{})

			require.NoError(t, err)
			require.False(t, removed)
		})
	})
}
