-- +migrate Up
CREATE TABLE queue_item (
    id uuid PRIMARY KEY,
    inputs text[],
    created_at timestamp NOT NULL
);

-- +migrate Up
CREATE TABLE node (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    execution_id uuid REFERENCES queue_item(id) ON DELETE CASCADE,
    name text,
    children INT[],
    parents INT[],
    inputs text[]
);

-- +migrate Up
CREATE TABLE result (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    node_id INT NOT NULL,
    ts timestamp DEFAULT CURRENT_TIMESTAMP,
    stdout text,
    stderr text,
    skipped boolean,
    outputs text[],
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

CREATE TABLE status (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    node_id INT NOT NULL,
    ts timestamp DEFAULT CURRENT_TIMESTAMP,
    external_id uuid,
    status text,
    submitted timestamp NOT NULL,
    started timestamp,
    ended timestamp,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);


