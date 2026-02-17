/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

import "encoding/json"

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
	TxNum          int64                 `json:"tx_num"`
	TxID           string                `json:"tx_id"`
	ValidationCode int64                 `json:"validation_code"`
	Reads          []ReadRecordResponse  `json:"reads"`
	Writes         []WriteRecordResponse `json:"writes"`
	Endorsements   []EndorsementResponse `json:"endorsements"`
}

type ReadRecordResponse struct {
	ID          int64  `json:"id"`
	NsID        string `json:"ns_id"`
	Key         string `json:"key"`
	Version     *int64 `json:"version,omitempty"`
	IsReadWrite bool   `json:"is_read_write"`
}

type WriteRecordResponse struct {
	ID           int64  `json:"id"`
	NsID         string `json:"ns_id"`
	Key          string `json:"key"`
	Value        string `json:"value"`
	IsBlindWrite bool   `json:"is_blind_write"`
	ReadVersion  *int64 `json:"read_version,omitempty"`
}

type EndorsementResponse struct {
	ID          int64           `json:"id"`
	NsID        string          `json:"ns_id"`
	Endorsement string          `json:"endorsement"`
	MspID       *string         `json:"msp_id,omitempty"`
	Identity    json.RawMessage `json:"identity,omitempty"`
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
