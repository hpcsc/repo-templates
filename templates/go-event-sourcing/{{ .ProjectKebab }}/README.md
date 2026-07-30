# {{ .ProjectKebab }}

An event-sourced service with a Postgres event store.

```sh
task db:up
task run -- open   -id acc-1 -owner "Ada Lovelace"
task run -- credit -id acc-1 -amount 500 -ref invoice-7
task run -- show   -id acc-1

task test:unit    # aggregates and use cases, in memory, no database
task test:e2e     # the same use cases against postgres, started for you
```

## What to keep and what to delete

`internal/es` is the foundation. Everything else under `internal/` is an
example account slice, kept deliberately thin so you can read it in one sitting
and then delete it.

```
internal/
  es/                          KEEP — the foundation
    command.go  event.go       es.Command, es.Event
    store.go                   es.Store, es.ErrVersionConflict
    aggregate.go               es.Root, es.Repository (retries on conflict)
    codec.go                   the event registry: encode, decode, versioning
    store/postgres.go          the durable store
    store/memory.go            an in-memory store for unit tests
    schema/                    embedded DDL, applied on open
  event/                       DELETE — one file per event
  command/                     DELETE — one file per command
  aggregate/                   DELETE — one file per aggregate
  use_cases/account/           DELETE — one folder per use case
  app/                         KEEP the shape, replace the contents
```

## The loop

Every write goes the same way, which is why it is worth writing down once:

```
use case Runner  →  es.Repository.Update  →  load stream
                                          →  rebuild aggregate via Apply
                                          →  aggregate decides, returns events
                                          →  append at the loaded version
```

The aggregate never writes and never reads a store. It folds its own history in
`Apply`, then answers a command with events. That is the whole contract:

```go
func (a *Account) Apply(e es.Event)
func (a *Account) OpenAccount(cmd command.OpenAccount) ([]es.Event, error)
```

Three things follow from it, and the example slice shows each:

**Deciding nothing is a valid answer.** Returning no events writes nothing, so
an aggregate reports "already done" without an error. Opening an account twice
and replaying a credit reference both take that path.

**Concurrency resolves itself.** Two writers that loaded the same version both
try to append the same next version, and the unique constraint on
`(stream_id, version)` means one loses. `Repository.Update` retries it, because
the loser is stale rather than wrong. The e2e suite runs eight concurrent
credits and expects all eight to land.

Retries are spread with a jittered backoff rather than fired immediately: each
round of the race lets only one writer past, so a herd retrying in lockstep
just collides again and the writers at the back exhaust their budget. That also
means the attempt count has to exceed the contention you actually expect —
`es.DefaultRetryAttempts` is the number to raise if a hot stream starts
surfacing `es.ErrVersionConflict`. A stream that needs a lot of them is usually
telling you the aggregate is too coarse.

**The read side does not use aggregates.** Aggregates exist to decide.
`use_cases/account/show` folds the stream into a view instead, which is fine
while streams are short — swap in a projection table later without changing the
signature.

## Adding a use case

A use case is a folder with a `Runner` and a `Report`, not a method on a
service that grows forever:

```
internal/use_cases/<area>/<verb>/runner.go
```

Take dependencies in `New`, take the request in `Run`, return a `Report`. Wire
it in `internal/app`, which is the only package that knows how the others fit
together — and which tests build through too, so there is no second wiring that
can drift.

## Adding an event

One file per event in `internal/event`, registered from an `init`:

```go
type AccountOpened struct{ ... }

func (AccountOpened) EventName() string { return "AccountOpened" }

func init() { es.Register[AccountOpened]() }
```

`EventName` is the name rows are stored under. Rename the Go type freely;
changing that string orphans every event already written.

When a stored shape has to change, prefer a new event type. If you genuinely
must reshape an existing one, `es.RegisterVersioned` hands your decoder the
schema version each row was written with so it can upgrade old rows on read.

## Schema changes

`internal/es/schema/*.sql` is applied in filename order on open, once each,
recorded with a hash of its contents. Editing a file that has already been
applied is refused — add `0002_….sql` instead. That keeps two databases with
the same name from quietly having different shapes.

The event table rejects `UPDATE` and `DELETE` with a trigger, so an application
bug cannot rewrite history.

## Configuration

`DATABASE_URL`, defaulting to
`postgres://postgres:postgres@localhost:5432/{{ .ProjectSnake }}?sslmode=disable`.

The e2e suite uses `{{ .ProjectSnake }}_test`, created by `task db:create-test`,
which `task test:e2e` runs for you.
