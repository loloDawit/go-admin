-- A superuser role would make every grant below decorative.
CREATE ROLE identity_user WITH LOGIN PASSWORD 'dev_only_identity' NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE catalog_user  WITH LOGIN PASSWORD 'dev_only_catalog'  NOSUPERUSER NOCREATEDB NOCREATEROLE;
CREATE ROLE orders_user   WITH LOGIN PASSWORD 'dev_only_orders'   NOSUPERUSER NOCREATEDB NOCREATEROLE;

CREATE DATABASE identity_db OWNER identity_user;
CREATE DATABASE catalog_db  OWNER catalog_user;
CREATE DATABASE orders_db   OWNER orders_user;

-- PUBLIC holds CONNECT on every database by default, which would let any role
-- reach a sibling database regardless of the grants above.
REVOKE CONNECT ON DATABASE identity_db FROM PUBLIC;
REVOKE CONNECT ON DATABASE catalog_db  FROM PUBLIC;
REVOKE CONNECT ON DATABASE orders_db   FROM PUBLIC;

GRANT CONNECT ON DATABASE identity_db TO identity_user;
GRANT CONNECT ON DATABASE catalog_db  TO catalog_user;
GRANT CONNECT ON DATABASE orders_db   TO orders_user;

-- Without this, any service role can still connect to postgres/template1
-- and enumerate pg_database/pg_roles for every service's name and owner.
REVOKE CONNECT ON DATABASE template1, postgres FROM PUBLIC;
