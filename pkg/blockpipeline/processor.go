/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"fmt"
	"log"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/parser"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
)

// BlockProcessor reads raw blocks from 'in', processes them and sends
// processed blocks to 'out'. Any fatal error is reported on errCh.
func BlockProcessor(ctx context.Context, in <-chan *common.Block, out chan<- *types.ProcessedBlock, errCh chan<- error) {
	log.Println("blockProcessor started")

	for {
		select {
		case <-ctx.Done():
			log.Println("blockProcessor stopping")
			return

		case blk, ok := <-in:
			if !ok {
				// Input channel closed unexpectedly.
				errCh <- fmt.Errorf("receivedBlocks channel closed")
				return
			}
			if blk == nil {
				// Skip nil blocks.
				continue
			}

			processed, err := processBlock(blk)
			if err != nil {
				errCh <- fmt.Errorf("block processing error: %w", err)
				return
			}

			// Respect context cancellation while attempting to send.
			select {
			case <-ctx.Done():
				log.Println("blockProcessor stopping before send")
				return
			case out <- processed:
			}
		}
	}
}

// processBlock converts a raw Fabric block into a ProcessedBlock using the parser package.
func processBlock(blk *common.Block) (*types.ProcessedBlock, error) {
	number := blk.GetHeader().GetNumber()
	txCount := len(blk.GetData().GetData())

	writes, blockInfo, err := parser.Parse(blk)
	if err != nil {
		return nil, err
	}

	return &types.ProcessedBlock{
		Number:    number,
		Txns:      txCount,
		Data:      writes,
		BlockInfo: blockInfo,
	}, nil
}
