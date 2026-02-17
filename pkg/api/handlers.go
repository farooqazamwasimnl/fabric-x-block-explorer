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

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/constants"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/logging"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/util"
	"github.com/jackc/pgx/v5/pgxpool"
)

type API struct {
	q    *dbsqlc.Queries
	pool *pgxpool.Pool
}

var logger = logging.New("api")

func NewAPI(pool *pgxpool.Pool) *API {
	return &API{
		q:    dbsqlc.New(pool),
		pool: pool,
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

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
	blockNum, err := strconv.ParseInt(blockNumStr, 10, 64)
	if err != nil {
		logger.Warnf("invalid block number: %v", err)
		writeError(w, "invalid block number", http.StatusBadRequest)
		return
	}

	limitTx := parseInt(r, "limitTx", constants.DefaultTxLimit)
	offsetTx := parseInt(r, "offsetTx", constants.DefaultTxOffset)
	limitWrites := parseInt(r, "limitWrites", constants.DefaultWriteLimit)
	offsetWrites := parseInt(r, "offsetWrites", constants.DefaultWriteOffset)

	block, err := a.q.GetBlock(r.Context(), blockNum)
	if err != nil {
		logger.Errorf("failed to get block %d: %v", blockNum, err)
		writeError(w, err.Error(), http.StatusNotFound)
		return
	}

	txs, err := a.q.GetTransactionsByBlock(r.Context(), dbsqlc.GetTransactionsByBlockParams{
		BlockNum: blockNum,
		Limit:    int32(limitTx),
		Offset:   int32(offsetTx),
	})
	if err != nil {
		logger.Errorf("failed to get transactions for block %d: %v", blockNum, err)
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
		txResp := a.buildTransactionResponse(r.Context(), tx, limitWrites, offsetWrites)
		resp.Transactions = append(resp.Transactions, txResp)
	}

	writeJSON(w, resp)
}

func (a *API) GetTxByID(w http.ResponseWriter, r *http.Request) {
	txHex := r.PathValue("tx_id_hex")
	txBytes, err := hex.DecodeString(txHex)
	if err != nil {
		logger.Warnf("invalid tx_id hex: %v", err)
		writeError(w, "invalid tx_id hex", http.StatusBadRequest)
		return
	}

	tx, err := a.q.GetTransactionByTxID(r.Context(), txBytes)
	if err != nil {
		logger.Warnf("transaction %s not found: %v", txHex, err)
		writeError(w, "not found", http.StatusNotFound)
		return
	}

	block, err := a.q.GetBlock(r.Context(), tx.BlockNum)
	if err != nil {
		logger.Errorf("failed to get block %d for tx %s: %v", tx.BlockNum, txHex, err)
		writeError(w, "internal error", http.StatusInternalServerError)
		return
	}

	resp := types.TxWithBlockResponse{
		Transaction: a.buildTransactionResponse(r.Context(), tx, 1000, 0),
		Block: types.BlockHeaderOnly{
			BlockNum:     block.BlockNum,
			TxCount:      block.TxCount,
			PreviousHash: hex.EncodeToString(block.PreviousHash),
			DataHash:     hex.EncodeToString(block.DataHash),
		},
	}

	writeJSON(w, resp)
}

func (a *API) buildTransactionResponse(ctx context.Context, tx dbsqlc.Transaction, limitWrites, offsetWrites int) types.TransactionWithWriteSets {
	reads, _ := a.q.GetReadsByTx(ctx, dbsqlc.GetReadsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	endorsements, _ := a.q.GetEndorsementsByTx(ctx, dbsqlc.GetEndorsementsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	writes, _ := a.q.GetWritesByTx(ctx, dbsqlc.GetWritesByTxParams{
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
		txResp.Reads = append(txResp.Reads, types.ReadRecordResponse{
			ID:          rrec.ID,
			NsID:        rrec.NsID,
			Key:         hex.EncodeToString(rrec.Key),
			Version:     util.NullableInt64ToPtr(rrec.Version),
			IsReadWrite: rrec.IsReadWrite,
		})
	}

	for _, wrec := range writes {
		txResp.Writes = append(txResp.Writes, types.WriteRecordResponse{
			ID:           wrec.ID,
			NsID:         wrec.NsID,
			Key:          hex.EncodeToString(wrec.Key),
			Value:        hex.EncodeToString(wrec.Value),
			IsBlindWrite: wrec.IsBlindWrite,
			ReadVersion:  util.NullableInt64ToPtr(wrec.ReadVersion),
		})
	}

	for _, erec := range endorsements {
		var identity json.RawMessage
		if len(erec.Identity) > 0 {
			identity = json.RawMessage(erec.Identity)
		}
		txResp.Endorsements = append(txResp.Endorsements, types.EndorsementResponse{
			ID:          erec.ID,
			NsID:        erec.NsID,
			Endorsement: hex.EncodeToString(erec.Endorsement),
			MspID:       util.NullableStringToPtr(erec.MspID),
			Identity:    identity,
		})
	}

	return txResp
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

// GetNamespacePolicies returns policy versions for a namespace.
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
			Policy:    json.RawMessage(row.Policy),
		})
		if latest {
			break
		}
	}

	writeJSON(w, resp)
}

// GetBlockHeightValue returns the current block height as an int64.
func (a *API) GetBlockHeightValue(ctx context.Context) (int64, error) {
	h, err := a.q.GetBlockHeight(ctx)
	if err != nil {
		return 0, err
	}
	return h.(int64), nil
}

type HealthResponse struct {
	Status  string `json:"status"`
	Details string `json:"details,omitempty"`
}

// HealthHandler implements a combined liveness/readiness check.
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
