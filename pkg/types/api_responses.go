/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

// ----------------------------
// API Response Types
// ----------------------------

type BlockResponse struct {
	BlockNum     int64                      `json:"block_num"`
	TxCount      int32                      `json:"tx_count"`
	PreviousHash string                     `json:"previous_hash"`
	DataHash     string                     `json:"data_hash"`
	Transactions []TransactionWithWriteSets `json:"transactions"`
}

type TransactionWithWriteSets struct {
	ID             int64                 `json:"id"`
	BlockNum       int64                 `json:"block_num"`
	TxNum          int64                 `json:"tx_num"`
	TxID           string                `json:"tx_id"`
	ValidationCode int64                 `json:"validation_code"`
	Writes         []WriteRecordResponse `json:"writes"`
}

type WriteRecordResponse struct {
	ID          int64  `json:"id"`
	NamespaceID int64  `json:"namespace_id"`
	BlockNum    int64  `json:"block_num"`
	TxNum       int64  `json:"tx_num"`
	TxID        string `json:"tx_id"`
	Key         string `json:"key"`
	Value       string `json:"value"`
	IsDelete    bool   `json:"is_delete"`
}

type TxWithBlockResponse struct {
	Transaction TransactionWithWriteSets `json:"transaction"`
	Block       BlockHeaderOnly          `json:"block"`
}

type BlockHeaderOnly struct {
	BlockNum     int64  `json:"block_num"`
	TxCount      int32  `json:"tx_count"`
	PreviousHash string `json:"previous_hash"`
	DataHash     string `json:"data_hash"`
}
