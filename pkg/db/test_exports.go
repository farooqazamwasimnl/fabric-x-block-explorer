/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"context"
	_ "embed"
	"testing"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/dbtest"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

//go:embed schema.sql
var schemaSQL string

// DatabaseTestEnv provides a test environment for database operations.
// This follows the fabric-x-committer pattern of exposing test helpers
// through test_exports.go.
type DatabaseTestEnv struct {
	Pool    *pgxpool.Pool
	Queries *dbsqlc.Queries
	tc      *dbtest.TestContainer
}

// NewDatabaseTestEnv creates a new test environment with a PostgreSQL testcontainer.
// The schema is automatically initialized, and cleanup is registered with t.Cleanup().
func NewDatabaseTestEnv(t *testing.T) *DatabaseTestEnv {
	t.Helper()

	// Create testcontainer
	tc := dbtest.PrepareTestEnv(t)

	// Initialize schema
	ctx := context.Background()
	_, err := tc.Pool.Exec(ctx, schemaSQL)
	require.NoError(t, err, "failed to initialize database schema")

	// Create queries
	queries := dbsqlc.New(tc.Pool)

	env := &DatabaseTestEnv{
		Pool:    tc.Pool,
		Queries: queries,
		tc:      tc,
	}

	// Register cleanup
	t.Cleanup(func() {
		tc.Close(t)
	})

	return env
}

// AssertBlockExists verifies that a block exists in the database
func (env *DatabaseTestEnv) AssertBlockExists(t *testing.T, blockNum int64) {
	t.Helper()

	ctx := context.Background()
	block, err := env.Queries.GetBlock(ctx, blockNum)
	require.NoError(t, err, "block %d should exist", blockNum)
	require.Equal(t, blockNum, block.BlockNum)
}

// AssertBlockNotExists verifies that a block does not exist in the database
func (env *DatabaseTestEnv) AssertBlockNotExists(t *testing.T, blockNum int64) {
	t.Helper()

	ctx := context.Background()
	_, err := env.Queries.GetBlock(ctx, blockNum)
	require.Error(t, err, "block %d should not exist", blockNum)
}

// GetBlockCount returns the total number of blocks in the database
func (env *DatabaseTestEnv) GetBlockCount(t *testing.T) int64 {
	t.Helper()

	ctx := context.Background()
	var count int64
	err := env.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM blocks").Scan(&count)
	require.NoError(t, err, "failed to count blocks")
	return count
}

// GetTransactionCount returns the total number of transactions in the database
func (env *DatabaseTestEnv) GetTransactionCount(t *testing.T) int64 {
	t.Helper()

	ctx := context.Background()
	var count int64
	err := env.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions").Scan(&count)
	require.NoError(t, err, "failed to count transactions")
	return count
}
