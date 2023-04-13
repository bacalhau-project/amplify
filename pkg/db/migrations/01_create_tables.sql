-- +migrate Up
CREATE TABLE queue_item (
    id uuid PRIMARY KEY,
    inputs text[],
    created_at timestamp NOT NULL
);

-- +migrate Up
CREATE TABLE node (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    queue_item_id uuid NOT NULL,
    name text NOT NULL
    -- TODO: add foreign key constraint, need to change the order of how items are created
);

-- +migrate Up
CREATE TABLE edge (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    parent_id INT NOT NULL,
    child_id INT NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES node(id) ON DELETE CASCADE,
    FOREIGN KEY (child_id) REFERENCES node(id) ON DELETE CASCADE
);

-- +migrate Up
CREATE TABLE result (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    node_id INT NOT NULL,
    ts timestamp DEFAULT CURRENT_TIMESTAMP,
    execution_id text,
    stdout text,
    stderr text,
    skipped boolean,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

-- +migrate Up
CREATE TABLE status (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    ts timestamp DEFAULT CURRENT_TIMESTAMP,
    node_id INT NOT NULL,
    submitted timestamp NOT NULL,
    status text NOT NULL,
    started timestamp,
    ended timestamp,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

-- +migrate Up
CREATE TYPE io_spec_type AS ENUM ('input', 'output');

-- +migrate Up
CREATE TABLE io_spec (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    node_id INT NOT NULL,
    type io_spec_type NOT NULL,
    node_name text NOT NULL,
    input_id text NOT NULL,
    root boolean NOT NULL,
    value text,
    path text,
    context text,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);
