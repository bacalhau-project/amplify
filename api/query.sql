-- name: ListQueueItems :many
SELECT * 
FROM queue_item;

-- name: GetQueueItemDetail :one
SELECT *
FROM queue_item
WHERE id = $1 LIMIT 1;

-- name: CreateQueueItem :exec
INSERT INTO queue_item (id, inputs, created_at)
VALUES ($1, $2, $3);

-- -- name: DeleteAuthor :exec
-- DELETE FROM authors
-- WHERE id = $1;

-- -- name: UpdateAuthor :exec
-- UPDATE authors
--   set name = $2,
--   bio = $3
-- WHERE id = $1;