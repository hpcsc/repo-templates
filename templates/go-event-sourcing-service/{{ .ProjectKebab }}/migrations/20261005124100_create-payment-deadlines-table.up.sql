CREATE TABLE IF NOT EXISTS payment_deadlines (
    payment_id UUID PRIMARY KEY,
    stage      TEXT NOT NULL,
    attempt    INT NOT NULL,
    deadline   TIMESTAMPTZ NOT NULL,
    version    BIGINT NOT NULL
);

CREATE INDEX IF NOT EXISTS payment_deadlines_deadline ON payment_deadlines (deadline);
