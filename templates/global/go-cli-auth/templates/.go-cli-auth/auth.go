package cmd

import (
	"context"
	"fmt"
{{- if and .Computed.stored (not .Computed.oauth) }}
	"errors"
	"io"
	"os"
	"strings"
{{- end }}
{{- if .Computed.oauth }}
	"time"
{{- end }}

	"github.com/hpcsc/go-cli-project/internal/credential"
	"github.com/urfave/cli/v3"
{{- if and .Computed.stored (not .Computed.oauth) }}
	"golang.org/x/term"
{{- end }}
)

func newAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "auth",
		Usage: "{{ if .Computed.oauth }}log in, log out, or show the login{{ else if .Computed.stored }}store, remove, or show the API key{{ else }}show where the API key comes from{{ end }}",
		Commands: []*cli.Command{
			{
				Name:   "status",
				Usage:  "show where the {{ .Computed.label }} comes from, but not the {{ .Computed.label }}",
				Action: authStatus,
			},
{{- if .Computed.stored }}
			{
				Name:  "login",
				Usage: "{{ if .Computed.oauth }}log in with OAuth and store the tokens{{ else }}store the API key, which it reads from the terminal or from stdin{{ end }}",
{{- if .Computed.oauth }}
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "device", Usage: "log in with a code on another device, in place of a browser on this machine"},
				},
{{- end }}
				Action: authLogin,
			},
			{
				Name:   "logout",
				Usage:  "{{ if .Computed.oauth }}revoke the refresh token and remove the stored tokens{{ else }}remove the stored API key{{ end }}",
				Action: authLogout,
			},
{{- end }}
{{- if .Computed.oauth }}
			{
				Name:   "token",
				Usage:  "print the access token, for scripts",
				Action: authToken,
			},
{{- end }}
		},
	}
}

func authStatus(ctx context.Context, cmd *cli.Command) error {
	found, err := credential.Resolve(ctx)
	if err != nil {
		return err
	}
	out := cmd.Root().Writer
	_, err = fmt.Fprintf(out, "The {{ .Computed.label }} comes from %s.\n", found.Source)
{{- if .Computed.oauth }}
	if err == nil && found.Account != "" {
		_, err = fmt.Fprintf(out, "Account: %s\n", found.Account)
	}
	if err == nil && !found.Expiry.IsZero() {
		_, err = fmt.Fprintf(out, "Expires: %s\n", found.Expiry.Local().Format(time.RFC3339))
	}
{{- end }}
	return err
}
{{- if and .Computed.stored (not .Computed.oauth) }}

func authLogin(_ context.Context, cmd *cli.Command) error {
	host, err := credential.APIHost()
	if err != nil {
		return err
	}
	key, err := readAPIKey(cmd)
	if err != nil {
		return err
	}
	store, err := credential.NewStore()
	if err != nil {
		return err
	}
	source, err := store.Set(host, key)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.Root().Writer, "Stored the API key for %s in %s.\n", host, source)
	return err
}

func readAPIKey(cmd *cli.Command) (string, error) {
	in := cmd.Root().Reader
	var text []byte
	var err error
	if file, ok := in.(*os.File); ok && term.IsTerminal(int(file.Fd())) {
		_, _ = fmt.Fprint(cmd.Root().ErrWriter, "Paste the API key: ")
		text, err = term.ReadPassword(int(file.Fd()))
		_, _ = fmt.Fprintln(cmd.Root().ErrWriter)
	} else {
		text, err = io.ReadAll(in)
	}
	if err != nil {
		return "", err
	}
	key := strings.TrimSpace(string(text))
	if key == "" {
		return "", errors.New("no API key: paste it, or send it on stdin")
	}
	return key, nil
}

func authLogout(_ context.Context, cmd *cli.Command) error {
	host, err := credential.APIHost()
	if err != nil {
		return err
	}
	store, err := credential.NewStore()
	if err != nil {
		return err
	}
	removed, err := store.Delete(host)
	if err != nil {
		return err
	}
	if !removed {
		_, err = fmt.Fprintf(cmd.Root().Writer, "No API key for %s is stored.\n", host)
		return err
	}
	_, err = fmt.Fprintf(cmd.Root().Writer, "Removed the API key for %s.\n", host)
	return err
}
{{- end }}
{{- if .Computed.oauth }}

func authLogin(ctx context.Context, cmd *cli.Command) error {
	tokens, source, err := credential.Login(ctx, credential.LoginOptions{
		Device: cmd.Bool("device") || !credential.BrowserAvailable(),
		Out:    cmd.Root().ErrWriter,
	})
	if err != nil {
		return err
	}
	if tokens.Account == "" {
		_, err = fmt.Fprintf(cmd.Root().Writer, "Logged in. The tokens are in %s.\n", source)
		return err
	}
	_, err = fmt.Fprintf(cmd.Root().Writer, "Logged in as %s. The tokens are in %s.\n", tokens.Account, source)
	return err
}

func authLogout(ctx context.Context, cmd *cli.Command) error {
	host, err := credential.APIHost()
	if err != nil {
		return err
	}
	removed, err := credential.Logout(ctx, cmd.Root().ErrWriter)
	if err != nil {
		return err
	}
	if !removed {
		_, err = fmt.Fprintf(cmd.Root().Writer, "No tokens for %s are stored.\n", host)
		return err
	}
	_, err = fmt.Fprintf(cmd.Root().Writer, "Removed the tokens for %s.\n", host)
	return err
}

func authToken(ctx context.Context, cmd *cli.Command) error {
	found, err := credential.Resolve(ctx)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.Root().Writer, found.Value)
	return err
}
{{- end }}
