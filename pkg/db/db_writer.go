/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"context"
	"database/sql"
	"errors"
	"log"

	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
)

// BlockWriter writes processed blocks and their writes/transactions to the DB.
// It supports being constructed from either a *sql.DB (shared pool) or a
// dedicated *sql.Conn (per-writer dedicated connection).
type BlockWriter struct {
	db   *sql.DB
	conn *sql.Conn
}

// NewBlockWriter constructs a BlockWriter that uses the provided *sql.DB.
func NewBlockWriter(db *sql.DB) *BlockWriter {
	return &BlockWriter{db: db}
}

// NewBlockWriterFromConn constructs a BlockWriter that uses the provided *sql.Conn.
// This is useful when each writer goroutine should use its own dedicated DB connection.
func NewBlockWriterFromConn(conn *sql.Conn) *BlockWriter {
	return &BlockWriter{conn: conn}
}

// WriteProcessedBlock persists a processed block and its write records in a single transaction.
// It begins a transaction on the underlying connection (db or conn), uses sqlc-generated
// queries bound to that transaction, and commits or rolls back on error.
func (bw *BlockWriter) WriteProcessedBlock(ctx context.Context, pb *types.ProcessedBlock) error {
	if pb == nil {
		return errors.New("processed block is nil")
	}

	// Extract writes from pb.Data
	writes, ok := pb.Data.([]types.WriteRecord)
	if !ok {
		return errors.New("processed block Data is not []types.WriteRecord")
	}

	// Choose where to begin a transaction: prefer dedicated conn if present.
	var (
		tx  *sql.Tx
		err error
	)
	if bw.conn != nil {
		tx, err = bw.conn.BeginTx(ctx, nil)
	} else if bw.db != nil {
		tx, err = bw.db.BeginTx(ctx, nil)
	} else {
		return errors.New("no db or conn available in BlockWriter")
	}
	if err != nil {
		return err
	}

	// Ensure rollback on error
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	// Use sqlc queries bound to the transaction
	q := dbsqlc.New(tx)

	// Insert block header
	if err := q.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     int64(pb.BlockInfo.Number),
		TxCount:      int32(pb.Txns),
		PreviousHash: []byte(pb.BlockInfo.PreviousHash),
		DataHash:     []byte(pb.BlockInfo.DataHash),
	}); err != nil {
		return err
	}

	// Cache namespace -> id to avoid repeated upserts within the same block
	nsCache := make(map[string]int64)

	for _, w := range writes {
		// Upsert namespace (BYTEA)
		nsID, found := nsCache[w.Namespace]
		if !found {
			id, err := q.UpsertNamespace(ctx, []byte(w.Namespace))
			if err != nil {
				return err
			}
			nsID = id
			nsCache[w.Namespace] = id
		}

		// Insert transaction (BYTEA)
		if err := q.InsertTransaction(ctx, dbsqlc.InsertTransactionParams{
			BlockNum:       int64(w.BlockNum),
			TxNum:          int64(w.TxNum),
			TxID:           []byte(w.TxID),
			ValidationCode: int64(w.ValidationCode),
		}); err != nil {
			return err
		}

		// Insert write (BYTEA)
		if err := q.InsertWrite(ctx, dbsqlc.InsertWriteParams{
			NamespaceID: nsID,
			BlockNum:    int64(w.BlockNum),
			TxNum:       int64(w.TxNum),
			TxID:        []byte(w.TxID),
			Key:         []byte(w.Key),
			Value:       w.Value,
		}); err != nil {
			return err
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true

	// Log using the block number from BlockInfo for consistency
	log.Printf("db: stored block %d with %d writes", pb.BlockInfo.Number, len(writes))
	return nil
}
