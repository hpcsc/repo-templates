# Event Model: Account

## Overview

The example of this template is an account. An operator opens it, credits it and shows it from the
command line. Each command loads the stream `account-<id>`, decides, and appends the new events. The
`show` command folds the same stream into a report, so the application has no stored read model.

- Model: [event-model-account.emod](event-model-account.emod), in the [emod](https://github.com/hpcsc/emod) DSL
- Diagram: [event-model-account.svg](event-model-account.svg), drawn from the model

Check the model and draw the diagram again after a change:

```sh
emod validate docs/event-model-account.emod
emod lint docs/event-model-account.emod
emod diagram docs/event-model-account.emod --format svg --specs -o docs/event-model-account.svg
```

`emod validate` and `emod lint` report no findings for this model.

## Actors

| Actor | Type | Description |
| --- | --- | --- |
| Operator | Human | A person who runs the command line |

## Storyboard

The command line is the only touchpoint. Each command prints its result as JSON.

| Command | Inputs | Output |
| --- | --- | --- |
| `open` | `--id`, `--owner` | The account ID |
| `credit` | `--id`, `--amount`, `--ref` | The account ID and the reference |
| `show` | `--id` | The owner, the balance, the time of the open, and each credit |

The operator reads the account with `show` before a credit. The model shows this as a `reads` from the
`credit command` trigger to `AccountView`.

## Business Process: Keep an Account

### Timeline

```
open command   -> OpenAccount   -> AccountOpened
credit command -> CreditAccount -> AccountCredited   (once for each reference)
AccountOpened, AccountCredited  -> AccountView       (show command)
```

### Events

| Event | Description | Key data |
| --- | --- | --- |
| `AccountOpened` | An account was opened for an owner | `accountId`, `owner`, `openedAt` |
| `AccountCredited` | Money was credited to an account for a reference | `accountId`, `amount`, `reference`, `creditedAt` |

### Commands

| Command | Actor | Produces | Refused when | Required data |
| --- | --- | --- | --- | --- |
| `OpenAccount` | Operator | `AccountOpened` | `HasOwner` | `accountId`: `--id`. `owner`: `--owner` |
| `CreditAccount` | Operator | `AccountCredited` | `AccountExists`, `PositiveAmount` | `accountId`: `--id`. `amount`: `--amount`. `reference`: `--ref` |

The runner adds the time (`openedAt`, `creditedAt`) from its clock.

### Rules

| Rule | Statement | Code |
| --- | --- | --- |
| `HasOwner` | An account has an owner | `open.ErrNoOwner` |
| `AccountExists` | A credit goes only to an account that was opened | `credit.ErrNotOpen` |
| `PositiveAmount` | A credit is more than 0 | `credit.ErrInvalidAmount` |

A rule refuses the command, and the command exits with code 1.

Two repeated commands record nothing and succeed. The specs show them with `then = []`:

- `OpenAccount` for an account that is open
- `CreditAccount` with a reference that the account has

### Views

| View | Purpose | Backs | Data lineage |
| --- | --- | --- | --- |
| `AccountView` | One account with its balance and its credits | The `show` command | `accountId`, `owner`, `openedAt` ← `AccountOpened`. `balance` ← the sum of `AccountCredited.amount`. `credits` ← each `AccountCredited` |

### Message Processing Patterns

| # | From | To | Pattern | Component |
| --- | --- | --- | --- | --- |
| 1 | `open command` | `OpenAccount` | UI trigger | `internal/cli` |
| 2 | `OpenAccount` | `AccountOpened` | Aggregate | `internal/use_cases/account/open` |
| 3 | `credit command` | `CreditAccount` | UI trigger | `internal/cli` |
| 4 | `CreditAccount` | `AccountCredited` | Aggregate | `internal/use_cases/account/credit` |
| 5 | `AccountOpened`, `AccountCredited` | `AccountView` | Projection, folded when the command runs | `internal/use_cases/account/show` |

The aggregate is the state in `internal/account`, which each decide function reads.

## Autonomous Components

| Component | Owned events | Owned commands | Consumes events from |
| --- | --- | --- | --- |
| Accounts | `AccountOpened`, `AccountCredited` | `OpenAccount`, `CreditAccount` | None |

## Implementation Slices

| # | Slice | Events | Commands | Views | Package |
| --- | --- | --- | --- | --- | --- |
| 1 | Open Account | `AccountOpened` | `OpenAccount` | | `internal/use_cases/account/open` |
| 2 | Credit Account | `AccountCredited` | `CreditAccount` | | `internal/use_cases/account/credit` |
| 3 | Show Account | | | `AccountView` | `internal/use_cases/account/show` |

## What the Model Leaves Out

- The `events` command prints the raw events of any stream. It reads no business view, so it is not in
  the model.
- The exit codes, the schema versions of the events, and the SQLite store are not part of the domain.
  The [README](../README.md) tells about them.
