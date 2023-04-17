-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY result_node_id ON result (node_id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY edge_parent_id ON edge (parent_id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY edge_child_id ON edge (child_id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY io_spec_node_id ON io_spec (node_id);

-- +migrate Up notransaction
CREATE INDEX CONCURRENTLY status_node_id ON status (node_id);