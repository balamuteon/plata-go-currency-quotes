-- +goose Up
-- +goose StatementBegin
CREATE TABLE quote_updates (
    id UUID PRIMARY KEY,
    pair TEXT NOT NULL,
    idempotency_key VARCHAR(128) NOT NULL UNIQUE,
    status TEXT NOT NULL DEFAULT 'queued',
    price NUMERIC(24, 12),
    attempts SMALLINT NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    lease_until TIMESTAMPTZ,
    lease_token UUID,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX quote_updates_queued_idx
    ON quote_updates (next_attempt_at, created_at)
    WHERE status = 'queued';

CREATE INDEX quote_updates_expired_lease_idx
    ON quote_updates (lease_until, created_at)
    WHERE status = 'processing';

CREATE INDEX quote_updates_latest_quote_idx
    ON quote_updates (pair, updated_at DESC, created_at DESC)
    WHERE status = 'succeeded';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS quote_updates;
-- +goose StatementEnd
