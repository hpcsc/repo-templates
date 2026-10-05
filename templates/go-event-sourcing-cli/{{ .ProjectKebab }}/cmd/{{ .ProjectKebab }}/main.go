package main

import (
	"context"
	"os"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/cli"
)

func main() {
	os.Exit(cli.Execute(context.Background(), cli.NewRootCmd(), os.Args))
}
