-- Append-only event log. The unique constraint on (stream_id, version) is what
-- makes optimistic concurrency work: two writers that loaded the same version
-- both try to insert the same next version, and exactly one succeeds.
CREATE TABLE IF NOT EXISTS events (
  global_seq     BIGSERIAL   PRIMARY KEY,
  stream_id      TEXT        NOT NULL,
  version        INTEGER     NOT NULL,
  type           TEXT        NOT NULL,
  schema_version INTEGER     NOT NULL DEFAULT 1,
  data           JSONB       NOT NULL,
  recorded_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE (stream_id, version)
);

CREATE INDEX IF NOT EXISTS events_by_stream ON events (stream_id, version);

-- Events are facts. Rejecting updates and deletes in the database means a bug
-- in application code cannot quietly rewrite history.
CREATE OR REPLACE FUNCTION events_are_append_only() RETURNS trigger AS $$
BEGIN
  RAISE EXCEPTION 'events are append-only';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS events_no_update ON events;
CREATE TRIGGER events_no_update BEFORE UPDATE ON events
  FOR EACH ROW EXECUTE FUNCTION events_are_append_only();

DROP TRIGGER IF EXISTS events_no_delete ON events;
CREATE TRIGGER events_no_delete BEFORE DELETE ON events
  FOR EACH ROW EXECUTE FUNCTION events_are_append_only();
