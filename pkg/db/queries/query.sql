-- name: CreateQueueItem :exec
INSERT INTO queue_item (id, inputs, created_at)
VALUES ($1, $2, $3);

-- name: ListQueueItems :many
SELECT * 
FROM queue_item
WHERE sqlc.arg(createdAfter) <= created_at AND created_at < sqlc.arg(createdBefore)
ORDER BY CASE
    WHEN NOT @reverse::boolean THEN created_at
END DESC, CASE
    WHEN @reverse::boolean THEN created_at
END ASC
LIMIT $1;

-- name: GetQueueItemDetail :one
SELECT *
FROM queue_item
WHERE id = $1;

-- name: GetNodesByQueueItemID :many
SELECT *
FROM node
WHERE queue_item_id = $1;

-- name: CreateAndReturnNode :one
INSERT INTO node (queue_item_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetNodeByID :one
SELECT node.*, latest_status.submitted, latest_status.started, latest_status.ended, latest_status.status, result.execution_id, result.stdout, result.stderr, result.skipped,
    (SELECT array_agg(DISTINCT io_spec.id)::INT[] AS ids FROM io_spec WHERE io_spec.node_id = $1 AND io_spec.type = 'input') as inputs,
    (SELECT array_agg(DISTINCT io_spec.id)::INT[] AS ids FROM io_spec WHERE io_spec.node_id = $1 AND io_spec.type = 'output') as outputs,
    (SELECT array_agg(DISTINCT edge.parent_id)::INT[] AS ids FROM edge WHERE edge.child_id = $1) as parents,
    (SELECT array_agg(DISTINCT edge.child_id)::INT[] AS ids FROM edge WHERE edge.parent_id = $1) as children
FROM node
FULL OUTER JOIN (
    SELECT *
    FROM status
    WHERE status.node_id = $1
    ORDER BY id DESC LIMIT 1
) as latest_status ON node.id = latest_status.node_id
FULL OUTER JOIN result ON node.id = result.node_id
FULL OUTER JOIN io_spec ON node.id = io_spec.id
FULL OUTER JOIN edge ON node.id = edge.child_id
WHERE node.id = $1;

-- name: CreateEdge :exec
INSERT INTO edge (parent_id, child_id)
VALUES ($1, $2);

-- name: CreateIOSpec :exec
INSERT INTO io_spec (node_id, type, node_name, input_id, root, value, path, context)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetIOSpecByID :one
SELECT *
FROM io_spec
WHERE id = $1;

-- name: CreateResult :exec
INSERT INTO result (node_id, execution_id, stdout, stderr, skipped)
VALUES ($1, $2, $3, $4, $5);

-- name: CreateStatus :exec
INSERT INTO status (node_id, submitted, started, ended, status)
VALUES ($1, $2, $3, $4, $5);
