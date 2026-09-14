CREATE EXTENSION IF NOT EXISTS citext;

CREATE TYPE product_status AS ENUM ('draft', 'active', 'archived');

-- to_tsvector's two-argument form is immutable and so may back a generated
-- column; the single-argument form is not and is rejected here.
CREATE TABLE products (
    id           BIGSERIAL PRIMARY KEY,
    sku          CITEXT         NOT NULL UNIQUE,
    title        TEXT           NOT NULL,
    description  TEXT           NOT NULL DEFAULT '',
    price_minor  BIGINT         NOT NULL CHECK (price_minor >= 0),
    currency     CHAR(3)        NOT NULL,
    status       product_status NOT NULL DEFAULT 'draft',
    created_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ    NOT NULL DEFAULT now(),
    search       tsvector GENERATED ALWAYS AS (
        setweight(to_tsvector('english', title), 'A') ||
        setweight(to_tsvector('english', description), 'B')
    ) STORED
);

CREATE INDEX products_search_idx ON products USING GIN (search);
CREATE INDEX products_status_idx ON products (status);

CREATE TABLE product_images (
    id          BIGSERIAL PRIMARY KEY,
    product_id  BIGINT      NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    object_key  TEXT        NOT NULL UNIQUE,
    alt         TEXT        NOT NULL DEFAULT '',
    position    INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX product_images_product_idx ON product_images (product_id, position);
