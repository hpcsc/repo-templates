package cmd

import (
	"bufio"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

var Version = "main"

func Run() int {
	app := &cli.App{
		Name:                 "{{.ProjectKebab}}",
		Version:              Version,
		EnableBashCompletion: true,
		Action: func(context *cli.Context) error {
			reader := bufio.NewReader(os.Stdin)
			fmt.Printf("Name: ")
			text, err := reader.ReadString('\n')
			if err != nil {
				return fmt.Errorf("failed to read user input: %w", err)
			}

			fmt.Printf("hello %s\n", text)
			return nil
		},
		Commands: []*cli.Command{},
	}

	if err := app.Run(os.Args); err != nil {
		color.Red(err.Error())
		return 1
	}

	return 0
}
