/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"context"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
)

// API exposes database-backed HTTP handlers.
type API struct {
	q  *dbsqlc.Queries
	db *sql.DB
}

// NewAPI constructs an API instance from a *sql.DB.
func NewAPI(db *sql.DB) *API {
	return &API{
		q:  dbsqlc.New(db),
		db: db,
	}
}

// writeJSON writes v as JSON to the ResponseWriter and sets Content-Type.
func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an HTTP error with a message and status code.
func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}

//
// ------------------------------------------------------------
// GET /blocks/height
// ------------------------------------------------------------
//

func (a *API) GetBlockHeight(w http.ResponseWriter, r *http.Request) {
	height, err := a.q.GetBlockHeight(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// sqlc returns interface{} because of COALESCE
	h := height.(int64)

	writeJSON(w, map[string]int64{"height": h})
}

//
// ------------------------------------------------------------
// GET /blocks/{block_num}
// ------------------------------------------------------------
//

func (a *API) GetBlockByNumber(w http.ResponseWriter, r *http.Request) {
	// Note: r.PathValue is assumed to be provided by your router; keep as-is.
	blockNumStr := r.PathValue("block_num")
	blockNum, _ := strconv.ParseInt(blockNumStr, 10, 64)

	limitTx := parseInt(r, "limitTx", 100)
	offsetTx := parseInt(r, "offsetTx", 0)
	limitWrites := parseInt(r, "limitWrites", 1000)
	offsetWrites := parseInt(r, "offsetWrites", 0)

	block, err := a.q.GetBlock(r.Context(), blockNum)
	if err != nil {
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	txs, err := a.q.GetTransactionsByBlock(r.Context(), dbsqlc.GetTransactionsByBlockParams{
		BlockNum: blockNum,
		Limit:    int32(limitTx),
		Offset:   int32(offsetTx),
	})
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := types.BlockResponse{
		BlockNum:     block.BlockNum,
		TxCount:      block.TxCount,
		PreviousHash: hex.EncodeToString(block.PreviousHash),
		DataHash:     hex.EncodeToString(block.DataHash),
	}

	for _, tx := range txs {
		writes, _ := a.q.GetWritesByTx(r.Context(), dbsqlc.GetWritesByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
			Limit:    int32(limitWrites),
			Offset:   int32(offsetWrites),
		})

		txResp := types.TransactionWithWriteSets{
			ID:             tx.ID,
			BlockNum:       tx.BlockNum,
			TxNum:          tx.TxNum,
			TxID:           hex.EncodeToString(tx.TxID),
			ValidationCode: tx.ValidationCode,
		}

		for _, wrec := range writes {
			txResp.Writes = append(txResp.Writes, types.WriteRecordResponse{
				ID:          wrec.ID,
				NamespaceID: wrec.NamespaceID,
				BlockNum:    wrec.BlockNum,
				TxNum:       wrec.TxNum,
				TxID:        hex.EncodeToString(wrec.TxID),
				Key:         hex.EncodeToString(wrec.Key),
				Value:       hex.EncodeToString(wrec.Value),
				IsDelete:    wrec.IsDelete,
			})
		}

		resp.Transactions = append(resp.Transactions, txResp)
	}

	writeJSON(w, resp)
}

//
// ------------------------------------------------------------
// GET /tx/{tx_id_hex}
// ------------------------------------------------------------
//

func (a *API) GetTxByID(w http.ResponseWriter, r *http.Request) {
	// Note: r.PathValue is assumed to be provided by your router; keep as-is.
	txHex := r.PathValue("tx_id_hex")
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		writeError(w, "invalid tx_id hex", http.StatusBadRequest)
		return
	}

	tx, err := a.q.GetTransactionByTxID(r.Context(), txBytes)
	if err != nil {
		writeError(w, "not found", http.StatusNotFound)
		return
	}

	block, _ := a.q.GetBlock(r.Context(), tx.BlockNum)

	writes, _ := a.q.GetWritesByTx(r.Context(), dbsqlc.GetWritesByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
		Limit:    1000,
		Offset:   0,
	})

	resp := types.TxWithBlockResponse{
		Transaction: types.TransactionWithWriteSets{
			ID:             tx.ID,
			BlockNum:       tx.BlockNum,
			TxNum:          tx.TxNum,
			TxID:           hex.EncodeToString(tx.TxID),
			ValidationCode: tx.ValidationCode,
		},
		Block: types.BlockHeaderOnly{
			BlockNum:     block.BlockNum,
			TxCount:      block.TxCount,
			PreviousHash: hex.EncodeToString(block.PreviousHash),
			DataHash:     hex.EncodeToString(block.DataHash),
		},
	}

	for _, wrec := range writes {
		resp.Transaction.Writes = append(resp.Transaction.Writes, types.WriteRecordResponse{
			ID:          wrec.ID,
			NamespaceID: wrec.NamespaceID,
			BlockNum:    wrec.BlockNum,
			TxNum:       wrec.TxNum,
			TxID:        hex.EncodeToString(wrec.TxID),
			Key:         hex.EncodeToString(wrec.Key),
			Value:       hex.EncodeToString(wrec.Value),
			IsDelete:    wrec.IsDelete,
		})
	}

	writeJSON(w, resp)
}

//
// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------
//

func parseInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// GetBlockHeightValue returns the current block height as an int64.
func (a *API) GetBlockHeightValue(ctx context.Context) (int64, error) {
	h, err := a.q.GetBlockHeight(ctx)
	if err != nil {
		return 0, err
	}
	// sqlc returns interface{} because of COALESCE; assert to int64
	height := h.(int64)
	return height, nil
}

// HealthResponse is the JSON payload returned by the health endpoint.
type HealthResponse struct {
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// HealthHandler implements a combined liveness/readiness check.
// - Liveness: returns 200 if the process is running.
// - Readiness: attempts a short DB ping; if DB is unreachable returns 503.
func (a *API) HealthHandler(w http.ResponseWriter, r *http.Request) {
	// Liveness: process is alive if this handler runs.
	// Readiness: check DB connectivity with a short timeout.
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()

	// If API has a DB handle, try pinging it. If not, treat as ready.
	if a.db != nil {
		if err := a.db.PingContext(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			writeJSON(w, HealthResponse{
				Status:  "unavailable",
				Details: "db ping failed: " + err.Error(),
			})
			return
		}
	}

	// Ready
	writeJSON(w, HealthResponse{Status: "ok"})
}
