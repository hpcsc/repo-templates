//go:build unit

package cli_test

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/cli"
	"github.com/stretchr/testify/require"
)

type result struct {
	status int
	stdout string
	stderr string
}

func run(t *testing.T, args ...string) result {
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	var stdout, stderr bytes.Buffer
	root := cli.NewRootCmd()
	root.Writer = &stdout
	root.ErrWriter = &stderr
	status := cli.Execute(context.Background(), root, append([]string{"app"}, args...))
	return result{status: status, stdout: stdout.String(), stderr: stderr.String()}
}

func errorCode(t *testing.T, r result) string {
	var body struct {
		Code string `json:"code"`
	}
	require.NoError(t, json.Unmarshal([]byte(r.stderr), &body))
	return body.Code
}

func TestRoot(t *testing.T) {
	t.Run("open", func(t *testing.T) {
		t.Run("prints the ID of the account that it opens", func(t *testing.T) {
			r := run(t, "open", "--id", "acc-1", "--owner", "Ada")

			require.Equal(t, 0, r.status)
			require.JSONEq(t, `{"account_id": "acc-1"}`, r.stdout)
		})

		t.Run("exits with code 2 when a required flag is missing", func(t *testing.T) {
			r := run(t, "open", "--id", "acc-1")

			require.Equal(t, 2, r.status)
			require.Equal(t, "usage", errorCode(t, r))
		})
	})

	t.Run("credit", func(t *testing.T) {
		t.Run("exits with code 1 when the account is not open", func(t *testing.T) {
			r := run(t, "credit", "--id", "acc-unknown", "--amount", "500", "--ref", "invoice-7")

			require.Equal(t, 1, r.status)
			require.Equal(t, "refused", errorCode(t, r))
		})
	})
}
