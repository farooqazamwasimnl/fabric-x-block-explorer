-- name: InsertTransaction :exec
INSERT INTO transactions (block_num, tx_num, tx_id, validation_code)
VALUES ($1, $2, $3, $4)
ON CONFLICT (block_num, tx_num) DO NOTHING;

-- name: GetTransactionsByBlock :many
SELECT id, block_num, tx_num, tx_id, validation_code
FROM transactions
WHERE block_num = $1
ORDER BY tx_num
LIMIT $2 OFFSET $3;

-- name: GetTransactionByTxID :one
SELECT id, block_num, tx_num, tx_id, validation_code
FROM transactions
WHERE tx_id = $1;
