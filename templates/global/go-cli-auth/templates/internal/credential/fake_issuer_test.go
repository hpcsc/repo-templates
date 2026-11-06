//go:build unit

package credential_test

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
)

const testClientID = "test-client"

type fakeIssuer struct {
	server *httptest.Server

	mu             sync.Mutex
	expiresIn      int
	challenges     map[string]string
	verifiedLogins int
	issued         int
	revokedTokens  []string
	otherState     bool
	refusesLogin   bool
	refusesRefresh bool
	refusesRevoke  bool
	noDevice       bool
	otherIssuer    bool
}

func newFakeIssuer(t *testing.T) *fakeIssuer {
	t.Helper()
	f := &fakeIssuer{expiresIn: 3600, challenges: map[string]string{}}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /.well-known/openid-configuration", f.discovery)
	mux.HandleFunc("GET /authorize", f.authorize)
	mux.HandleFunc("POST /device", f.device)
	mux.HandleFunc("POST /token", f.token)
	mux.HandleFunc("POST /revoke", f.revoke)
	f.server = httptest.NewServer(mux)
	t.Cleanup(f.server.Close)
	return f
}

func (f *fakeIssuer) URL() string {
	return f.server.URL
}

func (f *fakeIssuer) change(apply func(*fakeIssuer)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	apply(f)
}

func (f *fakeIssuer) read(get func(*fakeIssuer)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	get(f)
}

func (f *fakeIssuer) discovery(w http.ResponseWriter, _ *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	name := f.server.URL
	if f.otherIssuer {
		name = "https://issuer.example.test"
	}
	document := map[string]string{
		"issuer":                 name,
		"authorization_endpoint": f.server.URL + "/authorize",
		"token_endpoint":         f.server.URL + "/token",
		"revocation_endpoint":    f.server.URL + "/revoke",
	}
	if !f.noDevice {
		document["device_authorization_endpoint"] = f.server.URL + "/device"
	}
	writeJSON(w, http.StatusOK, document)
}

func (f *fakeIssuer) authorize(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	query := r.URL.Query()
	redirect, err := url.Parse(query.Get("redirect_uri"))
	if err != nil || query.Get("client_id") != testClientID {
		http.Error(w, "an unknown client or redirect URL", http.StatusBadRequest)
		return
	}
	values := redirect.Query()
	values.Set("state", query.Get("state"))
	if f.otherState {
		values.Set("state", "another-state")
	}
	switch {
	case query.Get("code_challenge_method") != "S256" || query.Get("code_challenge") == "":
		values.Set("error", "invalid_request")
	case f.refusesLogin:
		values.Set("error", "access_denied")
	default:
		code := fmt.Sprintf("code-%d", len(f.challenges)+1)
		f.challenges[code] = query.Get("code_challenge")
		values.Set("code", code)
	}
	redirect.RawQuery = values.Encode()
	http.Redirect(w, r, redirect.String(), http.StatusFound)
}

func (f *fakeIssuer) device(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"device_code":      "device-1",
		"user_code":        "ABCD-EFGH",
		"verification_uri": f.server.URL + "/activate",
		"expires_in":       600,
		"interval":         1,
	})
}

func (f *fakeIssuer) token(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err := r.ParseForm(); err != nil || r.PostForm.Get("client_id") != testClientID {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_client"})
		return
	}
	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		sum := sha256.Sum256([]byte(r.PostForm.Get("code_verifier")))
		if f.challenges[r.PostForm.Get("code")] != base64.RawURLEncoding.EncodeToString(sum[:]) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}
		f.verifiedLogins++
	case "urn:ietf:params:oauth:grant-type:device_code":
		if r.PostForm.Get("device_code") != "device-1" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}
	case "refresh_token":
		if f.refusesRefresh || r.PostForm.Get("refresh_token") != fmt.Sprintf("refresh-%d", f.issued) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_grant"})
			return
		}
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_grant_type"})
		return
	}
	f.issued++
	claims := base64.RawURLEncoding.EncodeToString([]byte(`{"sub":"user-1","email":"ada@example.com"}`))
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  fmt.Sprintf("access-%d", f.issued),
		"refresh_token": fmt.Sprintf("refresh-%d", f.issued),
		"token_type":    "Bearer",
		"expires_in":    f.expiresIn,
		"id_token":      "eyJhbGciOiJub25lIn0." + claims + ".",
	})
}

func (f *fakeIssuer) revoke(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.refusesRevoke {
		http.Error(w, "the issuer cannot revoke now", http.StatusServiceUnavailable)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	f.revokedTokens = append(f.revokedTokens, r.PostForm.Get("token"))
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
