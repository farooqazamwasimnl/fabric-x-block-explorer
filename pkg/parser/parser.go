/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package parser

import (
	"fmt"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"google.golang.org/protobuf/proto"
)

// Parse converts a Fabric block into a slice of WriteRecord and BlockInfo.
func Parse(b *common.Block) ([]types.WriteRecord, *types.BlockInfo, error) {
	writes := []types.WriteRecord{}

	// -------------------------------
	// Block Header
	// -------------------------------
	header := b.GetHeader()
	if header == nil {
		return writes, nil, fmt.Errorf("block header missing")
	}

	blockInfo := &types.BlockInfo{
		Number:       header.Number,
		PreviousHash: header.PreviousHash,
		DataHash:     header.DataHash,
	}

	// -------------------------------
	// Transaction Filter
	// -------------------------------
	if b.Metadata == nil || len(b.Metadata.Metadata) <= int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return writes, blockInfo, fmt.Errorf("block metadata missing TRANSACTIONS_FILTER")
	}
	txFilter := b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]

	// -------------------------------
	// Parse Each Transaction
	// -------------------------------
	for txNum, envBytes := range b.Data.Data {
		// Skip if txNum is out of range for the filter
		if txNum >= len(txFilter) {
			continue
		}

		validationCode := protoblocktx.Status(txFilter[txNum])
		// Only process committed transactions
		if validationCode != protoblocktx.Status_COMMITTED {
			continue
		}

		// Unmarshal envelope
		env := &common.Envelope{}
		if err := proto.Unmarshal(envBytes, env); err != nil {
			fmt.Printf("block %d tx %d invalid envelope: %s\n", header.Number, txNum, err)
			continue
		}

		// Extract RW sets
		rwsets, err := rwSets(env)
		if err != nil {
			fmt.Printf("block %d tx %d invalid rwset: %s\n", header.Number, txNum, err)
			continue
		}

		// Convert RW sets to WriteRecord and attach validation code
		for _, rw := range rwsets {
			records := types.Records(
				rw.Namespace,
				header.Number,
				uint64(txNum),
				rw.TxID,
				rw.Rwset,
			)

			for i := range records {
				records[i].ValidationCode = int32(validationCode)
			}

			writes = append(writes, records...)
		}
	}

	return writes, blockInfo, nil
}

// rwSets extracts namespace read-write sets and txID from an envelope.
// Returns a slice of nsRwset preserving the original structure.
func rwSets(env *common.Envelope) ([]nsRwset, error) {
	out := []nsRwset{}

	// Payload
	pl := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, pl); err != nil {
		return out, fmt.Errorf("payload: %w", err)
	}

	// Channel header -> TxID
	chdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(pl.Header.ChannelHeader, chdr); err != nil {
		return out, fmt.Errorf("channel header: %w", err)
	}
	txID := chdr.TxId

	// Transaction (custom protoblocktx)
	tx := &protoblocktx.Tx{}
	if err := proto.Unmarshal(pl.Data, tx); err != nil {
		return out, fmt.Errorf("transaction: %w", err)
	}

	// Namespaces -> build ReadWriteSet per namespace
	for _, ns := range tx.Namespaces {
		rws := types.ReadWriteSet{
			Reads:  []types.KVRead{},
			Writes: []types.KVWrite{},
		}

		// Blind writes
		for _, bw := range ns.BlindWrites {
			rws.Writes = append(rws.Writes, types.KVWrite{
				Key:   string(bw.Key),
				Value: bw.Value,
			})
		}

		// Normal reads + writes
		for _, rw := range ns.ReadWrites {
			read := types.KVRead{Key: string(rw.Key)}
			if rw.Version != nil && *rw.Version > 0 {
				read.Version = &types.Version{
					BlockNum: *rw.Version,
				}
			}
			rws.Reads = append(rws.Reads, read)

			rws.Writes = append(rws.Writes, types.KVWrite{
				Key:   string(rw.Key),
				Value: rw.Value,
			})
		}

		out = append(out, nsRwset{
			Namespace: ns.NsId,
			Rwset:     rws,
			TxID:      txID,
		})
	}

	return out, nil
}

type nsRwset struct {
	Namespace string             `json:"namespace"`
	Rwset     types.ReadWriteSet `json:"rwset"`
	TxID      string             `json:"-"`
}
