CREATE TABLE queue_item (
    id uuid PRIMARY KEY,
    inputs text[],
    created_at timestamp NOT NULL
);

CREATE TABLE node (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    queue_item_id uuid NOT NULL,
    name text NOT NULL
    -- TODO: add foreign key constraint, need to change the order of how items are created
);

CREATE TABLE edge (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    parent_id INT NOT NULL,
    child_id INT NOT NULL,
    FOREIGN KEY (parent_id) REFERENCES node(id) ON DELETE CASCADE,
    FOREIGN KEY (child_id) REFERENCES node(id) ON DELETE CASCADE
);

CREATE TABLE result (
    id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
    ts timestamp DEFAULT CURRENT_TIMESTAMP,
    node_id INT NOT NULL,
    execution_id text,
    stdout text,
    stderr text,
    skipped boolean,
    FOREIGN KEY (node_id) REFERENCES node(id) ON DELETE CASCADE
);

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

CREATE TYPE io_spec_type AS ENUM ('input', 'output');

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
