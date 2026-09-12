-- PostgreSQL 15+ no longer grants PUBLIC default CREATE on schema public
-- (confirmed against the postgres:17 image this stack pins: pg_namespace
-- .nspacl for a fresh database carries no CREATE grant to PUBLIC). There is
-- therefore no privilege to revoke here; only the comment marks this schema
-- as owned by this service.
COMMENT ON SCHEMA public IS 'catalog service';
