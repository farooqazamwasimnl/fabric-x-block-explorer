# Fabric-X Block Explorer

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](../LICENSE)

**Fabric-X Block Explorer** is a high-performance Go-based application that streams blocks from Hyperledger Fabric networks, parses transaction data, stores it in PostgreSQL, and exposes REST and gRPC APIs for real-time blockchain exploration and analysis.

## Table of Contents

1. [Overview](#overview)
2. [Features](#features)
3. [Architecture](#architecture)
4. [Components](#components)
5. [Prerequisites](#prerequisites)
6. [Quick Start](#quick-start)
7. [Configuration](#configuration)
8. [API Documentation](#api-documentation)
9. [Development](#development)
10. [Deployment](#deployment)
11. [Database Schema](#database-schema)
12. [Health Checks and Monitoring](#health-checks-and-monitoring)
13. [Troubleshooting](#troubleshooting)
14. [License](#license)

---

## Overview

**Fabric-X Block Explorer** provides real-time visibility into Hyperledger Fabric network activity. It connects to a Fabric sidecar service, ingests blocks as they are committed, extracts transaction and writeset data, and stores them in a PostgreSQL database for efficient querying.

### Purpose

Provide an easy-to-run, extensible, and observable block explorer for Fabric networks that is simple to deploy and integrate into existing infrastructure.

### Key Responsibilities

- **Block Ingestion**: Continuously streams blocks from Fabric via gRPC sidecar connection
- **Transaction Parsing**: Extracts transaction details, writesets, and namespace policies  
- **Data Persistence**: Stores structured data in PostgreSQL using optimized schemas
- **REST API**: Provides HTTP endpoints for querying blocks, transactions, and policies
- **gRPC API**: Offers high-performance RPC endpoints for programmatic access
- **Swagger Documentation**: Interactive API documentation via OpenAPI specification
- **Health Monitoring**: Exposes readiness and liveness endpoints for container orchestration

---

## Features

- ✅ **Real-time Block Streaming**: Direct gRPC connection to Fabric sidecar with automatic reconnection
- ✅ **Transaction Processing Pipeline**: Multi-stage concurrent processing with worker pools
- ✅ **PostgreSQL Storage**: Normalized schema with SQLC-generated type-safe queries
- ✅ **Dual API Support**: REST (HTTP/JSON) and gRPC (Protocol Buffers) endpoints
- ✅ **Swagger UI**: Interactive API documentation and testing interface
- ✅ **Structured Logging**: JSON-formatted logs with configurable levels (zerolog)
- ✅ **Configuration Management**: YAML-based config with environment variable overrides
- ✅ **Docker Support**: Multi-stage Dockerfile and docker-compose orchestration
- ✅ **Health Checks**: Database connectivity validation for Kubernetes/Docker
- ✅ **Backoff Strategy**: Exponential backoff with jitter for sidecar reconnection

---

## Architecture

### High-Level Architecture

```
┌──────────────┐       ┌─────────────────┐       ┌──────────────┐
│   Fabric     │       │  Block Explorer │       │  PostgreSQL  │
│   Sidecar    ├──────>│   (gRPC/REST)   ├──────>│   Database   │
│  (gRPC API)  │ Block │                 │ Store │              │
└──────────────┘ Stream└─────────────────┘ Data  └──────────────┘
                             │
                             │ Expose APIs
                             v
                    ┌────────────────────┐
                    │   REST Clients     │
                    │   gRPC Clients     │
                    │   Swagger UI       │
                    └────────────────────┘
```

### Data Flow

1. **Block Reception**: `BlockReceiver` connects to Fabric sidecar and receives blocks via gRPC streaming
2. **Channel Processing**: Raw blocks are enqueued into `rawBlockChan` for asynchronous processing
3. **Parsing**: `BlockProcessor` workers parse blocks into structured `ProcessedBlock` objects
4. **Enrichment**: Parser extracts channel headers, transaction IDs, read/write sets, and policies
5. **Persistence**: `BlockWriter` workers persist processed data to PostgreSQL in batches
6. **API Serving**: REST and gRPC servers query the database and return results to clients

### Processing Pipeline

```
Sidecar → BlockReceiver → rawBlockChan → BlockProcessor → processedBlockChan → BlockWriter → PostgreSQL
            (1 goroutine)  (buffered)   (N workers)      (buffered)         (M workers)     (database)
```

**Pipeline Configuration**:
- `RAW_CHANNEL_SIZE`: Buffer size for raw blocks (default: configurable)
- `PROCESS_CHANNEL_SIZE`: Buffer size for processed blocks (default: configurable)
- `PROCESS_WORKERS`: Number of concurrent block processors (default: configurable)
- `WRITE_WORKERS`: Number of concurrent database writers (default: configurable)

---

## Components

### Core Packages

#### `cmd/explorer`
Application entry point and server bootstrap. Initializes configuration, database connections, gRPC/REST servers, and the block processing pipeline.

#### `pkg/sidecarstream`
Wrapper around the Fabric sidecar client. Manages gRPC connections to the sidecar deliver service with:
- Automatic reconnection on connection failure
- Exponential backoff with jitter for retry logic
- Block stream starting from configurable block number
- Optional end block number for bounded streaming

#### `pkg/blockpipeline`

**`BlockReceiver`**: Connects to sidecar and pushes raw blocks into the pipeline  
**`BlockProcessor`**: Parses raw blocks using `pkg/parser` into domain objects  
**`BlockWriter`**: Persists processed blocks, transactions, and writesets to PostgreSQL

#### `pkg/parser`
Extracts structured data from Fabric blocks:
- Channel header information
- Transaction IDs and validation codes
- Per-namespace read sets (key, version)
- Per-namespace write sets (key, value, is_delete)
- Namespace endorsement policies (from meta namespace)

#### `pkg/db`
Database layer with PostgreSQL connectivity:
- Connection pool management using `pgx/v5`
- SQLC-generated type-safe queries
- Transaction support for batch writes
- Schema migration via SQL scripts

#### `pkg/api`
REST API implementation:
- HTTP handlers for blocks, transactions, writesets, and policies
- JSON request/response marshaling
- Error handling and validation
- Router configuration with gorilla/mux

#### `pkg/app`
Application-level orchestration:
- Server lifecycle management
- Graceful shutdown handling
- Component wiring and dependency injection

#### `pkg/swagger`
OpenAPI documentation:
- Swagger YAML specification
- Embedded Swagger UI assets
- Interactive API testing interface

#### `pkg/contracts`
Protobuf definitions for gRPC API:
- Block message definitions
- Transaction message definitions
- Service contracts

#### `pkg/config`
Configuration management:
- YAML file parsing
- Environment variable overrides
- Validation and defaults

#### `pkg/workerpool`
Worker pool implementation for concurrent processing pipelines.

---

## Prerequisites

### Required Dependencies

- **Go 1.21+**: [Download](https://go.dev/dl/)
- **Docker 20.10+**: [Install Docker](https://docs.docker.com/get-docker/)
- **Docker Compose 2.0+**: [Install Docker Compose](https://docs.docker.com/compose/install/)
- **PostgreSQL 14+**: Database for persistent storage
- **sqlc 1.18+**: SQL code generation tool (optional, for schema changes)

### Optional Tools

- **make**: Build automation (recommended)
- **curl**: API testing
- **grpcurl**: gRPC endpoint testing

---

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer.git
cd fabric-x-block-explorer
```

### 2. Start with Docker Compose (Recommended)

The easiest way to get started is using docker-compose, which starts both PostgreSQL and the explorer:

```bash
docker-compose up
```

This will:
- Start a PostgreSQL container with the required schema
- Build and start the explorer container
- Expose REST API on `http://localhost:8080`
- Expose Swagger UI on `http://localhost:8080/swagger/`

### 3. Start Locally (Development)

**3.1. Start PostgreSQL**

```bash
docker run --name explorer-postgres \
  -e POSTGRES_PASSWORD=postgres \
  -e POSTGRES_DB=explorer \
  -p 5432:5432 \
  -d postgres:14
```

**3.2. Initialize Database Schema**

```bash
psql -h localhost -U postgres -d explorer -f pkg/db/schema.sql
```

**3.3. Configure the Explorer**

Copy the example configuration:

```bash
cp config.yaml.example config.yaml
```

Edit `config.yaml` to match your environment (see [Configuration](#configuration) section).

**3.4. Run the Explorer**

```bash
go run ./cmd/explorer/main.go
```

The explorer will start and connect to both the sidecar and database.

### 4. Verify Installation

Test the health endpoint:

```bash
curl http://localhost:8080/healthz
```

Expected response:
```json
{"status":"ok"}
```

Access Swagger UI: `http://localhost:8080/swagger/`

---

## Configuration

The explorer is configured via a YAML file (`config.yaml`) with optional environment variable overrides. All configuration parameters are validated on startup.

### Configuration File Location

The explorer searches for `config.yaml` in:
1. Current working directory
2. `/etc/fabric-x-explorer/` (for system installations)
3. `~/.config/fabric-x-explorer/` (for user installations)

### Configuration Structure

```yaml
# Sidecar Connection
sidecar:
  host: "host.docker.internal"   # Sidecar hostname/IP
  port: 4001                      # Sidecar gRPC port
  channel: "mychannel"            # Fabric channel name
  start_block: 0                  # Start streaming from block number (0 = genesis)
  end_block: 0                    # Optional: Stop at block number (0 = no limit)

# Database Configuration
database:
  host: "localhost"               # PostgreSQL hostname
  port: 5432                      # PostgreSQL port
  user: "postgres"                # Database user
  password: "postgres"            # Database password
  name: "explorer"                # Database name
  sslmode: "disable"              # SSL mode: disable, require, verify-ca, verify-full

# HTTP Server
server:
  http_addr: ":8080"              # HTTP bind address (REST + Swagger)

# Logging
logging:
  level: "info"                   # Log level: debug, info, warn, error
  format: "json"                  # Log format: json, console

# Pipeline Configuration
pipeline:
  raw_channel_size: 100           # Raw block channel buffer size
  process_channel_size: 100       # Processed block channel buffer size
  process_workers: 4              # Number of block processor workers
  write_workers: 2                # Number of database writer workers
```

### Environment Variables

All configuration values can be overridden with environment variables using the pattern `EXPLORER_<SECTION>_<KEY>`:

```bash
# Sidecar configuration
export EXPLORER_SIDECAR_HOST="192.168.1.100"
export EXPLORER_SIDECAR_PORT="4001"
export EXPLORER_SIDECAR_CHANNEL="mychannel"
export EXPLORER_SIDECAR_START_BLOCK="0"

# Database configuration
export EXPLORER_DATABASE_HOST="postgres.example.com"
export EXPLORER_DATABASE_PORT="5432"
export EXPLORER_DATABASE_USER="explorer_user"
export EXPLORER_DATABASE_PASSWORD="secret_password"
export EXPLORER_DATABASE_NAME="fabric_explorer"
export EXPLORER_DATABASE_SSLMODE="require"

# Server configuration
export EXPLORER_SERVER_HTTP_ADDR=":9090"

# Logging configuration
export EXPLORER_LOGGING_LEVEL="debug"
export EXPLORER_LOGGING_FORMAT="console"

# Pipeline configuration
export EXPLORER_PIPELINE_RAW_CHANNEL_SIZE="200"
export EXPLORER_PIPELINE_PROCESS_CHANNEL_SIZE="200"
export EXPLORER_PIPELINE_PROCESS_WORKERS="8"
export EXPLORER_PIPELINE_WRITE_WORKERS="4"
```

### Configuration Best Practices

1. **Never commit config.yaml**: The `.gitignore` excludes it; use `config.yaml.example` as a template
2. **Use environment variables in containers**: Override config in Docker/Kubernetes deployments
3. **Validate sidecar connectivity**: Ensure the sidecar is reachable before starting the explorer
4. **Adjust worker counts**: Tune `process_workers` and `write_workers` based on load and resources
5. **Enable SSL for production**: Set `database.sslmode` to `require` or `verify-full` for PostgreSQL
6. **Use structured logging**: Keep `logging.format` as `json` for production observability

---

## API Documentation

The explorer exposes **5 REST endpoints** and corresponding **gRPC services**. All REST endpoints return JSON.

### REST Endpoints

#### 1. Get Current Block Height

**Endpoint**: `GET /blocks/height`

**Description**: Returns the highest block number currently stored in the database.

**Response**:
```json
{
  "height": 12345
}
```

**Example**:
```bash
curl http://localhost:8080/blocks/height
```

---

#### 2. Get Block by Number

**Endpoint**: `GET /blocks/{block_number}`

**Description**: Retrieves a specific block by its block number, including all transactions and writesets.

**Path Parameters**:
- `block_number` (integer, required): Block number to retrieve

**Response**:
```json
{
  "block_number": 100,
  "block_hash": "a1b2c3...",
  "previous_hash": "d4e5f6...",
  "data_hash": "g7h8i9...",
  "timestamp": "2024-01-15T10:30:00Z",
  "transaction_count": 5
}
```

**Example**:
```bash
curl http://localhost:8080/blocks/100
```

**Error Responses**:
- `404 Not Found`: Block number does not exist
- `400 Bad Request`: Invalid block number format

---

#### 3. Get Transaction by ID

**Endpoint**: `GET /tx/{transaction_id}`

**Description**: Retrieves a specific transaction by its transaction ID (hex-encoded).

**Path Parameters**:
- `transaction_id` (string, required): Transaction ID in hexadecimal format

**Response**:
```json
{
  "tx_id": "abc123...",
  "block_number": 100,
  "tx_index": 2,
  "channel_id": "mychannel",
  "validation_code": 0,
  "timestamp": "2024-01-15T10:30:00Z",
  "writesets": [
    {
      "namespace": "mycc",
      "key": "asset1",
      "value": "{\"owner\":\"alice\",\"value\":1000}",
      "is_delete": false
    }
  ]
}
```

**Example**:
```bash
curl http://localhost:8080/tx/abc123def456...
```

**Error Responses**:
- `404 Not Found`: Transaction ID does not exist
- `400 Bad Request`: Invalid transaction ID format

---

#### 4. Get Namespace Policies

**Endpoint**: `GET /policies/{namespace}`

**Description**: Retrieves endorsement policies for a specific namespace (chaincode).

**Path Parameters**:
- `namespace` (string, required): Namespace/chaincode name

**Query Parameters**:
- `latest` (boolean, optional): If `true`, returns only the latest policy version (default: `false`)

**Response**:
```json
[
  {
    "id": 1,
    "namespace": "mycc",
    "version": 2,
    "policy": "OR('Org1MSP.peer', 'Org2MSP.peer')"
  }
]
```

**Example**:
```bash
# Get all policies for namespace
curl http://localhost:8080/policies/mycc

# Get only the latest policy
curl http://localhost:8080/policies/mycc?latest=true
```

**Error Responses**:
- `404 Not Found`: Namespace has no policies
- `400 Bad Request`: Invalid namespace format

---

#### 5. Health Check

**Endpoint**: `GET /healthz`

**Description**: Returns the health status of the explorer, including database connectivity.

**Response (Healthy)**:
```json
{
  "status": "ok"
}
```

**Response (Unhealthy)**:
```json
{
  "status": "unavailable",
  "details": "db ping failed: connection refused"
}
```

**HTTP Status Codes**:
- `200 OK`: Service is healthy
- `503 Service Unavailable`: Service is unhealthy (database unreachable)

**Example**:
```bash
curl http://localhost:8080/healthz
```

**Usage**: Kubernetes/Docker health checks should use this endpoint for liveness and readiness probes.

---

### Swagger / OpenAPI

The explorer serves an interactive Swagger UI for API exploration and testing:

**Swagger UI**: `http://localhost:8080/swagger/`  
**OpenAPI Spec**: `http://localhost:8080/swagger.yaml`

The Swagger UI allows you to:
- Browse all available endpoints
- View request/response schemas
- Test endpoints directly from the browser
- Download the OpenAPI specification

---

### gRPC API

The explorer also exposes gRPC endpoints for high-performance programmatic access. See `pkg/contracts/contracts.proto` for service definitions.

**gRPC Server**: `localhost:9090` (configurable)

**Example using grpcurl**:
```bash
# List available services
grpcurl -plaintext localhost:9090 list

# Call GetBlockHeight
grpcurl -plaintext localhost:9090 explorer.BlockExplorer/GetBlockHeight

# Call GetBlock
grpcurl -plaintext -d '{"block_number": 100}' localhost:9090 explorer.BlockExplorer/GetBlock
```

**Note**: gRPC endpoints are not documented in Swagger (OpenAPI limitation). Use `.proto` files or gRPC reflection for discovery.

---

## Development

### Building Locally

```bash
# Install dependencies
go mod download

# Build the binary
go build -o explorer ./cmd/explorer/main.go

# Run the binary
./explorer
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run tests with verbose output
go test -v ./...
```

### Database Schema Changes

The explorer uses **sqlc** for type-safe SQL query generation. When modifying the database schema:

1. Update `pkg/db/schema.sql` with your schema changes
2. Update SQL queries in `pkg/db/queries/*.sql`
3. Regenerate Go code:

```bash
sqlc generate
```

**Running sqlc with Docker** (if not installed locally):

```bash
docker run --rm \
  -v "$(pwd):/src" \
  -w /src \
  sqlc/sqlc generate
```

### Code Style and Linting

The project follows standard Go conventions:

```bash
# Format code
go fmt ./...

# Run linter (requires golangci-lint)
golangci-lint run

# Vet code
go vet ./...
```

---

## Deployment

### Docker

#### Build Docker Image

```bash
docker build -t fabric-x-explorer:latest .
```

The multi-stage Dockerfile:
- Caches Go module downloads
- Builds a static binary
- Creates a minimal runtime image (~20MB)

#### Run Docker Container

```bash
docker run --rm -p 8080:8080 \
  -e EXPLORER_SIDECAR_HOST="host.docker.internal" \
  -e EXPLORER_SIDECAR_PORT="4001" \
  -e EXPLORER_SIDECAR_CHANNEL="mychannel" \
  -e EXPLORER_DATABASE_HOST="postgres.example.com" \
  -e EXPLORER_DATABASE_PORT="5432" \
  -e EXPLORER_DATABASE_USER="postgres" \
  -e EXPLORER_DATABASE_PASSWORD="postgres" \
  -e EXPLORER_DATABASE_NAME="explorer" \
  fabric-x-explorer:latest
```

---

### Docker Compose

A sample `docker-compose.yaml` is provided with:
- PostgreSQL database with schema initialization
- Fabric-X Explorer with environment variable configuration
- Health checks for both services
- Volume mounts for persistent storage

#### Start Services

```bash
docker-compose up -d
```

#### View Logs

```bash
docker-compose logs -f explorer
```

#### Stop Services

```bash
docker-compose down
```

#### Configuration

Edit `docker-compose.yaml` to customize:
- Database credentials
- Sidecar connection settings
- Port mappings
- Resource limits

---

### Kubernetes

Example Kubernetes manifests (create `k8s/` directory):

**Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: fabric-x-explorer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: fabric-x-explorer
  template:
    metadata:
      labels:
        app: fabric-x-explorer
    spec:
      containers:
      - name: explorer
        image: fabric-x-explorer:latest
        ports:
        - containerPort: 8080
        env:
        - name: EXPLORER_SIDECAR_HOST
          value: "sidecar-service"
        - name: EXPLORER_DATABASE_HOST
          value: "postgres-service"
        livenessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 10
          periodSeconds: 30
        readinessProbe:
          httpGet:
            path: /healthz
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 10
```

**Service**:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: fabric-x-explorer
spec:
  selector:
    app: fabric-x-explorer
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8080
  type: LoadBalancer
```

---

## Database Schema

The explorer uses PostgreSQL with four main tables:

### `blocks` Table

Stores block metadata:

| Column | Type | Description |
|--------|------|-------------|
| `block_number` | BIGINT PRIMARY KEY | Block sequence number |
| `block_hash` | TEXT NOT NULL | Block hash (hex) |
| `previous_hash` | TEXT | Previous block hash |
| `data_hash` | TEXT | Data hash |
| `timestamp` | TIMESTAMP | Block timestamp |
| `transaction_count` | INTEGER | Number of transactions |

### `transactions` Table

Stores transaction metadata:

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL PRIMARY KEY | Auto-increment ID |
| `tx_id` | TEXT UNIQUE NOT NULL | Transaction ID (hex) |
| `block_number` | BIGINT REFERENCES blocks | Parent block number |
| `tx_index` | INTEGER | Transaction index in block |
| `channel_id` | TEXT | Channel name |
| `validation_code` | INTEGER | Validation code (0 = valid) |
| `timestamp` | TIMESTAMP | Transaction timestamp |

### `writesets` Table

Stores key-value writes:

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL PRIMARY KEY | Auto-increment ID |
| `tx_id` | TEXT REFERENCES transactions | Parent transaction ID |
| `block_number` | BIGINT | Block number |
| `namespace` | TEXT | Chaincode namespace |
| `key` | TEXT | State key |
| `value` | TEXT | State value (NULL if deleted) |
| `is_delete` | BOOLEAN | True if this is a delete operation |

### `namespaces` Table

Stores namespace endorsement policies:

| Column | Type | Description |
|--------|------|-------------|
| `id` | SERIAL PRIMARY KEY | Auto-increment ID |
| `namespace` | TEXT NOT NULL | Chaincode namespace |
| `version` | BIGINT | Policy version |
| `policy` | TEXT | Endorsement policy |
| `block_number` | BIGINT | Block where policy was set |

**Indexes**:
- `blocks.block_number` (primary key)
- `transactions.tx_id` (unique index)
- `transactions.block_number` (foreign key index)
- `writesets.tx_id`, `writesets.namespace`, `writesets.key` (query optimization)
- `namespaces.namespace`, `namespaces.version` (query optimization)

---

## Health Checks and Monitoring

### Health Endpoint

**Liveness Probe**: `/healthz`

The health endpoint performs:
1. Database connectivity check (ping)
2. Connection pool status validation

**Response Codes**:
- `200 OK`: Healthy (database reachable)
- `503 Service Unavailable`: Unhealthy (database unreachable)

### Docker Health Check

Example in `docker-compose.yaml`:

```yaml
healthcheck:
  test: ["CMD-SHELL", "curl -f http://localhost:8080/healthz || exit 1"]
  interval: 30s
  timeout: 5s
  retries: 3
  start_period: 10s
```

### Kubernetes Health Probes

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 30
  timeoutSeconds: 5
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5
  periodSeconds: 10
  timeoutSeconds: 3
  failureThreshold: 2
```

### Logging

The explorer uses **structured logging** (JSON format) via zerolog:

**Log Levels**: `debug`, `info`, `warn`, `error`

**Example Logs**:
```json
{"level":"info","component":"sidecar","message":"Connected to sidecar","host":"localhost","port":4001}
{"level":"info","component":"pipeline","message":"Block processed","block_number":100,"tx_count":5}
{"level":"error","component":"database","message":"Database connection failed","error":"connection refused"}
```

**Adjust Log Level**:
```bash
export EXPLORER_LOGGING_LEVEL="debug"
```

### Metrics (Future Enhancement)

Consider adding Prometheus metrics for:
- Blocks processed per second
- Transaction processing latency
- Database query latency
- Pipeline channel depths
- Error rates by component

---

## Troubleshooting

### Common Issues

#### 1. Cannot Connect to Sidecar

**Symptoms**: Logs show "connection refused" or "context deadline exceeded"

**Solutions**:
- Verify sidecar is running: `curl http://localhost:4001/healthz`
- Check firewall rules and network connectivity
- In Docker, use `host.docker.internal` instead of `localhost`
- Verify `EXPLORER_SIDECAR_HOST` and `EXPLORER_SIDECAR_PORT` configuration

#### 2. Database Connection Failed

**Symptoms**: Health endpoint returns 503, logs show "db ping failed"

**Solutions**:
- Verify PostgreSQL is running: `psql -h localhost -U postgres -d explorer`
- Check database credentials in configuration
- Ensure database schema is initialized: `psql -f pkg/db/schema.sql`
- Verify `EXPLORER_DATABASE_*` environment variables

#### 3. No Blocks Being Processed

**Symptoms**: Block height stays at 0, no data in database

**Solutions**:
- Check sidecar channel configuration: Ensure `EXPLORER_SIDECAR_CHANNEL` matches Fabric channel
- Verify start block: Set `EXPLORER_SIDECAR_START_BLOCK=0` to start from genesis
- Check Fabric network activity: Ensure transactions are being submitted
- Review logs for parsing errors: Look for "block parsing failed" messages

#### 4. Swagger UI Not Loading

**Symptoms**: 404 error when accessing `/swagger/`

**Solutions**:
- Verify server is running: `curl http://localhost:8080/healthz`
- Check HTTP bind address: Ensure `EXPLORER_SERVER_HTTP_ADDR` is correct
- Clear browser cache and retry
- Verify `pkg/swagger/ui/` directory exists and contains assets

#### 5. High Memory Usage

**Symptoms**: Container OOM killed or high memory consumption

**Solutions**:
- Reduce pipeline buffer sizes: Lower `raw_channel_size` and `process_channel_size`
- Reduce worker counts: Lower `process_workers` and `write_workers`
- Monitor channel depths: Add metrics to track buffer usage
- Enable database connection pooling limits

#### 6. Slow Query Performance

**Symptoms**: API endpoints timeout or respond slowly

**Solutions**:
- Add database indexes: Ensure indexes on `block_number`, `tx_id`, `namespace`, `key`
- Optimize queries: Review SQLC-generated queries in `pkg/db/sqlc/`
- Increase database resources: Add more CPU/memory to PostgreSQL
- Use connection pooling: Verify `pgx` pool configuration
- Add query result caching: Implement Redis or in-memory cache

---

## Contributing

We welcome contributions! Please follow these guidelines:

1. **Fork the repository** and create a feature branch
2. **Follow Go conventions**: Use `gofmt`, `go vet`, and `golangci-lint`
3. **Write tests**: Ensure new code has unit tests
4. **Update documentation**: Keep README and Swagger specs up to date
5. **Sign your commits**: Use DCO signoff (`git commit -s`)
6. **Submit a pull request**: Describe your changes clearly

### Development Setup

See the [Development](#development) section for local setup instructions.

### Code of Conduct

This project follows the [Hyperledger Code of Conduct](https://wiki.hyperledger.org/display/HYP/Hyperledger+Code+of+Conduct).

---

## License

This project is licensed under the **Apache License 2.0**. See [LICENSE](../LICENSE) for details.

---

## Additional Resources

- **Hyperledger Fabric Documentation**: https://hyperledger-fabric.readthedocs.io/
- **Fabric Sidecar**: https://github.com/hyperledger/fabric-sidecar
- **PostgreSQL Documentation**: https://www.postgresql.org/docs/
- **sqlc Documentation**: https://docs.sqlc.dev/
- **Go Documentation**: https://go.dev/doc/

---

**Maintained by**: [LF Decentralized Trust Labs](https://github.com/LF-Decentralized-Trust-labs)  
**Repository**: [fabric-x-block-explorer](https://github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer)
