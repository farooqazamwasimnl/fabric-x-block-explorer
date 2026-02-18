# Fabric-X Block Explorer - Architecture Documentation

## Table of Contents

1. [System Overview](#system-overview)
2. [High-Level Architecture](#high-level-architecture)
3. [Code Structure](#code-structure)
4. [Component Design](#component-design)
5. [Data Flow](#data-flow)
6. [Database Schema](#database-schema)
7. [API Design](#api-design)
8. [Concurrency Model](#concurrency-model)
9. [Error Handling](#error-handling)
10. [Testing Strategy](#testing-strategy)

---

## System Overview

Fabric-X Block Explorer is a real-time blockchain data indexer and query service for Hyperledger Fabric networks. It provides:

- **Real-time ingestion**: Streams blocks from Fabric sidecar service via gRPC
- **Structured storage**: Parses and stores blockchain data in PostgreSQL
- **Dual API**: REST (HTTP/JSON) and gRPC (Protocol Buffers) query interfaces
- **High performance**: Concurrent processing pipeline with worker pools
- **Production-ready**: Health checks, structured logging, graceful shutdown

### Design Principles

1. **Separation of Concerns**: Clear boundaries between ingestion, processing, storage, and API layers
2. **Concurrency**: Multi-stage pipeline with buffered channels and worker pools
3. **Type Safety**: SQLC-generated database code, Protocol Buffers for gRPC
4. **Testability**: Comprehensive unit and integration tests (75%+ coverage)
5. **Observability**: Structured logging, health endpoints, error propagation

---

## High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          Fabric-X Block Explorer                             │
│                                                                              │
│  ┌────────────┐     ┌──────────────┐     ┌─────────────┐     ┌──────────┐ │
│  │   Fabric   │────>│  Sidecar     │────>│   Block     │────>│          │ │
│  │   Network  │     │   gRPC       │     │  Receiver   │     │   Raw    │ │
│  │            │     │  Streaming   │     │ (goroutine) │     │  Channel │ │
│  └────────────┘     └──────────────┘     └─────────────┘     └─────┬────┘ │
│                                                                      │      │
│                                                                      v      │
│                     ┌────────────────────────────────────────────────────┐ │
│                     │        Block Processor Pool (N workers)           │ │
│                     │  ┌─────────┐  ┌─────────┐       ┌─────────┐      │ │
│                     │  │ Worker1 │  │ Worker2 │  ...  │ WorkerN │      │ │
│                     │  │ Parser  │  │ Parser  │       │ Parser  │      │ │
│                     │  └────┬────┘  └────┬────┘       └────┬────┘      │ │
│                     └───────┼────────────┼─────────────────┼───────────┘ │
│                             │            │                 │             │
│                             v            v                 v             │
│                     ┌─────────────────────────────────────────┐          │
│                     │      Processed Block Channel            │          │
│                     └───────────────────┬─────────────────────┘          │
│                                         │                                │
│                                         v                                │
│                     ┌────────────────────────────────────────────────┐   │
│                     │        Block Writer Pool (M workers)           │   │
│                     │  ┌─────────┐  ┌─────────┐       ┌─────────┐   │   │
│                     │  │ Writer1 │  │ Writer2 │  ...  │ WriterM │   │   │
│                     │  │   DB    │  │   DB    │       │   DB    │   │   │
│                     │  └────┬────┘  └────┬────┘       └────┬────┘   │   │
│                     └───────┼────────────┼─────────────────┼────────┘   │
│                             │            │                 │            │
│                             v            v                 v            │
│                     ┌─────────────────────────────────────────┐         │
│                     │         PostgreSQL Database             │         │
│                     │  ┌─────────┐  ┌──────────┐  ┌────────┐ │         │
│                     │  │ Blocks  │  │  Txns    │  │ Writes │ │         │
│                     │  └─────────┘  └──────────┘  └────────┘ │         │
│                     └─────────────────────────────────────────┘         │
│                                         ^                               │
│                                         │                               │
│                     ┌───────────────────┴───────────────────┐           │
│                     │          API Layer                    │           │
│                     │  ┌──────────┐      ┌──────────────┐  │           │
│                     │  │   REST   │      │    gRPC      │  │           │
│                     │  │   :8080  │      │    :9090     │  │           │
│                     │  └──────────┘      └──────────────┘  │           │
│                     └─────────────────────────────────────────┘         │
│                                         │                               │
└─────────────────────────────────────────┼───────────────────────────────┘
                                          │
                                          v
                              ┌───────────────────────┐
                              │   External Clients    │
                              │  - Web Applications   │
                              │  - CLI Tools          │
                              │  - Monitoring Systems │
                              └───────────────────────┘
```

---

## Code Structure

```
fabric-x-block-explorer/
├── cmd/
│   └── explorer/
│       └── main.go                 # Application entry point
│
├── pkg/
│   ├── api/                        # REST & gRPC API layer
│   │   ├── handlers.go             # REST HTTP handlers
│   │   ├── router.go               # HTTP routing configuration
│   │   ├── grpc_server.go          # gRPC service implementation
│   │   ├── proto/                  # Protocol Buffer definitions
│   │   │   ├── explorer.proto      # API service contract
│   │   │   └── explorer.pb.go      # Generated protobuf code
│   │   └── *_test.go               # API integration tests
│   │
│   ├── app/                        # Application orchestration
│   │   ├── server.go               # Server lifecycle management
│   │   └── server_test.go          # Server tests
│   │
│   ├── blockpipeline/              # Block processing pipeline
│   │   ├── backoff.go              # Exponential backoff for reconnection
│   │   ├── processor.go            # Block parsing workers
│   │   ├── receiver.go             # Sidecar block receiver
│   │   ├── writer.go               # Database persistence workers
│   │   └── *_test.go               # Pipeline component tests
│   │
│   ├── config/                     # Configuration management
│   │   ├── config.go               # YAML parsing, env overrides
│   │   └── config_test.go          # Configuration tests
│   │
│   ├── contracts/                  # Smart contract utilities (placeholder)
│   │   └── contracts.go            
│   │
│   ├── db/                         # Database access layer
│   │   ├── db_writer.go            # High-level write operations
│   │   ├── postgress.go            # Connection pool management
│   │   ├── schema.sql              # Database schema DDL
│   │   ├── queries/                # SQL query definitions
│   │   │   ├── blocks.sql          # Block queries
│   │   │   ├── transactions.sql    # Transaction queries
│   │   │   ├── writesets.sql       # Writeset queries
│   │   │   └── namespaces.sql      # Namespace policy queries
│   │   ├── sqlc/                   # SQLC generated code
│   │   │   ├── models.go           # Database models
│   │   │   ├── db.go               # SQLC DB interface
│   │   │   ├── querier.go          # Query interface
│   │   │   └── *_sql.go            # Generated query functions
│   │   ├── dbtest/                 # Test infrastructure
│   │   │   └── testcontainer.go    # PostgreSQL test containers
│   │   └── *_test.go               # Database tests
│   │
│   ├── parser/                     # Block parsing logic
│   │   ├── parser.go               # Transaction/writeset extraction
│   │   └── parser_test.go          # Parser tests
│   │
│   ├── sidecarstream/              # Fabric sidecar gRPC client
│   │   ├── streamer.go             # Block streaming client
│   │   └── streamer_test.go        # Streaming tests
│   │
│   ├── swagger/                    # OpenAPI documentation
│   │   ├── swagger.yaml            # API specification
│   │   ├── swagger.go              # Swagger handler
│   │   └── ui/                     # Swagger UI static assets
│   │
│   ├── types/                      # Domain models
│   │   ├── types.go                # Core blockchain types
│   │   └── api_responses.go        # API response models
│   │
│   ├── util/                       # Utility functions
│   │   ├── nullable.go             # NULL handling for database
│   │   └── nullable_test.go        # Utility tests
│   │
│   └── workerpool/                 # Worker pool implementation
│       └── workerpool.go           # Generic worker pool
│
├── scripts/                        # Build and deployment scripts
│   └── filter-coverage.sh          # Test coverage filtering
│
├── Docs/                           # Documentation
│   └── Readme.md                   # User documentation
│
├── Makefile                        # Build automation
├── Dockerfile                      # Container image definition
├── docker-compose.yaml             # Local development stack
├── config.yaml.example             # Configuration template
├── sqlc.yaml                       # SQLC configuration
├── go.mod                          # Go module dependencies
└── ARCHITECTURE.md                 # This file
```

---

## Component Design

### 1. Block Receiver (`pkg/blockpipeline/receiver.go`)

**Responsibility**: Connect to Fabric sidecar and stream blocks into the pipeline.

**Key Features**:
- Single goroutine per receiver
- Automatic reconnection with exponential backoff
- Forwards raw blocks to `rawBlockChan`
- Graceful shutdown on context cancellation

**Implementation**:
```go
type BlockReceiver struct {
    streamer   *sidecarstream.Streamer
    blockCh    chan<- *common.Block
    backoff    *backoff.ExponentialBackOff
}

func (r *BlockReceiver) Start(ctx context.Context, errCh chan<- error)
```

**Concurrency**: 1 goroutine

---

### 2. Block Processor (`pkg/blockpipeline/processor.go`)

**Responsibility**: Parse raw Fabric blocks into structured domain objects.

**Key Features**:
- Worker pool of N concurrent parsers
- Each worker processes blocks from `rawBlockChan`
- Outputs `ProcessedBlock` to `processedBlockChan`
- Uses `pkg/parser` for extraction logic

**Implementation**:
```go
type BlockProcessor struct {
    in      <-chan *common.Block
    out     chan<- *types.ProcessedBlock
    workers int
}

func (p *BlockProcessor) Start(ctx context.Context, errCh chan<- error)
```

**Concurrency**: N worker goroutines (configurable, default: 4)

---

### 3. Block Writer (`pkg/blockpipeline/writer.go`)

**Responsibility**: Persist processed blocks to PostgreSQL in batches.

**Key Features**:
- Worker pool of M concurrent writers
- Each worker has dedicated database connection
- Transaction support for atomic writes
- Panic recovery for resilience

**Implementation**:
```go
type BlockWriter struct {
    dbWriter *db.BlockWriter
    in       <-chan *types.ProcessedBlock
}

func (w *BlockWriter) Start(ctx context.Context, errCh chan<- error)
```

**Concurrency**: M worker goroutines (configurable, default: 2)

---

### 4. Parser (`pkg/parser/parser.go`)

**Responsibility**: Extract structured data from Fabric block protobuf messages.

**Parsing Steps**:
1. Extract block header (number, previous hash, data hash)
2. Iterate through block data envelopes
3. Parse channel header for transaction metadata
4. Extract transaction actions and endorsements
5. Parse read/write sets from transaction results
6. Extract namespace policies from `_meta` namespace

**Key Functions**:
```go
func Parse(block *common.Block, blockWriter *db.BlockWriter) (*types.ProcessedBlock, error)
```

**Output**: `ProcessedBlock` with:
- `BlockInfo`: Block metadata
- `[]Transaction`: List of transactions with validation codes
- `[]ReadRecord`: Read set entries (key, version, namespace)
- `[]WriteRecord`: Write set entries (key, value, is_delete)
- `[]NamespacePolicy`: Endorsement policies per namespace

---

### 5. Database Layer (`pkg/db/`)

**Technology**: PostgreSQL with `pgx/v5` driver and SQLC code generation.

**Components**:
- **Connection Pool**: `pgxpool.Pool` for connection management
- **SQLC Queries**: Type-safe query functions generated from SQL
- **Transaction Support**: Batch inserts with rollback on error
- **Test Infrastructure**: `testcontainers` for isolated integration tests

**Schema Overview**:
```sql
blocks (block_num, previous_hash, data_hash, tx_count)
transactions (id, block_num, tx_num, tx_id, validation_code)
reads (tx_id, ns_id, key, version, is_read_write)
writes (tx_id, ns_id, key, value, is_blind_write)
endorsements (tx_id, ns_id, endorsement_bytes, msp_id, identity)
namespace_policies (ns_id, version, policy, block_num)
```

**Indexes**:
- `block_num` (PRIMARY KEY)
- `tx_id` (indexed for fast lookup)
- `ns_id` (indexed for namespace queries)

---

### 6. API Layer (`pkg/api/`)

**REST API** (`handlers.go`, `router.go`):
- Built with standard library `net/http`
- JSON request/response marshaling
- Path parameters with `http.ServeMux`
- Error handling with HTTP status codes

**Endpoints**:
```
GET /blocks/height              → Current blockchain height
GET /blocks/{block_num}         → Block details by number
GET /tx/{tx_id_hex}             → Transaction details by ID
GET /policies/{namespace}       → Namespace endorsement policies
GET /healthz                    → Service health check
```

**gRPC API** (`grpc_server.go`, `proto/explorer.proto`):
- Protocol Buffers for efficient serialization
- Reflection enabled for `grpcurl` testing
- Same query logic as REST, different transport

**Services**:
```protobuf
rpc GetBlockHeight(BlockHeightRequest) returns (BlockHeightResponse)
rpc GetBlock(GetBlockRequest) returns (BlockResponse)
rpc GetTransaction(GetTransactionRequest) returns (TransactionResponse)
rpc GetNamespacePolicies(GetNamespacePoliciesRequest) returns (NamespacePoliciesResponse)
rpc HealthCheck(HealthRequest) returns (HealthResponse)
```

---

## Data Flow

### Block Ingestion Pipeline

```
1. Sidecar Streaming
   ↓
   BlockReceiver.Start()
   ↓
   streamer.StartDeliver(ctx, rawBlockChan)
   
2. Raw Block Channel
   ↓
   rawBlockChan (buffered, size: config.RawChannelSize)
   
3. Block Processing
   ↓
   BlockProcessor workers (N goroutines)
   ↓
   parser.Parse(block) → ProcessedBlock
   
4. Processed Block Channel
   ↓
   processedBlockChan (buffered, size: config.ProcessChannelSize)
   
5. Database Persistence
   ↓
   BlockWriter workers (M goroutines)
   ↓
   db.WriteProcessedBlock(block) → PostgreSQL
   
6. Query API
   ↓
   REST/gRPC handlers
   ↓
   SQLC queries → PostgreSQL
   ↓
   JSON/Protobuf response → Clients
```

### Error Propagation

Errors from each component are sent to a shared `errCh` channel:

```go
errCh := make(chan error, 10)

go receiver.Start(ctx, errCh)
go processor.Start(ctx, errCh)
go writer.Start(ctx, errCh)

select {
case err := <-errCh:
    logger.Error("pipeline error", "error", err)
    cancel() // Trigger graceful shutdown
case <-ctx.Done():
    logger.Info("shutting down gracefully")
}
```

---

## Database Schema

### Tables

#### `blocks`
```sql
CREATE TABLE blocks (
    block_num      BIGINT PRIMARY KEY,
    previous_hash  BYTEA NOT NULL,
    data_hash      BYTEA NOT NULL,
    tx_count       INTEGER NOT NULL,
    created_at     TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

#### `transactions`
```sql
CREATE TABLE transactions (
    id               BIGSERIAL PRIMARY KEY,
    block_num        BIGINT NOT NULL REFERENCES blocks(block_num) ON DELETE CASCADE,
    tx_num           INTEGER NOT NULL,
    tx_id            VARCHAR(128) NOT NULL UNIQUE,
    validation_code  INTEGER NOT NULL,
    created_at       TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_transactions_block_num ON transactions(block_num);
CREATE INDEX idx_transactions_tx_id ON transactions(tx_id);
```

#### `reads`
```sql
CREATE TABLE reads (
    id            BIGSERIAL PRIMARY KEY,
    tx_id         VARCHAR(128) NOT NULL REFERENCES transactions(tx_id) ON DELETE CASCADE,
    ns_id         VARCHAR(255) NOT NULL,
    key           TEXT NOT NULL,
    version       BIGINT,
    is_read_write BOOLEAN DEFAULT FALSE
);

CREATE INDEX idx_reads_tx_id ON reads(tx_id);
CREATE INDEX idx_reads_ns_id ON reads(ns_id);
```

#### `writes`
```sql
CREATE TABLE writes (
    id              BIGSERIAL PRIMARY KEY,
    tx_id           VARCHAR(128) NOT NULL REFERENCES transactions(tx_id) ON DELETE CASCADE,
    ns_id           VARCHAR(255) NOT NULL,
    key             TEXT NOT NULL,
    value           BYTEA,
    is_blind_write  BOOLEAN DEFAULT FALSE,
    read_version    BIGINT
);

CREATE INDEX idx_writes_tx_id ON writes(tx_id);
CREATE INDEX idx_writes_ns_id ON writes(ns_id);
```

#### `endorsements`
```sql
CREATE TABLE endorsements (
    id                 BIGSERIAL PRIMARY KEY,
    tx_id              VARCHAR(128) NOT NULL REFERENCES transactions(tx_id) ON DELETE CASCADE,
    ns_id              VARCHAR(255) NOT NULL,
    endorsement_bytes  BYTEA NOT NULL,
    msp_id             VARCHAR(255),
    identity           TEXT
);

CREATE INDEX idx_endorsements_tx_id ON endorsements(tx_id);
```

#### `namespace_policies`
```sql
CREATE TABLE namespace_policies (
    id         BIGSERIAL PRIMARY KEY,
    ns_id      VARCHAR(255) NOT NULL,
    version    BIGINT NOT NULL,
    policy     JSONB NOT NULL,
    block_num  BIGINT NOT NULL REFERENCES blocks(block_num) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(ns_id, version)
);

CREATE INDEX idx_namespace_policies_ns_id ON namespace_policies(ns_id);
CREATE INDEX idx_namespace_policies_block_num ON namespace_policies(block_num);
```

---

## API Design

### REST API Patterns

**Success Response**:
```json
{
  "block_num": 42,
  "tx_count": 3,
  "transactions": [...]
}
```

**Error Response**:
```json
{
  "error": "block not found"
}
```

**Status Codes**:
- `200 OK`: Successful query
- `400 Bad Request`: Invalid parameters
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Database or system error

### gRPC API Patterns

**Request/Response Flow**:
```protobuf
message GetBlockRequest {
  int64 block_num = 1;
}

message BlockResponse {
  int64 block_num = 1;
  int32 tx_count = 2;
  repeated TransactionWithWrites transactions = 5;
}
```

**Error Handling**:
- Uses gRPC status codes (`codes.NotFound`, `codes.InvalidArgument`)
- Detailed error messages in status descriptions

---

## Concurrency Model

### Goroutine Lifecycle

```go
// Server orchestrates all components
func (s *Server) Start(ctx context.Context) error {
    errCh := make(chan error, 10)
    
    // Start receiver (1 goroutine)
    go s.receiver.Start(ctx, errCh)
    
    // Start processor pool (N goroutines)
    go s.processor.Start(ctx, errCh)
    
    // Start writer pool (M goroutines)
    go s.writer.Start(ctx, errCh)
    
    // Wait for error or shutdown
    select {
    case err := <-errCh:
        return err
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

### Channel Buffer Sizes

Configured in `config.yaml`:

```yaml
buffers:
  raw_channel_size: 100       # Raw blocks from sidecar
  process_channel_size: 50    # Processed blocks to DB
  receiver_channel_size: 100  # Internal receiver buffer
```

**Tuning Guidelines**:
- Larger buffers → Better throughput, higher memory usage
- Smaller buffers → Lower memory, potential blocking
- Recommended: Buffer size = 2-3x worker count

---

## Error Handling

### Error Categories

1. **Transient Errors**: Network timeouts, temporary DB unavailability
   - **Strategy**: Exponential backoff with retry

2. **Permanent Errors**: Invalid block format, constraint violations
   - **Strategy**: Log error, skip block, continue processing

3. **Fatal Errors**: Configuration errors, missing dependencies
   - **Strategy**: Fail fast, exit application

### Panic Recovery

Database writers use `defer/recover` for resilience:

```go
defer func() {
    if r := recover(); r != nil {
        logger.Error("writer panic", "panic", r)
        errCh <- fmt.Errorf("writer panic: %v", r)
    }
}()
```

### Graceful Shutdown

```go
// SIGTERM/SIGINT handler
signalCh := make(chan os.Signal, 1)
signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)

go func() {
    <-signalCh
    logger.Info("shutdown signal received")
    cancel() // Cancel context for all goroutines
}()
```

---

## Testing Strategy

### Test Coverage: 75.5%

### Test Pyramid

```
        ┌─────────────┐
        │ Integration │  (API, DB, Pipeline tests)
        │   Tests     │
        └─────────────┘
              /\
             /  \
            /    \
           /      \
          /  Unit  \     (Parser, Config, Utils tests)
         /   Tests  \
        /            \
       └──────────────┘
```

### Test Infrastructure

**Unit Tests**:
- Pure functions (parser, config, utilities)
- Mock-free where possible
- Table-driven tests for comprehensive coverage

**Integration Tests**:
- `testcontainers` for PostgreSQL
- Real database transactions
- HTTP/gRPC client tests

**End-to-End Tests**:
- Docker Compose stack
- Real Fabric sidecar (manual testing)

### Running Tests

```bash
# All tests with database
make test-all

# Tests without database
make test-no-db

# Coverage report
make coverage

# Specific package
go test ./pkg/parser/... -v
```

### Test Organization

```
pkg/parser/parser_test.go        # Unit tests for parser
pkg/db/db_writer_test.go         # Integration tests with DB
pkg/api/handlers_test.go         # HTTP handler tests
pkg/blockpipeline/*_test.go      # Pipeline component tests
```

---

## Performance Characteristics

### Throughput

**Configuration**:
- Processor Workers: 4
- Writer Workers: 2
- Buffer Sizes: 100 (raw), 50 (processed)

**Observed Performance**:
- ~50-100 blocks/second (depends on transaction count)
- ~200-500 transactions/second
- Database write latency: ~10-50ms per block

### Optimization Opportunities

1. **Batch Database Writes**: Group multiple blocks into single transaction
2. **Connection Pooling**: Increase `max_connections` for higher concurrency
3. **Index Tuning**: Add indexes for frequent queries (e.g., `ns_id + key`)
4. **Compression**: Enable PostgreSQL TOAST compression for large writesets

---

## Deployment Considerations

### Environment Variables

Critical configuration can be overridden:

```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=explorer
DB_PASSWORD=secret
SIDECAR_HOST=fabric-peer
SIDECAR_PORT=7052
LOG_LEVEL=info
```

### Health Checks

```yaml
# Kubernetes liveness probe
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
```

### Resource Requirements

**Minimum**:
- CPU: 2 cores
- Memory: 2 GB RAM
- Disk: 50 GB (database grows with blockchain size)

**Recommended**:
- CPU: 4 cores
- Memory: 4 GB RAM
- Disk: 500 GB SSD

---

## Future Enhancements

1. **Metrics & Monitoring**: Prometheus metrics for pipeline throughput
2. **Block Indexing**: Full-text search on writeset keys/values
3. **Event Streaming**: Kafka/NATS integration for real-time notifications
4. **GraphQL API**: Alternative query interface
5. **Multi-Channel Support**: Index multiple Fabric channels simultaneously
6. **State Snapshots**: Periodic state dumps for disaster recovery

---

## References

- **Hyperledger Fabric**: https://hyperledger-fabric.readthedocs.io/
- **SQLC**: https://docs.sqlc.dev/
- **Protocol Buffers**: https://protobuf.dev/
- **PostgreSQL**: https://www.postgresql.org/docs/

---

**Last Updated**: February 18, 2026  
**Version**: 1.0.0  
**Maintainers**: LF Decentralized Trust Labs
