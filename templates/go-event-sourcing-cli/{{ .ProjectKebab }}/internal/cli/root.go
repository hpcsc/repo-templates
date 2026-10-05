package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/app"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/event"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/exit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/stream"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/credit"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/use_cases/account/open"
	"github.com/urfave/cli/v3"
)

const name = "{{ .ProjectKebab }}"

var version = "dev"

func NewRootCmd() *cli.Command {
	root := &cli.Command{
		Name:    name,
		Usage:   "an event-sourced command-line application",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "db", Usage: "the SQLite file that keeps the events", Value: defaultStorePath(name)},
		},
		Commands: []*cli.Command{
			newOpenCmd(),
			newCreditCmd(),
			newShowCmd(),
			newEventsCmd(),
		},
		// urfave prints the error and calls os.Exit when this handler is nil
		ExitErrHandler: func(context.Context, *cli.Command, error) {},
	}
	classifyUsageErrors(root)
	return root
}

func Execute(ctx context.Context, cmd *cli.Command, args []string) int {
	if err := cmd.Run(ctx, args); err != nil {
		err = mapError(err)
		_ = exit.WriteError(cmd.ErrWriter, err)
		return exit.ExitCodeFor(err)
	}
	return 0
}

func openApp(ctx context.Context, cmd *cli.Command) (*app.App, error) {
	return app.New(ctx, app.Config{
		Now:       func() time.Time { return time.Now().UTC() },
		StorePath: cmd.String("db"),
	})
}

func mapError(err error) error {
	var newer *event.NewerSchemaError
	switch {
	case errors.Is(err, stream.ErrVersionConflict):
		return exit.NewConflict(err.Error())
	case errors.As(err, &newer):
		return exit.New(exit.Usage, "stream-version", err.Error())
	case errors.Is(err, open.ErrNoOwner), errors.Is(err, credit.ErrNotOpen), errors.Is(err, credit.ErrInvalidAmount):
		return exit.NewRefused(err.Error())
	}
	return err
}

func classifyUsageErrors(cmd *cli.Command) {
	cmd.OnUsageError = func(_ context.Context, _ *cli.Command, err error, _ bool) error {
		return exit.NewUsage(err.Error())
	}
	for _, sub := range cmd.Commands {
		classifyUsageErrors(sub)
	}
}

func withApp(run func(ctx context.Context, cmd *cli.Command, a *app.App) error) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		if cmd.Args().Len() > 0 {
			return exit.NewUsage(fmt.Sprintf("%s: unknown argument %q", cmd.Name, cmd.Args().First()))
		}

		a, err := openApp(ctx, cmd)
		if err != nil {
			return err
		}
		defer func() { _ = a.Close() }()

		return run(ctx, cmd, a)
	}
}
