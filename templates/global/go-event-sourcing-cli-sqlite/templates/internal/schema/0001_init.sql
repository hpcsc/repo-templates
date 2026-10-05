-- One stream for each aggregate. UNIQUE(stream_id, seq) turns a stale append into a conflict.
CREATE TABLE events (
  global_seq INTEGER PRIMARY KEY AUTOINCREMENT,
  stream_id  TEXT    NOT NULL,
  seq        INTEGER NOT NULL,
  at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
  kind       TEXT    NOT NULL,
  v          INTEGER NOT NULL,
  data       TEXT    NOT NULL,
  UNIQUE (stream_id, seq)
);

CREATE TRIGGER events_no_update BEFORE UPDATE ON events
  BEGIN SELECT RAISE(ABORT, 'events are append-only'); END;
CREATE TRIGGER events_no_delete BEFORE DELETE ON events
  BEGIN SELECT RAISE(ABORT, 'events are append-only'); END;
