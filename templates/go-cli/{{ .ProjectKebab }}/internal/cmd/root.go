package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v3"
)

var Version = "main"

func Run(ctx context.Context) int {
	if err := newCommand().Run(ctx, os.Args); err != nil {
		color.Red(err.Error())
		return 1
	}

	return 0
}

func newCommand() *cli.Command {
	return &cli.Command{
		Name:                  "{{.ProjectKebab}}",
		Version:               Version,
		EnableShellCompletion: true,
		Action: func(_ context.Context, cmd *cli.Command) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Fprint(cmd.Root().Writer, "Name: ")
			text, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			fmt.Fprintf(cmd.Root().Writer, "hello %s\n", text)
			return nil
		},
		Commands: []*cli.Command{},
	}
}
