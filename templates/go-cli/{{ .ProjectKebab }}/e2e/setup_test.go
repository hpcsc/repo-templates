//go:build e2e

package e2e

import (
	"os"
	"testing"

	"github.com/hpcsc/{{.ProjectKebab}}/internal/cmd"
	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	os.Exit(testscript.RunMain(m, map[string]func() int{
		"{{.ProjectKebab}}": cmd.Run,
	}))
}
