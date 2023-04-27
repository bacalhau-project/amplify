-- +migrate Up
ALTER TABLE queue_item ADD UNIQUE (id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY node_queue_item_id ON node (queue_item_id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY io_spec_type ON io_spec (type);
