package main

import (
	"github.com/hpcsc/{{.ProjectKebab}}/internal/cmd"
	"os"
)

func main() {
	os.Exit(cmd.Run())
}
