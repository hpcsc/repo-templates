package credential

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

const (
	issuerURL         = ""
	clientID          = ""
	issuerURLVariable = "GO_CLI_PROJECT_OAUTH_ISSUER_URL"
	clientIDVariable  = "GO_CLI_PROJECT_OAUTH_CLIENT_ID"
)

var (
	scopes     = []string{"openid", "profile", "email", "offline_access"}
	httpClient = &http.Client{Timeout: 30 * time.Second}
)

type issuer struct {
	Issuer                      string `json:"issuer"`
	AuthorizationEndpoint       string `json:"authorization_endpoint"`
	TokenEndpoint               string `json:"token_endpoint"`
	DeviceAuthorizationEndpoint string `json:"device_authorization_endpoint"`
	RevocationEndpoint          string `json:"revocation_endpoint"`
	clientID                    string
}

func discover(ctx context.Context) (issuer, error) {
	base := strings.TrimSuffix(setting(issuerURLVariable, issuerURL), "/")
	id := setting(clientIDVariable, clientID)
	if base == "" || id == "" {
		return issuer{}, fmt.Errorf("the OAuth client is not set: set issuerURL and clientID in internal/credential/issuer.go, or set %s and %s", issuerURLVariable, clientIDVariable)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/.well-known/openid-configuration", nil)
	if err != nil {
		return issuer{}, err
	}
	response, err := httpClient.Do(request)
	if err != nil {
		return issuer{}, fmt.Errorf("discover the OAuth endpoints of %s: %w", base, err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return issuer{}, fmt.Errorf("discover the OAuth endpoints of %s: %s", base, response.Status)
	}
	var found issuer
	if err := json.NewDecoder(response.Body).Decode(&found); err != nil {
		return issuer{}, fmt.Errorf("discover the OAuth endpoints of %s: %w", base, err)
	}
	if strings.TrimSuffix(found.Issuer, "/") != base {
		return issuer{}, fmt.Errorf("the discovery document of %s names another issuer, %q", base, found.Issuer)
	}
	found.clientID = id
	return found, nil
}

func (i issuer) config() oauth2.Config {
	return oauth2.Config{
		ClientID: i.clientID,
		Endpoint: oauth2.Endpoint{
			AuthURL:       i.AuthorizationEndpoint,
			TokenURL:      i.TokenEndpoint,
			DeviceAuthURL: i.DeviceAuthorizationEndpoint,
			AuthStyle:     oauth2.AuthStyleInParams,
		},
		Scopes: scopes,
	}
}

func (i issuer) revoke(ctx context.Context, tokens Tokens) error {
	if i.RevocationEndpoint == "" {
		return nil
	}
	token, hint := tokens.RefreshToken, "refresh_token"
	if token == "" {
		token, hint = tokens.AccessToken, "access_token"
	}
	form := url.Values{"token": {token}, "token_type_hint": {hint}, "client_id": {i.clientID}}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, i.RevocationEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := httpClient.Do(request)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answered %s", i.RevocationEndpoint, response.Status)
	}
	return nil
}

func withHTTPClient(ctx context.Context) context.Context {
	return context.WithValue(ctx, oauth2.HTTPClient, httpClient)
}
