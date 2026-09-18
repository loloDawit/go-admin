-- Inserted in the same transaction as the projection update, so a redelivery
-- after a consumer crash conflicts here and the whole transaction is a no-op.
CREATE TABLE processed_events (
    event_id     TEXT        PRIMARY KEY,
    processed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE revenue_by_day (
    day              DATE   NOT NULL,
    currency         TEXT   NOT NULL,
    placed_count     INT    NOT NULL DEFAULT 0,
    paid_count       INT    NOT NULL DEFAULT 0,
    recognised_minor BIGINT NOT NULL DEFAULT 0,
    refunded_minor   BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (day, currency)
);

-- A refund reverses revenue on the day it was recognised, not the day of the
-- refund, so "revenue recognised for day X" stays true after the fact. This is
-- the only thing that remembers which day that was.
CREATE TABLE recognised_orders (
    order_id     BIGINT PRIMARY KEY,
    day          DATE   NOT NULL,
    currency     TEXT   NOT NULL,
    amount_minor BIGINT NOT NULL
);
