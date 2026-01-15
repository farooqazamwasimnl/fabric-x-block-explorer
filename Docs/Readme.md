# Fabric‑X Block Explorer

A lightweight service that ingests Hyperledger Fabric blocks from a Sidecar stream, parses transactions and read/write sets, persists processed data to a relational database, and exposes a REST API (with OpenAPI/Swagger) for querying blocks, transactions, and write records.

## Overview

### Purpose

Provide an easy-to-run block explorer for Fabric networks that is simple to extend, observable, and fast to build and deploy.

### Key Responsibilities

- Receive blocks from a Sidecar deliver stream
- Parse transactions and extract read/write sets
- Persist processed blocks, transactions, and writes to Postgres
- Serve a REST API for queries and a Swagger/OpenAPI spec
- Expose a health endpoint for liveness and readiness checks

## Architecture and Flow

### Components

- **cmd/explorer** — application entry point and server bootstrap
- **pkg/sidecarstream** — wrapper around the Sidecar client; manages deliver connections and reconnection/backoff
- **pkg/blockpipeline**
  - **BlockReceiver** — consumes Sidecar blocks and forwards them into the pipeline
  - **BlockProcessor** — parses blocks into ProcessedBlock using parser.Parse
  - **BlockWriter** — persists processed data to the database
- **pkg/parser** — extracts channel header, txID, and per-namespace read/write sets; converts them into domain records
- **pkg/db** — database writer and SQLC-generated queries
- **pkg/api** — HTTP handlers, router, and OpenAPI/Swagger serving

### Data Flow

- **Sidecar → Streamer**: Streamer.StartDeliver opens a deliver stream and writes common.Block messages to a buffered channel
- **Receiver → Pipeline**: BlockReceiver reads from the Sidecar channel and forwards non-nil blocks to the pipeline input channel; reconnects with backoff on errors
- **Processor**: BlockProcessor calls parser.Parse to produce WriteRecord entries and BlockInfo
- **Writer**: BlockWriter persists the processed block and associated write records to Postgres
- **API**: REST endpoints read persisted data and return JSON responses; /healthz performs a short DB ping for readiness

## API

All endpoints return JSON. Replace `localhost:8080` with your configured host/port.

### Get Current Block Height

```bash
GET /blocks/height
```

**Response 200:**
```json
{
  "height": 12345
}
```

### Get Block Details

```bash
GET /blocks/{block_num}?limitTx=100&offsetTx=0&limitWrites=1000&offsetWrites=0
```

**Response 200:**
```json
{
  "block_num": 123,
  "tx_count": 2,
  "previous_hash": "abcd...",
  "data_hash": "ef01...",
  "transactions": [
    {
      "id": 1,
      "block_num": 123,
      "tx_num": 0,
      "tx_id": "deadbeef...",
      "validation_code": 0,
      "writes": []
    }
  ]
}
```

### Get Transaction by tx_id

```bash
GET /tx/{tx_id_hex}
```

**Response 200:**
```json
{
  "transaction": {},
  "block": {
    "block_num": 123,
    "tx_count": 2,
    "previous_hash": "abcd...",
    "data_hash": "ef01..."
  }
}
```

### Health Endpoint

```bash
GET /healthz
```

- **200** → `{"status":"ok"}`
- **503** → `{"status":"unavailable","details":"db ping failed: <error>"}`

### Swagger / OpenAPI

The app serves an OpenAPI spec (e.g., `/swagger.yaml`) and a Swagger UI route if configured. Use the UI to explore endpoints interactively.


## Configuration

Environment variables are read by the config package.

### Sidecar

- `SIDECAR_HOST` — sidecar host (e.g., `host.docker.internal`)
- `SIDECAR_PORT` — sidecar port (e.g., `4001`)
- `SIDECAR_CHANNEL` — Fabric channel ID (e.g., `mychannel`)
- `SIDECAR_START_BLOCK` — start block number (default `0`)
- `SIDECAR_END_BLOCK` — optional end block number (omit for no upper limit)

### Database

- `DB_HOST` — Postgres host
- `DB_PORT` — Postgres port
- `DB_USER` — Postgres user
- `DB_PASSWORD` — Postgres password
- `DB_NAME` — Postgres database name
- `DB_SSLMODE` — Postgres sslmode (e.g., `disable`)

### Server

- `HTTP_ADDR` — HTTP bind address (default `:8080`)

### Optional Tuning

Recommended to expose via env if needed:

- `RAW_CHANNEL_SIZE`, `PROCESS_CHANNEL_SIZE` — channel buffer sizes
- `PROCESS_WORKERS`, `WRITE_WORKERS` — pipeline concurrency



## Build and Run

### Local (Development)

```bash
# run directly
go run ./cmd/explorer/main.go
```

### Docker

A multi-stage Dockerfile is included that caches Go modules and produces a small static binary.

**Build:**
```bash
docker build -t fabric-x-explorer:latest .
```

**Run:**
```bash
# run
docker run --rm -p 8080:8080 \
  -e SIDECAR_HOST="host.docker.internal" \
  -e SIDECAR_PORT="4001" \
  -e SIDECAR_CHANNEL="mychannel" \
  -e DB_HOST="host.docker.internal" \
  -e DB_PORT="5432" \
  -e DB_USER="postgres" \
  -e DB_PASSWORD="postgres" \
  -e DB_NAME="explorer" \
  fabric-x-explorer:latest
```

### Docker Compose

A sample `docker-compose.yml` is provided. It includes a healthcheck that calls `/healthz` and sensible logging options.

## Health Checks and Monitoring

**Liveness/Readiness:**
- `/healthz` returns 200 when the process is alive and DB ping succeeds
- Returns 503 when DB ping fails

**Docker Healthcheck Example:**
```yaml
healthcheck:
  test: ["CMD-SHELL", "curl -f http://localhost:8080/healthz || exit 1"]
  interval: 30s
  timeout: 5s
  retries: 3
  start_period: 10s
```

## Running sqlc with Docker
Implementation is done for Postgress for data storage, and sqlc tool is used to generate the query templates.
If you don't have `sqlc` installed locally, you can run it using Docker.  
This command generates code based on your `sqlc.yaml` configuration and SQL files:

```sh
docker run --rm \
  -v "$(pwd):/src" \
  -w /src \
  sqlc/sqlc generate
