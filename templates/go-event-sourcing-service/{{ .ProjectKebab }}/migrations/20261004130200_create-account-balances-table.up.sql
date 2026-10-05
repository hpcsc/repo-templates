CREATE TABLE IF NOT EXISTS account_balances (
    id         UUID PRIMARY KEY,
    owner      TEXT NOT NULL,
    balance    BIGINT NOT NULL,
    version    BIGINT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
