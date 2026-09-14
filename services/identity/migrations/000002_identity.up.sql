CREATE EXTENSION IF NOT EXISTS citext;

CREATE TABLE roles (
    id          BIGSERIAL PRIMARY KEY,
    name        TEXT        NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE permissions (
    id    BIGSERIAL PRIMARY KEY,
    name  TEXT NOT NULL UNIQUE
);

CREATE TABLE role_permissions (
    role_id        BIGINT NOT NULL REFERENCES roles(id)       ON DELETE CASCADE,
    permission_id  BIGINT NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

CREATE TABLE staff (
    id                    BIGSERIAL PRIMARY KEY,
    email                 CITEXT      NOT NULL UNIQUE,
    first_name            TEXT        NOT NULL,
    last_name             TEXT        NOT NULL,
    password_hash         TEXT        NOT NULL,
    role_id               BIGINT      NOT NULL REFERENCES roles(id),
    is_active             BOOLEAN     NOT NULL DEFAULT true,
    must_change_password  BOOLEAN     NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    token_hash  BYTEA       PRIMARY KEY,
    staff_id    BIGINT      NOT NULL REFERENCES staff(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    revoked_at  TIMESTAMPTZ
);

CREATE INDEX sessions_staff_id_idx ON sessions (staff_id);

INSERT INTO permissions (name) VALUES
    ('view_staff'), ('edit_staff'),
    ('view_roles'), ('edit_roles'),
    ('view_products'), ('edit_products'),
    ('view_orders'), ('edit_orders');

INSERT INTO roles (name) VALUES ('admin');

INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r CROSS JOIN permissions p WHERE r.name = 'admin';
