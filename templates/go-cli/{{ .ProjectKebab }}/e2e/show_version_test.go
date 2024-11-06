//go:build e2e

package e2e

import (
	"github.com/rogpeppe/go-internal/testscript"
	"testing"
)

func TestShowVersion(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/show_version",
	})
}
