/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
)

// BlockWriter writes processed blocks and their writes/transactions to the DB.
// It supports being constructed from either a *pgxpool.Pool (shared pool) or a
// dedicated *pgxpool.Conn (per-writer dedicated connection).
type BlockWriter struct {
	pool *pgxpool.Pool
	conn *pgxpool.Conn
}

// NewBlockWriter constructs a BlockWriter that uses the provided *pgxpool.Pool.
func NewBlockWriter(pool *pgxpool.Pool) *BlockWriter {
	return &BlockWriter{pool: pool}
}

// NewBlockWriterFromConn constructs a BlockWriter that uses the provided *pgxpool.Conn.
// This is useful when each writer goroutine should use its own dedicated DB connection.
func NewBlockWriterFromConn(conn *pgxpool.Conn) *BlockWriter {
	return &BlockWriter{conn: conn}
}

// WriteProcessedBlock persists a processed block and its write records in a single transaction.
// It begins a transaction on the underlying connection (db or conn), uses sqlc-generated
// queries bound to that transaction, and commits or rolls back on error.
func (bw *BlockWriter) WriteProcessedBlock(ctx context.Context, pb *types.ProcessedBlock) error {
	if pb == nil {
		return errors.New("processed block is nil")
	}

	// Extract parsed data from pb.Data
	parsedData, ok := pb.Data.(*types.ParsedBlockData)
	if !ok {
		return errors.New("processed block Data is not *types.ParsedBlockData")
	}
	writes := parsedData.Writes
	reads := parsedData.Reads
	txNamespaces := parsedData.TxNamespaces
	endorsements := parsedData.Endorsements

	var (
		tx  pgx.Tx
		err error
	)
	if bw.conn != nil {
		tx, err = bw.conn.Begin(ctx)
	} else if bw.pool != nil {
		tx, err = bw.pool.Begin(ctx)
	} else {
		return errors.New("no pool or conn available in BlockWriter")
	}
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	q := dbsqlc.New(tx)

	if err := q.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     int64(pb.BlockInfo.Number),
		TxCount:      int32(pb.Txns),
		PreviousHash: pb.BlockInfo.PreviousHash,
		DataHash:     pb.BlockInfo.DataHash,
	}); err != nil {
		return err
	}

	// Cache transaction IDs and tx_namespace IDs
	txIDCache := make(map[string]int64)
	txNsCache := make(map[string]int64)

	// Insert all transactions first (some may not have writes)
	for _, txNs := range txNamespaces {
		txKey := fmt.Sprintf("%d-%d", txNs.BlockNum, txNs.TxNum)
		if _, found := txIDCache[txKey]; !found {
			txID, err := q.InsertTransaction(ctx, dbsqlc.InsertTransactionParams{
				BlockNum:       int64(txNs.BlockNum),
				TxNum:          int64(txNs.TxNum),
				TxID:           []byte(txNs.TxID),
				ValidationCode: int64(txNs.ValidationCode),
			})
			if err != nil {
				return err
			}
			txIDCache[txKey] = txID
		}
	}

	// Insert transaction-namespace relationships
	for _, txNs := range txNamespaces {
		txKey := fmt.Sprintf("%d-%d", txNs.BlockNum, txNs.TxNum)
		txID := txIDCache[txKey]

		txNsID, err := q.InsertTxNamespace(ctx, dbsqlc.InsertTxNamespaceParams{
			TransactionID: txID,
			NsID:          txNs.NsID,
			NsVersion:     int64(txNs.NsVersion),
		})
		if err != nil {
			return err
		}

		txNsKey := fmt.Sprintf("%d-%d-%s", txNs.BlockNum, txNs.TxNum, txNs.NsID)
		txNsCache[txNsKey] = txNsID
	}

	// Insert reads
	for _, r := range reads {
		txNsKey := fmt.Sprintf("%d-%d-%s", r.BlockNum, r.TxNum, r.NsID)
		txNsID := txNsCache[txNsKey]

		var version pgtype.Int8
		if r.Version != nil {
			version.Int64 = int64(*r.Version)
			version.Valid = true
		}

		if err := q.InsertTxRead(ctx, dbsqlc.InsertTxReadParams{
			TxNamespaceID: txNsID,
			Key:           []byte(r.Key),
			Version:       version,
			IsReadWrite:   r.IsReadWrite,
		}); err != nil {
			return err
		}
	}

	// Insert endorsements
	for _, e := range endorsements {
		txNsKey := fmt.Sprintf("%d-%d-%s", e.BlockNum, e.TxNum, e.NsID)
		txNsID := txNsCache[txNsKey]

		var mspID pgtype.Text
		if e.MspID != nil {
			mspID.String = *e.MspID
			mspID.Valid = true
		}

		if err := q.InsertTxEndorsement(ctx, dbsqlc.InsertTxEndorsementParams{
			TxNamespaceID: txNsID,
			Endorsement:   e.Endorsement,
			MspID:         mspID,
			Identity:      e.Identity,
		}); err != nil {
			return err
		}
	}

	// Insert writes to tx_writes table
	for _, w := range writes {
		txNsKey := fmt.Sprintf("%d-%d-%s", w.BlockNum, w.TxNum, w.Namespace)
		txNsID := txNsCache[txNsKey]

		var readVersion pgtype.Int8
		if w.ReadVersion != nil {
			readVersion.Int64 = int64(*w.ReadVersion)
			readVersion.Valid = true
		}

		if err := q.InsertTxWrite(ctx, dbsqlc.InsertTxWriteParams{
			TxNamespaceID: txNsID,
			Key:           []byte(w.Key),
			Value:         w.Value,
			IsBlindWrite:  w.IsBlindWrite,
			ReadVersion:   readVersion,
		}); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true

	log.Printf("db: stored block %d with %d writes, %d reads", pb.BlockInfo.Number, len(writes), len(reads))
	return nil
}
