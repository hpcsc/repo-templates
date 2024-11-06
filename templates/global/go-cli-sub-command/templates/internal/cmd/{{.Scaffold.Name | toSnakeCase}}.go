package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

func new{{.Scaffold.Name | toPascalCase}}Subcommand() *cli.Command {
	return &cli.Command{
		Name:    "{{.Scaffold.Name}}",
		Usage:   "{{.Scaffold.Usage}}",
		Aliases: []string{},
		Action:  noArgumentsAction(defaultAction),
		Subcommands: []*cli.Command{
			{
				Name:   "slice-args",
				Usage:  "an example sub-command that takes a slice of arguments",
				Action: asSliceArgumentsAction(sliceArguments, "at least one argument required"),
			},
			{
				Name:   "two-arg",
				Usage:  "an example sub-command that takes two arguments",
				Action: asTwoArgumentsAction(twoArguments, "two arguments required"),
			},
			{
				Name:   "one-arg",
				Usage:  "an example sub-command that takes one argument",
				Action: asOneArgumentAction(oneArgument, "one argument required"),
			},
			{
				Name:   "no-args",
				Usage:  "an example sub-command that takes no arguments",
				Action: asNoArgumentsAction(noArguments),
			},

		},
	}
}

func sliceArguments(arguments []string) error {
	color.Green(fmt.Sprintf("slice arguments: %s", strings.Join(arguments, ", ")))

	return nil
}

func twoArguments(argument1 string, argument2 string) error {
	color.Green(fmt.Sprintf("two arguments: %s - %s", argument1, argument2))

	return nil
}

func oneArgument(argument string) error {
	color.Green(fmt.Sprintf("one argument: %s", argument))

	return nil
}

func noArguments() error {
	color.Green("no arguments")

	return nil
}

func defaultAction() error {
	color.Green("default action")

	return nil
}
