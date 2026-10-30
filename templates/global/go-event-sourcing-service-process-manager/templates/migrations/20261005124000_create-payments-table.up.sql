CREATE TABLE IF NOT EXISTS payments (
    id                UUID PRIMARY KEY,
    account_id        UUID NOT NULL,
    creditor_iban     TEXT NOT NULL,
    amount            BIGINT NOT NULL,
    status            TEXT NOT NULL,
    reason            TEXT NOT NULL DEFAULT '',
    network_reference TEXT NOT NULL DEFAULT '',
    version           BIGINT NOT NULL,
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
