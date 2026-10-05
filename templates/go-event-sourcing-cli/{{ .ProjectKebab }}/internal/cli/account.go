package cli

import (
	"context"

	"github.com/hpcsc/{{ .ProjectKebab }}/internal/app"
	"github.com/hpcsc/{{ .ProjectKebab }}/internal/exit"
	"github.com/urfave/cli/v3"
)

func newOpenCmd() *cli.Command {
	return &cli.Command{
		Name:  "open",
		Usage: "open an account",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the ID of the account", Required: true},
			&cli.StringFlag{Name: "owner", Usage: "the owner of the account", Required: true},
		},
		Action: withApp(func(ctx context.Context, cmd *cli.Command, a *app.App) error {
			report, err := a.Open.Run(ctx, cmd.String("id"), cmd.String("owner"))
			if err != nil {
				return err
			}
			return exit.WriteResult(cmd.Writer, report)
		}),
	}
}

func newCreditCmd() *cli.Command {
	return &cli.Command{
		Name:  "credit",
		Usage: "credit an account once for each reference",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the ID of the account", Required: true},
			&cli.Int64Flag{Name: "amount", Usage: "the amount to credit", Required: true},
			&cli.StringFlag{Name: "ref", Usage: "the reference of the credit, for example an invoice number", Required: true},
		},
		Action: withApp(func(ctx context.Context, cmd *cli.Command, a *app.App) error {
			report, err := a.Credit.Run(ctx, cmd.String("id"), cmd.Int64("amount"), cmd.String("ref"))
			if err != nil {
				return err
			}
			return exit.WriteResult(cmd.Writer, report)
		}),
	}
}

func newShowCmd() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show an account, its balance and its credits",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "id", Usage: "the ID of the account", Required: true},
		},
		Action: withApp(func(ctx context.Context, cmd *cli.Command, a *app.App) error {
			report, found, err := a.Show.Run(ctx, cmd.String("id"))
			if err != nil {
				return err
			}
			if !found {
				return exit.NewRefused("no account has the ID " + cmd.String("id"))
			}
			return exit.WriteResult(cmd.Writer, report)
		}),
	}
}

func newEventsCmd() *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "print the events of one stream as JSON, one event on each line",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "stream", Usage: "the stream ID, for example account-acc-1", Required: true},
		},
		Action: withApp(func(ctx context.Context, cmd *cli.Command, a *app.App) error {
			lines, err := a.Events.Run(ctx, cmd.String("stream"))
			if err != nil {
				return err
			}
			for _, line := range lines {
				if err := exit.WriteResult(cmd.Writer, line); err != nil {
					return err
				}
			}
			return nil
		}),
	}
}
