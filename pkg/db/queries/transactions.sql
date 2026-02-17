-- name: InsertTransaction :one
INSERT INTO transactions (block_num, tx_num, tx_id, validation_code)
VALUES ($1, $2, $3, $4)
ON CONFLICT (block_num, tx_num) DO UPDATE SET tx_id = EXCLUDED.tx_id
RETURNING id;

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

-- name: GetTransactionID :one
SELECT id
FROM transactions
WHERE block_num = $1 AND tx_num = $2;

-- name: InsertTxNamespace :exec
INSERT INTO tx_namespaces (transaction_id, ns_id, ns_version)
VALUES ($1, $2, $3)
ON CONFLICT (transaction_id, ns_id) DO NOTHING;
