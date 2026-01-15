/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package types

type ProcessedBlock struct {
	Number    uint64
	Txns      int
	Data      interface{}
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
	Key      string
	IsDelete bool
	Value    []byte
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
	IsDelete       bool
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
			IsDelete:  w.IsDelete,
		}
	}
	return rec
}

type BlockInfo struct {
	Number       uint64
	PreviousHash []byte
	DataHash     []byte
}
