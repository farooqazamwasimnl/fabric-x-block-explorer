CREATE TABLE IF NOT EXISTS blocks (
    block_num BIGINT PRIMARY KEY,
    tx_count INT NOT NULL,
    previous_hash BYTEA,
    data_hash BYTEA
);

CREATE TABLE IF NOT EXISTS transactions (
    id BIGSERIAL PRIMARY KEY,
    block_num BIGINT NOT NULL REFERENCES blocks(block_num),
    tx_num BIGINT NOT NULL,
    tx_id BYTEA NOT NULL,
    validation_code BIGINT NOT NULL,  
    UNIQUE (block_num, tx_num)
);

CREATE TABLE IF NOT EXISTS tx_namespaces (
    id BIGSERIAL PRIMARY KEY,
    transaction_id BIGINT NOT NULL REFERENCES transactions(id),
    ns_id TEXT NOT NULL,
    ns_version BIGINT NOT NULL,
    UNIQUE (transaction_id, ns_id)
);

CREATE TABLE IF NOT EXISTS tx_reads (
    id BIGSERIAL PRIMARY KEY,
    tx_namespace_id BIGINT NOT NULL REFERENCES tx_namespaces(id),
    key BYTEA NOT NULL,
    version BIGINT,
    is_read_write BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE TABLE IF NOT EXISTS namespaces (
    id BIGSERIAL PRIMARY KEY,
    name BYTEA NOT NULL UNIQUE   
);

CREATE TABLE IF NOT EXISTS writesets (
    id BIGSERIAL PRIMARY KEY,
    namespace_id BIGINT NOT NULL REFERENCES namespaces(id),
    block_num BIGINT NOT NULL,
    tx_num BIGINT NOT NULL,
    tx_id BYTEA NOT NULL,   
    key BYTEA NOT NULL,     
    value BYTEA
);
