CREATE TABLE IF NOT EXISTS checkpoints (
    name           TEXT PRIMARY KEY,
    transaction_id XID8 NOT NULL,
    sequence       BIGINT NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
