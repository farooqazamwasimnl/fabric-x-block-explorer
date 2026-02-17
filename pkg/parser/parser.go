/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package parser

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-protos-go-apiv2/msp"
	"github.com/hyperledger/fabric-protos-go-apiv2/peer"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"google.golang.org/protobuf/proto"
)

// Parse converts a Fabric block into ParsedBlockData and BlockInfo.
func Parse(b *common.Block) (*types.ParsedBlockData, *types.BlockInfo, error) {
	writes := []types.WriteRecord{}
	reads := []types.ReadRecord{}
	txNamespaces := []types.TxNamespaceRecord{}
	endorsements := []types.EndorsementRecord{}
	policies := []types.NamespacePolicyRecord{}

	header := b.GetHeader()
	if header == nil {
		return &types.ParsedBlockData{Writes: writes, Reads: reads, TxNamespaces: txNamespaces}, nil, fmt.Errorf("block header missing")
	}

	blockInfo := &types.BlockInfo{
		Number:       header.Number,
		PreviousHash: header.PreviousHash,
		DataHash:     header.DataHash,
	}

	if b.Metadata == nil || len(b.Metadata.Metadata) <= int(common.BlockMetadataIndex_TRANSACTIONS_FILTER) {
		return &types.ParsedBlockData{Writes: writes, Reads: reads, TxNamespaces: txNamespaces, Endorsements: endorsements, Policies: policies}, blockInfo, fmt.Errorf("block metadata missing TRANSACTIONS_FILTER")
	}
	txFilter := b.Metadata.Metadata[common.BlockMetadataIndex_TRANSACTIONS_FILTER]

	for txNum, envBytes := range b.Data.Data {
		if txNum >= len(txFilter) {
			continue
		}

		validationCode := protoblocktx.Status(txFilter[txNum])
		if validationCode != protoblocktx.Status_COMMITTED {
			continue
		}

		// Unmarshal envelope
		env := &common.Envelope{}
		if err := proto.Unmarshal(envBytes, env); err != nil {
			log.Printf("block %d tx %d invalid envelope: %v", header.Number, txNum, err)
			continue
		}

		// Check for namespace policy updates first
		if policyItems, ok := extractPolicies(env); ok {
			policies = append(policies, policyItems...)
			continue
		}

		// Extract RW sets (normal transaction)
		rwsets, err := rwSets(env)
		if err != nil {
			log.Printf("block %d tx %d invalid rwset: %v", header.Number, txNum, err)
			continue
		}

		// Convert RW sets to WriteRecord and attach validation code
		for _, rw := range rwsets {
			txNsRecord := types.TxNamespaceRecord{
				BlockNum:       header.Number,
				TxNum:          uint64(txNum),
				TxID:           rw.TxID,
				NsID:           rw.Namespace,
				NsVersion:      rw.NsVersion,
				ValidationCode: int32(validationCode),
			}
			txNamespaces = append(txNamespaces, txNsRecord)

			if len(rw.Endorsement) > 0 {
				// Try to extract identity from endorsement; fallback to signature-only
				mspID, identityJSON, err := endorsementToIdentityJSON(rw.Endorsement)
				if err != nil {
					endorsements = append(endorsements, types.EndorsementRecord{
						BlockNum:    header.Number,
						TxNum:       uint64(txNum),
						NsID:        rw.Namespace,
						Endorsement: rw.Endorsement,
					})
				} else {
					endorsements = append(endorsements, types.EndorsementRecord{
						BlockNum:    header.Number,
						TxNum:       uint64(txNum),
						NsID:        rw.Namespace,
						Endorsement: rw.Endorsement,
						MspID:       mspID,
						Identity:    identityJSON,
					})
				}
			}

			for _, read := range rw.Rwset.Reads {
				readRecord := types.ReadRecord{
					BlockNum:    header.Number,
					TxNum:       uint64(txNum),
					NsID:        rw.Namespace,
					Key:         read.Key,
					IsReadWrite: read.IsReadWrite,
				}
				if read.Version != nil {
					readRecord.Version = &read.Version.BlockNum
				}
				reads = append(reads, readRecord)
			}

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

	return &types.ParsedBlockData{
		Writes:       writes,
		Reads:        reads,
		TxNamespaces: txNamespaces,
		Endorsements: endorsements,
		Policies:     policies,
	}, blockInfo, nil
}

const metaNamespaceID = "_meta"

// policyToJSON converts protobuf policy bytes to a JSON object with base64-encoded policy
func policyToJSON(policyBytes []byte) (json.RawMessage, error) {
	// Store as base64-encoded bytes in a simple JSON structure
	// This allows storing in JSONB while preserving exact binary data
	return json.Marshal(map[string]string{
		"policy_bytes": base64.StdEncoding.EncodeToString(policyBytes),
	})
}

// endorsementToIdentityJSON extracts identity information from endorsement protobuf
func endorsementToIdentityJSON(endorsementBytes []byte) (*string, []byte, error) {
	// Parse the Endorsement protobuf
	endorsement := &peer.Endorsement{}
	if err := proto.Unmarshal(endorsementBytes, endorsement); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal endorsement: %w", err)
	}

	// Parse the SerializedIdentity from endorser field
	serializedID := &msp.SerializedIdentity{}
	if err := proto.Unmarshal(endorsement.Endorser, serializedID); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal endorser: %w", err)
	}

	// Extract mspid
	mspID := serializedID.Mspid

	// Create identity JSON structure
	identityData := map[string]interface{}{
		"mspid":    serializedID.Mspid,
		"id_bytes": base64.StdEncoding.EncodeToString(serializedID.IdBytes),
	}

	identityJSON, err := json.Marshal(identityData)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal identity: %w", err)
	}

	return &mspID, identityJSON, nil
}

