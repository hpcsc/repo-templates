CREATE TABLE IF NOT EXISTS events (
    sequence       BIGSERIAL PRIMARY KEY,
    transaction_id XID8 NOT NULL DEFAULT pg_current_xact_id(),
    id             UUID NOT NULL UNIQUE,
    stream_id      TEXT NOT NULL,
    version        BIGINT NOT NULL,
    type           TEXT NOT NULL,
    data           JSONB NOT NULL,
    metadata       JSONB,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT events_stream_version UNIQUE (stream_id, version)
);

CREATE INDEX IF NOT EXISTS events_position ON events (transaction_id, sequence);
CREATE INDEX IF NOT EXISTS events_type_position ON events (type, transaction_id, sequence);
