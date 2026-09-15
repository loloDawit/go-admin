CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE customers (
    id          BIGSERIAL PRIMARY KEY,
    email       CITEXT      NOT NULL UNIQUE,
    name        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TYPE order_status AS ENUM (
    'pending', 'paid', 'packed', 'shipped', 'delivered', 'cancelled', 'refunded'
);

CREATE SEQUENCE order_number_seq;

CREATE TABLE orders (
    id            BIGSERIAL PRIMARY KEY,
    number        TEXT         NOT NULL UNIQUE,
    customer_id   BIGINT       NOT NULL REFERENCES customers(id),
    status        order_status NOT NULL DEFAULT 'pending',
    total_minor   BIGINT       NOT NULL CHECK (total_minor >= 0),
    currency      CHAR(3)      NOT NULL,
    placed_at     TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX orders_customer_idx ON orders (customer_id, placed_at DESC);
CREATE INDEX orders_status_idx   ON orders (status);

-- product_id names a row in Catalog's own database, so no foreign key here.
CREATE TABLE order_items (
    id                  BIGSERIAL PRIMARY KEY,
    order_id            BIGINT  NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id          BIGINT  NOT NULL,
    title_snapshot      TEXT    NOT NULL,
    unit_price_minor    BIGINT  NOT NULL CHECK (unit_price_minor >= 0),
    currency            CHAR(3) NOT NULL,
    quantity            INT     NOT NULL CHECK (quantity > 0),
    line_total_minor    BIGINT  NOT NULL CHECK (line_total_minor >= 0)
);

CREATE INDEX order_items_order_idx ON order_items (order_id);

CREATE TABLE order_events (
    id          BIGSERIAL    PRIMARY KEY,
    order_id    BIGINT       NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    from_status order_status,
    to_status   order_status NOT NULL,
    actor_id    TEXT         NOT NULL,
    reason      TEXT         NOT NULL DEFAULT '',
    at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX order_events_order_idx ON order_events (order_id, at);
