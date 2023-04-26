-- +migrate Up
CREATE TABLE result_metadata_type (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    value text NOT NULL UNIQUE
);
CREATE TABLE result_metadata (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    queue_item_id uuid NOT NULL,
    type_id INT NOT NULL,
    value text NOT NULL,
    FOREIGN KEY (type_id) REFERENCES result_metadata_type(id) ON DELETE CASCADE,
    FOREIGN KEY (queue_item_id) REFERENCES queue_item(id) ON DELETE CASCADE
);
CREATE INDEX result_metadata_type_index ON result_metadata (value);
CREATE INDEX result_metadata_queue_item_id_index ON result_metadata (queue_item_id);