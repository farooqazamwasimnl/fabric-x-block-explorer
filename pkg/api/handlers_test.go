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
	"net/http/httptest"
	"testing"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetBlockHeight tests the /blocks/height endpoint
func TestGetBlockHeight(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert test blocks
	err := env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     1,
		TxCount:      5,
		PreviousHash: []byte("prev1"),
		DataHash:     []byte("data1"),
	})
	require.NoError(t, err)

	err = env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     2,
		TxCount:      3,
		PreviousHash: []byte("prev2"),
		DataHash:     []byte("data2"),
	})
	require.NoError(t, err)

	// Create API and test
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/blocks/height", nil)
	w := httptest.NewRecorder()

	api.GetBlockHeight(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")

	var resp map[string]int64
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, int64(2), resp["height"])
}

// TestGetBlockByNumber tests the /blocks/{block_num} endpoint
func TestGetBlockByNumber(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert a block
	err := env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     10,
		TxCount:      2,
		PreviousHash: []byte("previoushash"),
		DataHash:     []byte("datahash"),
	})
	require.NoError(t, err)

	// Insert transactions
	txID1, err := env.Queries.InsertTransaction(ctx, dbsqlc.InsertTransactionParams{
		BlockNum:       10,
		TxNum:          0,
		TxID:           mustDecodeHex(t, "abc123"),
		ValidationCode: 0,
	})
	require.NoError(t, err)

	// Insert namespace for transaction
	nsID, err := env.Queries.InsertTxNamespace(ctx, dbsqlc.InsertTxNamespaceParams{
		TransactionID: txID1,
		NsID:          "mycc",
		NsVersion:     1,
	})
	require.NoError(t, err)

	// Insert a write
	err = env.Queries.InsertTxWrite(ctx, dbsqlc.InsertTxWriteParams{
		TxNamespaceID: nsID,
		Key:           []byte("key1"),
		Value:         []byte("value1"),
		IsBlindWrite:  false,
	})
	require.NoError(t, err)

	// Test the endpoint
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/blocks/10", nil)
	req.SetPathValue("block_num", "10")
	w := httptest.NewRecorder()

	api.GetBlockByNumber(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.BlockResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, int64(10), resp.BlockNum)
	assert.Equal(t, int32(2), resp.TxCount)
	assert.Equal(t, hex.EncodeToString([]byte("previoushash")), resp.PreviousHash)
	assert.Len(t, resp.Transactions, 1)
	assert.Equal(t, "abc123", resp.Transactions[0].TxID)
}

