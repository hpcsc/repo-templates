# {{ .ProjectKebab }}

An event-sourced command-line application. Each command reads the stream that it needs, decides, and
appends the new events. A command that only reads folds the stream into a report, so the application has
no read model to keep up to date.

```sh
task run -- open   --id acc-1 --owner "Ada Lovelace"
task run -- credit --id acc-1 --amount 500 --ref invoice-7
task run -- show   --id acc-1
task run -- events --stream account-acc-1

task test:unit
```

The in-memory store keeps the events only while one command runs, so each command starts with an empty
store. The SQLite store keeps them between commands. The template adds it when you answer yes to its
question, and the `go-event-sourcing-cli-sqlite` template scaffold adds it to an existing project.

Each command prints its result as JSON on stdout. An error prints `{"error": ..., "code": ...}` on stderr,
and the exit code tells the kind of error:

| Exit code | Code | Meaning |
| --- | --- | --- |
| 0 | | The command succeeded. |
| 1 | `refused` | A rule refused the command, for example a credit to an account that is not open. |
| 2 | `usage`, `stream-version`, `internal` | The command line is wrong, or the program failed. |
| 3 | `conflict` | Other processes changed the stream twice while this command decided. |

## Layout

| Path | Contents |
| --- | --- |
| `cmd/{{ .ProjectKebab }}` | The entry point. |
| `internal/cli` | The commands, their flags, and the map from errors to exit codes. |
| `internal/app` | The composition root: it opens the store and builds every use case. |
| `internal/es` | `Event` and `Record`, which every package uses. |
| `internal/event` | The events, with a `Kind` for each, and the codec with schema versions. |
| `internal/stream` | `Store`, `Fold`, and `Update`, which loads, folds, decides and appends. |
| `internal/store` | The stores. |
| `internal/account` | The state of an account, folded from its stream. |
| `internal/use_cases/<area>/<verb>` | One use case: a runner and its decide function. |
| `internal/exit` | The exit codes and the JSON output of an error. |

## The Write Path

```
use case runner  ->  stream.Update  ->  load the stream
                                    ->  fold it into the state
                                    ->  decide: return the new events
                                    ->  append at the loaded seq
```

The decide function reads the state and returns events. It does not write. When it returns no events,
nothing is recorded, so a repeated command is not an error. A second open of the same account takes this
path, and so does a credit with a reference that the account already has.

When another process appends to the stream between the load and the append, `Update` loads the stream
again, decides again and tries once more. A second conflict returns `stream.ErrVersionConflict`, and the
command exits with code 3.

## Add a Use Case

1. Add the event to `internal/event`: a struct with a `Kind` method that returns its name, and a
   `register` call in the `init` function of that file.
2. Add a field or a case to the state in `internal/account`, or add a state for a new aggregate.
3. Add a package `internal/use_cases/<area>/<verb>` with a `Runner` and a `decide` function. The runner
   calls `stream.Update` with the decide function.
4. Add the runner to `internal/app`, and a command to `internal/cli`.

The store saves the kind with each event. You can rename the struct, but do not change the kind after a
store has an event of that kind.

## Change an Event

Each saved event keeps the schema version of its kind. `register` gives version 1.

| Change | What to do |
| --- | --- |
| Add a field that an old build can ignore | Change the struct only. |
| Add a field that an old build must not ignore | Register the kind with `registerVersioned` and the next version. The decoder reads older data as it is. |
| Rename or remove a field, or change what a field means | Register the kind with `registerVersioned` and the next version. The decoder changes older data to the current shape. |

When an old build reads an event with a newer schema version, `Decode` returns a `NewerSchemaError`, and
the command exits with code 2 and the code `stream-version`.

{{- if .Scaffold.SQLite }}

## SQLite Store

The SQLite store keeps the events in one file. The `--db` flag names the file. The default is
`$XDG_DATA_HOME/{{ .ProjectKebab }}/events.db`, or `~/.local/share/{{ .ProjectKebab }}/events.db`. The store uses `modernc.org/sqlite`, which
needs no C compiler, so `CGO_ENABLED=0` still builds a static binary.

- The store applies the files in `internal/schema` in name order when it opens, and records the hash of
  each file. It refuses to open when an applied file changed, so add a new file for each change.
- It refuses to open a store that has a schema file that this build does not have, because a newer build
  wrote that store.
- Triggers refuse `UPDATE` and `DELETE` on the `events` table.
- `UNIQUE(stream_id, seq)` turns a stale append into `stream.ErrVersionConflict`.
{{- end }}
