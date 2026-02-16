-- name: InsertWrite :exec
INSERT INTO writesets
(namespace_id, block_num, tx_num, tx_id, key, value)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetWritesByTx :many
SELECT id, namespace_id, block_num, tx_num, tx_id, key, value
FROM writesets
WHERE block_num = $1 AND tx_num = $2
ORDER BY id
LIMIT $3 OFFSET $4;
