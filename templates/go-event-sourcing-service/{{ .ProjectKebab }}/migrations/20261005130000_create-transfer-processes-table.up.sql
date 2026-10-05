CREATE TABLE IF NOT EXISTS transfer_processes (
    id             UUID PRIMARY KEY,
    from_account   UUID NOT NULL,
    to_account     UUID NOT NULL,
    amount         BIGINT NOT NULL,
    state          TEXT NOT NULL,
    reason         TEXT NOT NULL DEFAULT '',
    stream_version BIGINT NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
