-- PUBLIC may create objects in the public schema by default.
REVOKE CREATE ON SCHEMA public FROM PUBLIC;

COMMENT ON SCHEMA public IS 'orders service';
