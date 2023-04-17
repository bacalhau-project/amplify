-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY queue_item_created_at ON queue_item (created_at);