package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/app"
)

const defaultDatabaseURL = "postgres://postgres:postgres@localhost:5432/{{ .ProjectSnake }}?sslmode=disable"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		usage()
		return fmt.Errorf("a command is required")
	}

	ctx := context.Background()

	application, err := app.New(ctx, app.Config{
		DatabaseURL: envOr("DATABASE_URL", defaultDatabaseURL),
		Now:         time.Now,
	})
	if err != nil {
		return err
	}
	defer application.Close()

	switch args[0] {
	case "open":
		return openAccount(ctx, application, args[1:])
	case "credit":
		return creditAccount(ctx, application, args[1:])
	case "show":
		return showAccount(ctx, application, args[1:])
	default:
		usage()
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func openAccount(ctx context.Context, application *app.App, args []string) error {
	flags := flag.NewFlagSet("open", flag.ContinueOnError)
	id := flags.String("id", "", "account id")
	owner := flags.String("owner", "", "account owner")
	if err := flags.Parse(args); err != nil {
		return err
	}

	report, err := application.Open.Run(ctx, *id, *owner)
	if err != nil {
		return err
	}

	return emit(report)
}

func creditAccount(ctx context.Context, application *app.App, args []string) error {
	flags := flag.NewFlagSet("credit", flag.ContinueOnError)
	id := flags.String("id", "", "account id")
	amount := flags.Int64("amount", 0, "amount in minor units")
	reference := flags.String("ref", "", "unique reference, so a retry is safe")
	if err := flags.Parse(args); err != nil {
		return err
	}

	report, err := application.Credit.Run(ctx, *id, *amount, *reference)
	if err != nil {
		return err
	}

	return emit(report)
}

func showAccount(ctx context.Context, application *app.App, args []string) error {
	flags := flag.NewFlagSet("show", flag.ContinueOnError)
	id := flags.String("id", "", "account id")
	if err := flags.Parse(args); err != nil {
		return err
	}

	report, found, err := application.Show.Run(ctx, *id)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no account %q", *id)
	}

	return emit(report)
}

func emit(report any) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(report)
}

func envOr(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Fprint(os.Stderr, `usage:
  {{ .ProjectKebab }} open   -id ACC-1 -owner "Ada Lovelace"
  {{ .ProjectKebab }} credit -id ACC-1 -amount 500 -ref invoice-7
  {{ .ProjectKebab }} show   -id ACC-1

DATABASE_URL overrides the connection string.
`)
}