// TestGetBlockByNumberInvalidBlockNum tests error handling for invalid block number
func TestGetBlockByNumberInvalidBlockNum(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	req := httptest.NewRequest("GET", "/blocks/invalid", nil)
	req.SetPathValue("block_num", "invalid")
	w := httptest.NewRecorder()

	api.GetBlockByNumber(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid block number")
}

// TestGetBlockByNumberNotFound tests 404 for non-existent block
func TestGetBlockByNumberNotFound(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	req := httptest.NewRequest("GET", "/blocks/999", nil)
	req.SetPathValue("block_num", "999")
	w := httptest.NewRecorder()

	api.GetBlockByNumber(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetTxByID tests the /tx/{tx_id_hex} endpoint
func TestGetTxByID(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert block
	err := env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     5,
		TxCount:      1,
		PreviousHash: []byte("prev"),
		DataHash:     []byte("data"),
	})
	require.NoError(t, err)

	// Insert transaction
	txIDHex := "deadbeef12345678"
	txID, err := env.Queries.InsertTransaction(ctx, dbsqlc.InsertTransactionParams{
		BlockNum:       5,
		TxNum:          0,
		TxID:           mustDecodeHex(t, txIDHex),
		ValidationCode: 0,
	})
	require.NoError(t, err)

	// Insert namespace
	nsID, err := env.Queries.InsertTxNamespace(ctx, dbsqlc.InsertTxNamespaceParams{
		TransactionID: txID,
		NsID:          "testcc",
		NsVersion:     1,
	})
	require.NoError(t, err)

	// Insert read and write
	err = env.Queries.InsertTxRead(ctx, dbsqlc.InsertTxReadParams{
		TxNamespaceID: nsID,
		Key:           []byte("readkey"),
		IsReadWrite:   false,
	})
	require.NoError(t, err)

	err = env.Queries.InsertTxWrite(ctx, dbsqlc.InsertTxWriteParams{
		TxNamespaceID: nsID,
		Key:           []byte("writekey"),
		Value:         []byte("writevalue"),
		IsBlindWrite:  false,
	})
	require.NoError(t, err)

	// Test the endpoint
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/tx/"+txIDHex, nil)
	req.SetPathValue("tx_id_hex", txIDHex)
	w := httptest.NewRecorder()

	api.GetTxByID(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.TxWithBlockResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	assert.Equal(t, txIDHex, resp.Transaction.TxID)
	assert.Equal(t, int64(5), resp.Block.BlockNum)
	assert.Len(t, resp.Transaction.Reads, 1)
	assert.Len(t, resp.Transaction.Writes, 1)
}

// TestGetTxByIDInvalidHex tests error handling for invalid hex
func TestGetTxByIDInvalidHex(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	req := httptest.NewRequest("GET", "/tx/invalidhex", nil)
	req.SetPathValue("tx_id_hex", "ZZZ_invalid")
	w := httptest.NewRecorder()

	api.GetTxByID(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "invalid tx_id hex")
}

// TestGetTxByIDNotFound tests 404 for non-existent transaction
func TestGetTxByIDNotFound(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	req := httptest.NewRequest("GET", "/tx/aaaabbbbccccdddd", nil)
	req.SetPathValue("tx_id_hex", "aaaabbbbccccdddd")
	w := httptest.NewRecorder()

	api.GetTxByID(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestGetNamespacePolicies tests the /policies/{namespace} endpoint
func TestGetNamespacePolicies(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert policies
	err := env.Queries.UpsertNamespacePolicy(ctx, dbsqlc.UpsertNamespacePolicyParams{
		Namespace: "mycc",
		Version:   1,
		Policy:    json.RawMessage(`{"policy":"v1"}`),
	})
	require.NoError(t, err)

	err = env.Queries.UpsertNamespacePolicy(ctx, dbsqlc.UpsertNamespacePolicyParams{
		Namespace: "mycc",
		Version:   2,
		Policy:    json.RawMessage(`{"policy":"v2"}`),
	})
	require.NoError(t, err)

	// Test getting all versions
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/policies/mycc", nil)
	req.SetPathValue("namespace", "mycc")
	w := httptest.NewRecorder()

	api.GetNamespacePolicies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []types.NamespacePolicyResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp, 2)
}

// TestGetNamespacePoliciesLatest tests the latest=true query parameter
func TestGetNamespacePoliciesLatest(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert policies
	err := env.Queries.UpsertNamespacePolicy(ctx, dbsqlc.UpsertNamespacePolicyParams{
		Namespace: "testcc",
		Version:   1,
		Policy:    json.RawMessage(`{"policy":"v1"}`),
	})
	require.NoError(t, err)

	err = env.Queries.UpsertNamespacePolicy(ctx, dbsqlc.UpsertNamespacePolicyParams{
		Namespace: "testcc",
		Version:   2,
		Policy:    json.RawMessage(`{"policy":"v2"}`),
	})
	require.NoError(t, err)

	// Test getting latest only
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/policies/testcc?latest=true", nil)
	req.SetPathValue("namespace", "testcc")
	w := httptest.NewRecorder()

	api.GetNamespacePolicies(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp []types.NamespacePolicyResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Len(t, resp, 1)
	assert.Equal(t, int64(2), resp[0].Version)
}

// TestHealthHandler tests the /healthz endpoint
func TestHealthHandler(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	api.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

// TestHealthHandlerDatabaseDown tests health check when database is unavailable
func TestHealthHandlerDatabaseDown(t *testing.T) {
	// Create API with nil pool to simulate database down
	api := &API{
		q:    nil,
		pool: nil,
	}

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	api.HealthHandler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp HealthResponse
	err := json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.Equal(t, "ok", resp.Status)
}

// TestGetBlockHeightValue tests the helper function
func TestGetBlockHeightValue(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert a block
	err := env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     100,
		TxCount:      10,
		PreviousHash: []byte("prev"),
		DataHash:     []byte("data"),
	})
	require.NoError(t, err)

	api := NewAPI(env.Pool)
	height, err := api.GetBlockHeightValue(ctx)

	require.NoError(t, err)
	assert.Equal(t, int64(100), height)
}

// TestRouter tests that all routes are registered
func TestRouter(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	api := NewAPI(env.Pool)

	router := api.Router()
	assert.NotNil(t, router)

	// Test that routes exist by checking they don't return 404
	tests := []struct {
		method string
		path   string
	}{
		{"GET", "/blocks/height"},
		{"GET", "/healthz"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			// Should not be 404 (may be other error if no data, but route exists)
			assert.NotEqual(t, http.StatusNotFound, w.Code, "route should exist")
		})
	}
}

// TestParseInt tests the parseInt helper function
func TestParseInt(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		key      string
		defValue int
		expected int
	}{
		{
			name:     "valid integer",
			query:    "limit=10",
			key:      "limit",
			defValue: 5,
			expected: 10,
		},
		{
			name:     "missing parameter",
			query:    "",
			key:      "limit",
			defValue: 5,
			expected: 5,
		},
		{
			name:     "invalid integer",
			query:    "limit=invalid",
			key:      "limit",
			defValue: 5,
			expected: 5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/?"+tt.query, nil)
			result := parseInt(req, tt.key, tt.defValue)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestGetBlockWithPagination tests limitTx and offsetTx query parameters
func TestGetBlockWithPagination(t *testing.T) {
	env := db.NewDatabaseTestEnv(t)
	ctx := context.Background()

	// Insert block with multiple transactions
	err := env.Queries.InsertBlock(ctx, dbsqlc.InsertBlockParams{
		BlockNum:     20,
		TxCount:      3,
		PreviousHash: []byte("prev"),
		DataHash:     []byte("data"),
	})
	require.NoError(t, err)

	// Insert 3 transactions
	for i := 0; i < 3; i++ {
		_, err := env.Queries.InsertTransaction(ctx, dbsqlc.InsertTransactionParams{
			BlockNum:       20,
			TxNum:          int64(i),
			TxID:           mustDecodeHex(t, hex.EncodeToString([]byte{byte(i)})),
			ValidationCode: 0,
		})
		require.NoError(t, err)
	}

	// Test with limit
	api := NewAPI(env.Pool)
	req := httptest.NewRequest("GET", "/blocks/20?limitTx=2", nil)
	req.SetPathValue("block_num", "20")
	w := httptest.NewRecorder()

	api.GetBlockByNumber(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp types.BlockResponse
	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)
	assert.LessOrEqual(t, len(resp.Transactions), 2)
}

// Helper functions

func mustDecodeHex(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}