// extractPolicies attempts to parse namespace policy updates from an envelope payload.
// Returns ok=true if the payload is a policy update.
func extractPolicies(env *common.Envelope) ([]types.NamespacePolicyRecord, bool) {
	pl := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, pl); err != nil {
		return nil, false
	}

	chdr := &common.ChannelHeader{}
	if pl.Header == nil || pl.Header.ChannelHeader == nil {
		return nil, false
	}
	if err := proto.Unmarshal(pl.Header.ChannelHeader, chdr); err != nil {
		return nil, false
	}
	if chdr.Type != int32(common.HeaderType_CONFIG) && chdr.Type != int32(common.HeaderType_CONFIG_UPDATE) {
		return nil, false
	}

	policies := &protoblocktx.NamespacePolicies{}
	if err := proto.Unmarshal(pl.Data, policies); err == nil && len(policies.Policies) > 0 {
		items := make([]types.NamespacePolicyRecord, 0, len(policies.Policies))
		for _, pd := range policies.Policies {
			if len(pd.Policy) == 0 {
				continue
			}
			ns := pd.Namespace
			if ns == "" {
				ns = metaNamespaceID
			}
			policyJSON, err := policyToJSON(pd.Policy)
			if err != nil {
				log.Printf("failed to convert policy to JSON for namespace %s: %v", ns, err)
				continue
			}
			items = append(items, types.NamespacePolicyRecord{
				Namespace:  ns,
				Version:    pd.Version,
				PolicyJSON: policyJSON,
			})
		}
		if len(items) > 0 {
			return items, true
		}
	}

	configTx := &protoblocktx.ConfigTransaction{}
	if err := proto.Unmarshal(pl.Data, configTx); err == nil && len(configTx.Envelope) > 0 {
		policyJSON, err := policyToJSON(configTx.Envelope)
		if err != nil {
			log.Printf("failed to convert config envelope to JSON: %v", err)
			return nil, false
		}
		return []types.NamespacePolicyRecord{
			{
				Namespace:  metaNamespaceID,
				Version:    configTx.Version,
				PolicyJSON: policyJSON,
			},
		}, true
	}

	return nil, false
}

// rwSets extracts namespace read-write sets and txID from an envelope.
// Returns a slice of nsRwset preserving the original structure.
func rwSets(env *common.Envelope) ([]nsRwset, error) {
	out := []nsRwset{}

	pl := &common.Payload{}
	if err := proto.Unmarshal(env.Payload, pl); err != nil {
		return out, fmt.Errorf("payload: %w", err)
	}

	chdr := &common.ChannelHeader{}
	if err := proto.Unmarshal(pl.Header.ChannelHeader, chdr); err != nil {
		return out, fmt.Errorf("channel header: %w", err)
	}
	txID := chdr.TxId

	tx := &protoblocktx.Tx{}
	if err := proto.Unmarshal(pl.Data, tx); err != nil {
		return out, fmt.Errorf("transaction: %w", err)
	}

	if len(tx.Signatures) > 0 && len(tx.Signatures) != len(tx.Namespaces) {
		log.Printf("tx %s signature count %d does not match namespaces %d", txID, len(tx.Signatures), len(tx.Namespaces))
	}

	for i, ns := range tx.Namespaces {
		rws := types.ReadWriteSet{
			Reads:  []types.KVRead{},
			Writes: []types.KVWrite{},
		}

		for _, ro := range ns.ReadsOnly {
			read := types.KVRead{Key: string(ro.Key), IsReadWrite: false}
			if ro.Version != nil && *ro.Version > 0 {
				read.Version = &types.Version{
					BlockNum: *ro.Version,
				}
			}
			rws.Reads = append(rws.Reads, read)
		}

		for _, bw := range ns.BlindWrites {
			rws.Writes = append(rws.Writes, types.KVWrite{
				Key:          string(bw.Key),
				Value:        bw.Value,
				IsBlindWrite: true,
				ReadVersion:  nil,
			})
		}

		for _, rw := range ns.ReadWrites {
			read := types.KVRead{Key: string(rw.Key), IsReadWrite: true}
			var readVersion *uint64
			if rw.Version != nil && *rw.Version > 0 {
				read.Version = &types.Version{
					BlockNum: *rw.Version,
				}
				readVersion = rw.Version
			}
			rws.Reads = append(rws.Reads, read)

			rws.Writes = append(rws.Writes, types.KVWrite{
				Key:          string(rw.Key),
				Value:        rw.Value,
				IsBlindWrite: false,
				ReadVersion:  readVersion,
			})
		}

		var endorsement []byte
		if i < len(tx.Signatures) {
			endorsement = tx.Signatures[i]
		}

		out = append(out, nsRwset{
			Namespace:   ns.NsId,
			Rwset:       rws,
			TxID:        txID,
			NsVersion:   ns.NsVersion,
			Endorsement: endorsement,
		})
	}

	return out, nil
}

type nsRwset struct {
	Namespace   string             `json:"namespace"`
	Rwset       types.ReadWriteSet `json:"rwset"`
	TxID        string             `json:"-"`
	NsVersion   uint64             `json:"-"`
	Endorsement []byte             `json:"-"`
}
