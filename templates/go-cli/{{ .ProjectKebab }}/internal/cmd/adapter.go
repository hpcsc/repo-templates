package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// an adapter function that adapt a no-arguments function to CLI action handler
func asNoArgumentsAction(f func() error) func(ctx *cli.Context) error {
	return func(_ *cli.Context) error {
		return f()
	}
}

func asOneArgumentAction(f func(string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		query := ctx.Args().First()
		if len(query) == 0 {
			return fmt.Errorf(validationMsg)
		}

		return f(query)
	}
}

func asTwoArgumentsAction(f func(string, string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		query := ctx.Args().Slice()
		if len(query) < 2 {
			return fmt.Errorf(validationMsg)
		}

		return f(query[0], query[1])
	}
}

func asSliceArgumentsAction(f func([]string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		args := ctx.Args().Slice()
		if len(args) == 0 {
			return fmt.Errorf(validationMsg)
		}

		return f(args)
	}
}
