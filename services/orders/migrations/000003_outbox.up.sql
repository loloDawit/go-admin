CREATE TABLE outbox (
    id           BIGSERIAL   PRIMARY KEY,
    event_id     TEXT        NOT NULL UNIQUE,
    subject      TEXT        NOT NULL,
    payload      JSONB       NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    published_at TIMESTAMPTZ
);

-- The publisher's whole query plan: it scans only what it has not sent.
CREATE INDEX outbox_unpublished_idx ON outbox (id) WHERE published_at IS NULL;
