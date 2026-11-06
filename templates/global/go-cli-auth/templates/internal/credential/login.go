package credential

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/cli/browser"
	"golang.org/x/oauth2"
)

const browserTimeout = 5 * time.Minute

type LoginOptions struct {
	Device bool
	Out    io.Writer
	Open   func(address string) error
}

type callback struct {
	code string
	err  error
}

func Login(ctx context.Context, options LoginOptions) (Tokens, string, error) {
	host, err := APIHost()
	if err != nil {
		return Tokens{}, "", err
	}
	store, err := NewStore()
	if err != nil {
		return Tokens{}, "", err
	}
	found, err := discover(ctx)
	if err != nil {
		return Tokens{}, "", err
	}
	ctx = withHTTPClient(ctx)
	var token *oauth2.Token
	if options.Device {
		token, err = deviceLogin(ctx, found, options.Out)
	} else {
		token, err = browserLogin(ctx, found, options)
	}
	if err != nil {
		return Tokens{}, "", err
	}
	tokens := newTokens(token, Tokens{})
	source, err := save(store, host, tokens)
	if err != nil {
		return Tokens{}, "", err
	}
	return tokens, source, nil
}

func BrowserAvailable() bool {
	if os.Getenv("BROWSER") != "" {
		return true
	}
	if os.Getenv("SSH_CONNECTION") != "" || os.Getenv("SSH_TTY") != "" {
		return false
	}
	if runtime.GOOS == "linux" {
		return os.Getenv("DISPLAY") != "" || os.Getenv("WAYLAND_DISPLAY") != ""
	}
	return true
}

func browserLogin(ctx context.Context, found issuer, options LoginOptions) (*oauth2.Token, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	config := found.config()
	config.RedirectURL = fmt.Sprintf("http://%s/callback", listener.Addr())
	state := oauth2.GenerateVerifier()
	verifier := oauth2.GenerateVerifier()
	results := make(chan callback, 1)
	server := &http.Server{Handler: callbackHandler(state, results), ReadHeaderTimeout: 10 * time.Second}
	go func() { _ = server.Serve(listener) }()
	defer func() {
		stop, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(stop)
	}()

	address := config.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
	_, _ = fmt.Fprintf(options.Out, "Open this URL in a browser to log in:\n\n    %s\n\n", address)
	open := options.Open
	if open == nil {
		open = openBrowser
	}
	if err := open(address); err != nil {
		_, _ = fmt.Fprintf(options.Out, "The browser did not open: %v\n", err)
	}

	ctx, cancel := context.WithTimeout(ctx, browserTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("the browser sent no login in %s: %w", browserTimeout, ctx.Err())
	case result := <-results:
		if result.err != nil {
			return nil, result.err
		}
		return config.Exchange(ctx, result.code, oauth2.VerifierOption(verifier))
	}
}

func callbackHandler(state string, results chan<- callback) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		var result callback
		switch {
		case query.Get("state") != state:
			result.err = errors.New("the login response has a different state: run auth login again")
		case query.Get("error") != "":
			result.err = fmt.Errorf("the issuer refused the login: %s %s", query.Get("error"), query.Get("error_description"))
		case query.Get("code") == "":
			result.err = errors.New("the login response has no code")
		default:
			result.code = query.Get("code")
		}
		if result.err != nil {
			http.Error(w, result.err.Error(), http.StatusBadRequest)
		} else {
			_, _ = fmt.Fprintln(w, "The login is complete. You can close this tab.")
		}
		select {
		case results <- result:
		default:
		}
	})
	return mux
}

func deviceLogin(ctx context.Context, found issuer, out io.Writer) (*oauth2.Token, error) {
	if found.DeviceAuthorizationEndpoint == "" {
		return nil, fmt.Errorf("the issuer %s has no device_authorization_endpoint: run auth login without --device on a machine with a browser", found.Issuer)
	}
	config := found.config()
	response, err := config.DeviceAuth(ctx)
	if err != nil {
		return nil, err
	}
	_, _ = fmt.Fprintf(out, "Open %s on any device and enter the code %s\n", response.VerificationURI, response.UserCode)
	return config.DeviceAccessToken(ctx, response)
}

func openBrowser(address string) error {
	command := os.Getenv("BROWSER")
	if command == "" {
		return browser.OpenURL(address)
	}
	cmd := exec.Command("sh", "-c", command+` "$1"`, "sh", address)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
