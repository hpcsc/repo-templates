package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

// an adapter function that adapt a no-arguments function to CLI action handler
func asNoArgumentsAction(f func(*cli.Context) error) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		return f(ctx)
	}
}

func asOneArgumentAction(f func(*cli.Context, string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		query := ctx.Args().First()
		if len(query) == 0 {
			return fmt.Errorf(validationMsg)
		}

		return f(ctx, query)
	}
}

func asTwoArgumentsAction(f func(*cli.Context, string, string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		query := ctx.Args().Slice()
		if len(query) < 2 {
			return fmt.Errorf(validationMsg)
		}

		return f(ctx, query[0], query[1])
	}
}

func asSliceArgumentsAction(f func(*cli.Context, []string) error, validationMsg string) func(ctx *cli.Context) error {
	return func(ctx *cli.Context) error {
		args := ctx.Args().Slice()
		if len(args) == 0 {
			return fmt.Errorf(validationMsg)
		}

		return f(ctx, args)
	}
}
