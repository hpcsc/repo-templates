package cmd

import (
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
		Commands: []*cli.Command{
		},
	}

	if err := app.Run(os.Args); err != nil {
		color.Red(err.Error())
		return 1
	}

	return 0
}
