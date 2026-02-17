/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

type ProcessedBlock struct {
	Number    uint64
	Txns      int
	Data      any
	BlockInfo *BlockInfo
}

// ReadWriteSet is the outcome of an optimistically executed transaction.
type ReadWriteSet struct {
	Reads  []KVRead
	Writes []KVWrite
}

type KVRead struct {
	Key     string
	Version *Version
}

type KVWrite struct {
	Key   string
	Value []byte
}

type Version struct {
	BlockNum uint64
	TxNum    uint64
}

// WriteRecord represents a single write or delete in the world state.
type WriteRecord struct {
	Namespace      string
	Key            string
	BlockNum       uint64
	TxNum          uint64
	Value          []byte
	TxID           string
	ValidationCode int32
}

func Records(namespace string, blockNum, txNum uint64, txID string, rws ReadWriteSet) []WriteRecord {
	rec := make([]WriteRecord, len(rws.Writes))
	for i, w := range rws.Writes {
		rec[i] = WriteRecord{
			Namespace: namespace,
			BlockNum:  blockNum,
			TxNum:     txNum,
			TxID:      txID,
			Key:       w.Key,
			Value:     w.Value,
		}
	}
	return rec
}

type BlockInfo struct {
	Number       uint64
	PreviousHash []byte
	DataHash     []byte
}

// TxNamespaceRecord represents a namespace within a transaction.
type TxNamespaceRecord struct {
	BlockNum       uint64
	TxNum          uint64
	TxID           string
	NsID           string
	NsVersion      uint64
	ValidationCode int32
}

// ParsedBlockData contains both writes and namespace records.
type ParsedBlockData struct {
	Writes       []WriteRecord
	TxNamespaces []TxNamespaceRecord
}
