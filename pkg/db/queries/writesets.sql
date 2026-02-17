-- name: InsertWrite :exec
INSERT INTO writesets
(namespace_id, block_num, tx_num, tx_id, key, value)
VALUES ($1, $2, $3, $4, $5, $6);

