# {{ .ProjectKebab }}

A base for an event-sourced Go application that keeps its events in Postgres. Consumers read the events
with a loop that wakes on a `LISTEN/NOTIFY` signal, so the application needs no other event store.

The repository contains:

- **An event store in one Postgres table:** optimistic concurrency for each stream, and a global order
  that a late commit cannot break.
- **A subscription with a read loop and a signal:** no catch-up mode and no live mode, and no event lost
  at a switch.
- **One consumer loop for every kind of consumer:**
  - Projections save each batch and its checkpoint in one transaction.
  - Reactions start at the end of the store and save the checkpoint after each event.
- **An example domain:** a bank account with an HTTP API, a balance projection, and a welcome bonus
  reaction that sends a command.

[docs/event-subscription.md](docs/event-subscription.md) explains the design: the write path, the read
loop, and how to choose the kind of consumer.

## Requirements

- [mise](https://mise.jdx.dev/) for Go, `task`, `migrate`, `gotestsum`, `shellcheck` and `govulncheck`
- Docker, `jq` and `tmuxp`

```sh
mise install
```

## Run the Example

```sh
task run
```

This starts Postgres on port 45432, runs the migrations, and opens a tmux window with the `api` and
`catchup` processes.

In another terminal:

```sh
task api:open-account -- alice          # returns the new account ID
export ID=<the account ID>
task api:deposit -- 500
task api:withdraw -- 200
task api:get-account                    # balance 1300: 500 - 200, plus the welcome bonus of 1000
task api:close-account                  # a closed account rejects deposits and withdrawals
```

The balance comes from the `account_balances` read model, so it can show the change a short time after
the command.

Other tasks:

| Task | What it does |
| --- | --- |
| `task db:cli` | Opens `psql` on the local database. |
| `task db:query -- 'select * from events'` | Runs one query on the local database. |
| `task remove-dependencies` | Stops the local database and deletes its container. |
| `task migrate:create -- <name>` | Creates a new migration in `migrations/`. |

## Test

```sh
task test:unit
task test:integration   # starts a separate Postgres on port 45433
```

Unit tests have the `unit` build tag. Integration tests have the `integration` build tag and use the
values in `.env.integration`.

## Replace the Example

Replace the account example with your own domain:

- `internal/usecase/account`: the aggregate, the events, the commands, and one package for each use case
- `internal/usecase/register.go`: the registration of the event types and the command handlers
- `internal/api/usecase/account`: the HTTP handler
- the migrations of the read model: `account_balances` and its status column
- the consumers in `cmd/catchup/main.go`, and the routes in `internal/api/usecase/register.go`

## Layout

| Path | Contents |
| --- | --- |
| `cmd/api` | The HTTP API. It sends commands and reads the read models. |
| `cmd/catchup` | The process that runs the projections and the reactions. |
| `internal/domain` | Events, aggregates, commands and their bus. |
| `internal/domain/event/stream` | The write path: append to a stream, read a stream. |
| `internal/domain/event/subscription` | The read loop and the listener. |
| `internal/domain/consumer` | The loop for every consumer. |
| `internal/domain/projection`, `internal/domain/reaction` | The two kinds of consumer. |
| `internal/domain/checkpoint/store` | Checkpoints in Postgres. |
| `internal/usecase/<domain>` | The aggregate of a domain, its events, its commands and its repository. |
| `internal/usecase/<domain>/<use case>` | One use case: its command handler, and its projection or reaction. |
| `internal/usecase/register.go` | The registration of every event type and every command handler. |
| `internal/api` | The HTTP server, routes, middleware and handlers. |
| `migrations` | The schema. The `migrate` container applies it at start. |

## Add an Event

1. Add a struct for the payload to the `event` package of the domain, for example `MoneyDeposited` in
   `internal/usecase/account/event`.
2. Give the struct an `EventType` method that returns the name of the event type.
3. Register the struct in the `RegisterEvents` function of the domain, for example in
   `internal/usecase/account/events.go`.

The store saves the name of the event type with each event. When the store reads the event, the registry
finds the struct from that name. You can rename the struct, but do not change the name of the event type
after the store has an event of that type.

## Change an Event

Each saved event keeps the schema version of its payload struct. A registration with no schema version
has version 1.

| Change | What to do |
| --- | --- |
| Add a field that an old build can ignore | Change the struct only. |
| Add a field that an old build must not ignore | Change the struct. Increase the schema version, and pass `nil` as the upcast. |
| Rename or remove a field, or change what a field means | Change the struct. Increase the schema version, and give an upcast. |

```go
reg.Register[event.MoneyDeposited](registry.WithSchemaVersion(2, upcastMoneyDeposited))
```

The upcast gets the schema version and the data of an older event, and returns the data in the current
shape. The registry calls it before it parses the data.

When an old build reads an event with a newer schema version, the registry returns
`registry.ErrNewerSchemaVersion`, and the read loop stops. Therefore an old `catchup` stops during a
deploy, and does not write wrong data to a read model. The `catchup` of the new build reads the event.

## Add a Command

1. Add a method to the aggregate. The method checks the rule and calls the private `record` method of the
   aggregate with the new event, for example `domain.NewEvent(&event.MoneyDeposited{...})`. `Apply`
   changes the state from that event.
2. Add a payload type for the command, for example `Deposit` in `internal/usecase/account/command`.
3. Add a package for the use case, for example `internal/usecase/account/deposit`. Give it a handler type
   with a `Handle(ctx, domain.Command[Deposit]) error` method. The handler loads the aggregate from its
   repository, calls the method, and saves.
4. Give the package a `Register` function that calls `bus.Register[Deposit](handler{...})`, and call it
   from `internal/usecase/register.go`.

Send the command with `bus.Dispatch(ctx, domain.NewCommand(Deposit{...}))`. The bus finds the handler from
the payload type, and calls it again on a concurrency conflict.

## Add a Consumer

- **A projection:** implement `projection.Interface`. Write to the read model through the `pgx.Tx` that
  `Handle` gets. Add the table in a migration. Add `projection.NewProjector(...)` to the consumers in
  `cmd/catchup/main.go`.
- **A process manager:** a projection that keeps its state, and appends the requests it decides as
  events in the same transaction, plus a reaction with `reaction.StartAtBeginning()` that sends each
  request.
- **A reaction:** implement `reaction.Interface`, or use `external.NewHttpReaction`. Make the effect
  safe to repeat. A message to another service is a contract, not an event, as
  [docs/event-subscription.md](docs/event-subscription.md) tells. A reaction that sends a command takes the command ID from `dispatch.CommandIDFor`, as
  `welcomebonus.Reaction` does. Add `reaction.NewReactor(...)` to the consumers in `cmd/catchup/main.go`.

The name of each consumer is the key of its checkpoint. Do not rename a consumer after it runs in
production: a projection with a new name rebuilds from the beginning, and a reaction with a new name
starts again at the end of the store.

{{- if .Scaffold.ProcessManager }}

## Process Manager Example

A money transfer between two accounts runs as a process manager. A failed credit refunds the debit.
[docs/process-manager-example.md](docs/process-manager-example.md) explains the design.

```sh
export FROM=<an account ID with money> TO=<another account ID>
task api:transfer -- 300                # returns the new transfer ID
export TRANSFER=<the transfer ID>
task api:get-transfer                   # completed, or failed with the reason
```

A transfer to a closed account fails, and the process refunds the source account.

To replace it with your own process, change these:

- `internal/usecase/transfer`: the aggregate that starts a transfer, its events, its commands, the start
  use case and the process manager
- `internal/usecase/account/transfers`, and the `transfer` files in `internal/usecase/account`: the account
  side of a transfer
- `internal/api/usecase/transfer`: the HTTP handler
- the migration of the `transfer_processes` table
{{- end }}
