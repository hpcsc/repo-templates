//go:build unit

package credential_test

import (
	"bytes"
	"errors"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/hpcsc/go-cli-project/internal/credential"
	"github.com/stretchr/testify/require"
)

func isolateWithIssuer(t *testing.T, issuer *fakeIssuer) string {
	t.Helper()
	dir := isolate(t)
	t.Setenv("GO_CLI_PROJECT_OAUTH_ISSUER_URL", issuer.URL())
	t.Setenv("GO_CLI_PROJECT_OAUTH_CLIENT_ID", testClientID)
	return dir
}

func browse(address string) error {
	response, err := http.Get(address)
	if err != nil {
		return err
	}
	return response.Body.Close()
}

func loginInBrowser(t *testing.T) credential.Tokens {
	t.Helper()
	tokens, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})
	require.NoError(t, err)
	return tokens
}

func TestLogin(t *testing.T) {
	t.Run("browser", func(t *testing.T) {
		t.Run("stores the tokens after the issuer checks the PKCE code verifier", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			dir := isolateWithIssuer(t, issuer)

			tokens, source, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})

			require.NoError(t, err)
			require.Equal(t, "ada@example.com", tokens.Account)
			require.Equal(t, filepath.Join(dir, "credentials.json"), source)
			issuer.read(func(f *fakeIssuer) { require.Equal(t, 1, f.verifiedLogins) })
			found, err := credential.Resolve(t.Context())
			require.NoError(t, err)
			require.Equal(t, "access-1", found.Value)
		})

		t.Run("prints the URL, so that a person can open it when the browser does not open", func(t *testing.T) {
			isolateWithIssuer(t, newFakeIssuer(t))
			var out bytes.Buffer
			person := func(address string) error {
				go func() { _ = browse(address) }()
				return errors.New("no browser")
			}

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &out, Open: person})

			require.NoError(t, err)
			require.Contains(t, out.String(), "/authorize?")
			require.Contains(t, out.String(), "The browser did not open: no browser")
		})

		t.Run("rejects a response with another state", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.otherState = true })

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})

			require.ErrorContains(t, err, "different state")
		})

		t.Run("reports a login that the issuer refuses", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.refusesLogin = true })

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})

			require.ErrorContains(t, err, "access_denied")
		})
	})

	t.Run("device", func(t *testing.T) {
		t.Run("prints the code to enter and stores the tokens", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			var out bytes.Buffer

			tokens, _, err := credential.Login(t.Context(), credential.LoginOptions{Device: true, Out: &out})

			require.NoError(t, err)
			require.Equal(t, "ada@example.com", tokens.Account)
			require.Contains(t, out.String(), issuer.URL()+"/activate")
			require.Contains(t, out.String(), "ABCD-EFGH")
			found, err := credential.Resolve(t.Context())
			require.NoError(t, err)
			require.Equal(t, "access-1", found.Value)
		})

		t.Run("fails when the issuer has no device authorization endpoint", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.noDevice = true })

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Device: true, Out: &bytes.Buffer{}})

			require.ErrorContains(t, err, "has no device_authorization_endpoint")
		})
	})

	t.Run("settings", func(t *testing.T) {
		t.Run("without the issuer settings, the error names the constants to set", func(t *testing.T) {
			isolate(t)
			t.Setenv("GO_CLI_PROJECT_OAUTH_ISSUER_URL", "")
			t.Setenv("GO_CLI_PROJECT_OAUTH_CLIENT_ID", "")

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})

			require.ErrorContains(t, err, "set issuerURL and clientID in internal/credential/issuer.go")
		})

		t.Run("rejects a discovery document that names another issuer", func(t *testing.T) {
			issuer := newFakeIssuer(t)
			isolateWithIssuer(t, issuer)
			issuer.change(func(f *fakeIssuer) { f.otherIssuer = true })

			_, _, err := credential.Login(t.Context(), credential.LoginOptions{Out: &bytes.Buffer{}, Open: browse})

			require.ErrorContains(t, err, "names another issuer")
		})
	})
}
