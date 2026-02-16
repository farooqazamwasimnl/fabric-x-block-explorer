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
