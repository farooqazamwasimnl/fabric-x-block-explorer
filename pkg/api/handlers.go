/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
)

// API exposes database-backed HTTP handlers.
type API struct {
	q    *dbsqlc.Queries
	pool *pgxpool.Pool
}

// NewAPI constructs an API instance from a *pgxpool.Pool.
func NewAPI(pool *pgxpool.Pool) *API {
	return &API{
		q:    dbsqlc.New(pool),
		pool: pool,
	}
}

// writeJSON writes v as JSON to the ResponseWriter and sets Content-Type.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes an HTTP error with a message and status code.
func writeError(w http.ResponseWriter, msg string, code int) {
	http.Error(w, msg, code)
}

func (a *API) GetBlockHeight(w http.ResponseWriter, r *http.Request) {
	height, err := a.q.GetBlockHeight(r.Context())
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h := height.(int64)

	writeJSON(w, map[string]int64{"height": h})
}

func (a *API) GetBlockByNumber(w http.ResponseWriter, r *http.Request) {
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
		reads, _ := a.q.GetReadsByTx(r.Context(), dbsqlc.GetReadsByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
		})

		endorsements, _ := a.q.GetEndorsementsByTx(r.Context(), dbsqlc.GetEndorsementsByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
		})

		writes, _ := a.q.GetWritesByTx(r.Context(), dbsqlc.GetWritesByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
			Limit:    int32(limitWrites),
			Offset:   int32(offsetWrites),
		})

		txResp := types.TransactionWithWriteSets{
			ID:             tx.ID,
			TxNum:          tx.TxNum,
			TxID:           hex.EncodeToString(tx.TxID),
			ValidationCode: tx.ValidationCode,
		}

		for _, rrec := range reads {
			var version *int64
			if rrec.Version.Valid {
				version = &rrec.Version.Int64
			}
			txResp.Reads = append(txResp.Reads, types.ReadRecordResponse{
				ID:          rrec.ID,
				NsID:        rrec.NsID,
				Key:         hex.EncodeToString(rrec.Key),
				Version:     version,
				IsReadWrite: rrec.IsReadWrite,
			})
		}

		for _, wrec := range writes {
			var readVersion *int64
			if wrec.ReadVersion.Valid {
				readVersion = &wrec.ReadVersion.Int64
			}
			txResp.Writes = append(txResp.Writes, types.WriteRecordResponse{
				ID:           wrec.ID,
				NsID:         wrec.NsID,
				Key:          hex.EncodeToString(wrec.Key),
				Value:        hex.EncodeToString(wrec.Value),
				IsBlindWrite: wrec.IsBlindWrite,
				ReadVersion:  readVersion,
			})
		}

		for _, erec := range endorsements {
			var mspID *string
			if erec.MspID.Valid {
				mspID = &erec.MspID.String
			}
			var identity json.RawMessage
			if len(erec.Identity) > 0 {
				identity = json.RawMessage(erec.Identity)
			}
			txResp.Endorsements = append(txResp.Endorsements, types.EndorsementResponse{
				ID:          erec.ID,
				NsID:        erec.NsID,
				Endorsement: hex.EncodeToString(erec.Endorsement),
				MspID:       mspID,
				Identity:    identity,
			})
		}

		resp.Transactions = append(resp.Transactions, txResp)
	}

	writeJSON(w, resp)
}

func (a *API) GetTxByID(w http.ResponseWriter, r *http.Request) {
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

	reads, _ := a.q.GetReadsByTx(r.Context(), dbsqlc.GetReadsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	endorsements, _ := a.q.GetEndorsementsByTx(r.Context(), dbsqlc.GetEndorsementsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	writes, _ := a.q.GetWritesByTx(r.Context(), dbsqlc.GetWritesByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
		Limit:    1000,
		Offset:   0,
	})

	resp := types.TxWithBlockResponse{
		Transaction: types.TransactionWithWriteSets{
			ID:             tx.ID,
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

	for _, rrec := range reads {
		var version *int64
		if rrec.Version.Valid {
			version = &rrec.Version.Int64
		}
		resp.Transaction.Reads = append(resp.Transaction.Reads, types.ReadRecordResponse{
			ID:          rrec.ID,
			NsID:        rrec.NsID,
			Key:         hex.EncodeToString(rrec.Key),
			Version:     version,
			IsReadWrite: rrec.IsReadWrite,
		})
	}

	for _, wrec := range writes {
		var readVersion *int64
		if wrec.ReadVersion.Valid {
			readVersion = &wrec.ReadVersion.Int64
		}
		resp.Transaction.Writes = append(resp.Transaction.Writes, types.WriteRecordResponse{
			ID:           wrec.ID,
			NsID:         wrec.NsID,
			Key:          hex.EncodeToString(wrec.Key),
			Value:        hex.EncodeToString(wrec.Value),
			IsBlindWrite: wrec.IsBlindWrite,
			ReadVersion:  readVersion,
		})
	}

	for _, erec := range endorsements {
		var mspID *string
		if erec.MspID.Valid {
			mspID = &erec.MspID.String
		}
		var identity json.RawMessage
		if len(erec.Identity) > 0 {
			identity = json.RawMessage(erec.Identity)
		}
		resp.Transaction.Endorsements = append(resp.Transaction.Endorsements, types.EndorsementResponse{
			ID:          erec.ID,
			NsID:        erec.NsID,
			Endorsement: hex.EncodeToString(erec.Endorsement),
			MspID:       mspID,
			Identity:    identity,
		})
	}

	writeJSON(w, resp)
}

// GetNamespacePolicies returns policy versions for a namespace.
// Optional query param: latest=true to return only the most recent policy.
func (a *API) GetNamespacePolicies(w http.ResponseWriter, r *http.Request) {
	ns := r.PathValue("namespace")
	latest := r.URL.Query().Get("latest") == "true"

	rows, err := a.q.GetNamespacePolicies(r.Context(), ns)
	if err != nil {
		writeError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := []types.NamespacePolicyResponse{}
	for _, row := range rows {
		resp = append(resp, types.NamespacePolicyResponse{
			ID:        row.ID,
			Namespace: row.Namespace,
			Version:   row.Version,
			Policy:    hex.EncodeToString(row.Policy),
		})
		if latest {
			break
		}
	}

	writeJSON(w, resp)
}


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
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if a.pool != nil {
		if err := a.pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			writeJSON(w, HealthResponse{
				Status:  "unavailable",
				Details: "db ping failed: " + err.Error(),
			})
			return
		}
	}

	writeJSON(w, HealthResponse{Status: "ok"})
}
