DROP INDEX IF EXISTS order_events_order_idx;
DROP TABLE IF EXISTS order_events;

DROP INDEX IF EXISTS order_items_order_idx;
DROP TABLE IF EXISTS order_items;

DROP INDEX IF EXISTS orders_status_idx;
DROP INDEX IF EXISTS orders_customer_idx;
DROP TABLE IF EXISTS orders;

DROP SEQUENCE IF EXISTS order_number_seq;
DROP TYPE IF EXISTS order_status;

DROP TABLE IF EXISTS customers;

DROP EXTENSION IF EXISTS citext;
