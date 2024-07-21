package request

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRouteMatcher(t *testing.T) {
	t.Run("matches exact routes", func(t *testing.T) {
		matcher := NewRouteMatcher([]string{
			"/route-1",
			"/route-2/sub",
		})

		isMatch := matcher("/route-1")
		require.True(t, isMatch)

		isMatch = matcher("/route-2/sub")
		require.True(t, isMatch)
	})

	t.Run("matches pattern routes", func(t *testing.T) {
		matcher := NewRouteMatcher([]string{
			"/route-1/{somePathParameter}/sub",
		})

		isMatch := matcher("/route-1/{pathParameter}/sub")
		require.True(t, isMatch)
	})

	t.Run("does not match unregistered routes", func(t *testing.T) {
		matcher := NewRouteMatcher([]string{
			"/route-1",
			"/route-2/sub",
			"/route-3/{pathParameter}/sub",
		})

		isMatch := matcher("/route-4/{pathParameter}/sub")
		require.False(t, isMatch)

		isMatch = matcher("/route-5")
		require.False(t, isMatch)
	})

}
