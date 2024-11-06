//go:build unit

package cmd

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func cliContextWithArguments(t *testing.T, arguments ...string) *cli.Context {
	var flags flag.FlagSet
	require.NoError(t, flags.Parse(arguments))

	return cli.NewContext(
		&cli.App{},
		&flags,
		nil,
	)
}

func TestNoArgumentsAction(t *testing.T) {
	t.Run(
		"invoke adapted action", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1", "argument-2")
			invoked := false
			adaptedAction := asNoArgumentsAction(
				func(_ *cli.Context) error {
					invoked = true
					return nil
				},
			)

			err := adaptedAction(ctx)

			require.NoError(t, err)
			require.Truef(t, invoked, "adapted action was not invoked")
		},
	)
}

func TestOneArgumentAction(t *testing.T) {
	t.Run(
		"return error when no arguments provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t)
			action := func(_ *cli.Context, arg string) error {
				require.Fail(t, "should not be called when error happens")
				return nil
			}
			adaptedAction := asOneArgumentAction(action, "one argument is required")

			err := adaptedAction(ctx)

			require.ErrorContains(t, err, "one argument is required")
		},
	)

	t.Run(
		"return no error when exactly one argument provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1")
			action := func(_ *cli.Context, arg string) error {
				require.Equal(t, arg, "argument-1")
				return nil
			}
			adaptedAction := asOneArgumentAction(action, "one argument is required")

			err := adaptedAction(ctx)

			require.NoError(t, err)
		},
	)

	t.Run(
		"return no error when more than one argument provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1", "argument-2", "argument-3")
			action := func(_ *cli.Context, arg string) error {
				// only the first argument passed to this function
				require.Equal(t, arg, "argument-1")
				return nil
			}
			adaptedAction := asOneArgumentAction(action, "one argument is required")

			err := adaptedAction(ctx)

			require.NoError(t, err)
		},
	)
}

func TestTwoArgumentsAction(t *testing.T) {
	t.Run(
		"return error when no arguments provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t)
			action := func(_ *cli.Context, arg1 string, arg2 string) error {
				require.Fail(t, "should not be called when error happens")
				return nil
			}
			adaptedAction := asTwoArgumentsAction(action, "two arguments are required")

			err := adaptedAction(ctx)

			require.ErrorContains(t, err, "two arguments are required")
		},
	)

	t.Run(
		"return error when one argument provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1")
			action := func(_ *cli.Context, arg1 string, arg2 string) error {
				require.Fail(t, "should not be called when error happens")
				return nil
			}
			adaptedAction := asTwoArgumentsAction(action, "two arguments are required")

			err := adaptedAction(ctx)

			require.ErrorContains(t, err, "two arguments are required")
		},
	)

	t.Run(
		"return no error when exactly two arguments provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1", "argument-2")
			action := func(_ *cli.Context, arg1 string, arg2 string) error {
				require.Equal(t, "argument-1", arg1)
				require.Equal(t, "argument-2", arg2)
				return nil
			}
			adaptedAction := asTwoArgumentsAction(action, "two arguments are required")

			err := adaptedAction(ctx)

			require.NoError(t, err)
		},
	)

	t.Run(
		"return no error when more than two argument provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1", "argument-2", "argument-3")
			action := func(_ *cli.Context, arg1 string, arg2 string) error {
				// only the first two arguments passed to this function
				require.Equal(t, "argument-1", arg1)
				require.Equal(t, "argument-2", arg2)
				return nil
			}
			adaptedAction := asTwoArgumentsAction(action, "two arguments are required")

			err := adaptedAction(ctx)

			require.NoError(t, err)
		},
	)
}

func TestSliceArgumentAction(t *testing.T) {
	t.Run(
		"return error when no arguments provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t)
			action := func(_ *cli.Context, args []string) error {
				require.Fail(t, "should not be called when error happens")
				return nil
			}
			adaptedAction := asSliceArgumentsAction(action, "one argument is required")

			err := adaptedAction(ctx)

			require.ErrorContains(t, err, "one argument is required")
		},
	)

	t.Run(
		"return no error when arguments provided", func(t *testing.T) {
			ctx := cliContextWithArguments(t, "argument-1", "argument-2", "argument-3")
			action := func(_ *cli.Context, args []string) error {
				require.Equal(t, []string{"argument-1", "argument-2", "argument-3"}, args)
				return nil
			}
			adaptedAction := asSliceArgumentsAction(action, "one argument is required")

			err := adaptedAction(ctx)

			require.NoError(t, err)
		},
	)
}
