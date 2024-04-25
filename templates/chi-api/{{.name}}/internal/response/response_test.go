//go:build unit

package response

import (
	"testing"

	"github.com/gookit/validate"
	"github.com/stretchr/testify/require"
)

func TestFailWithValidationErrors(t *testing.T) {
	t.Run("return messages with field, validator and error flatten", func(t *testing.T) {
		errs := validate.Errors(map[string]validate.MS{
			"field-1": {
				"validator-1": "error-1",
				"validator-2": "error-2",
			},
			"field-2": {
				"validator-3": "error-3",
			},
		})

		resp := FailWithValidationErrors(errs)

		require.False(t, resp.Successful)
		require.ElementsMatch(t, []string{
			"field-1: failed validator-1 with error: error-1",
			"field-1: failed validator-2 with error: error-2",
			"field-2: failed validator-3 with error: error-3",
		}, resp.Messages)
	})
}
