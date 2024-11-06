//go:build e2e

package e2e

import (
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func Test{{.Scaffold.Name | toPascalCase }}(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata/{{.Scaffold.Name | toSnakeCase }}",
	})
}
