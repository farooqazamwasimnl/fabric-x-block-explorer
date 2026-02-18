/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"fmt"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/logging"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/parser"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
)

var logger = logging.New("blockpipeline")

// BlockProcessor parses raw blocks and sends them to the output channel.
func BlockProcessor(ctx context.Context, in <-chan *common.Block, out chan<- *types.ProcessedBlock, errCh chan<- error) {
	logger.Info("blockProcessor started")

	for {
		select {
		case <-ctx.Done():
			logger.Info("blockProcessor stopping")
			return

		case blk, ok := <-in:
			if !ok {
				errCh <- fmt.Errorf("receivedBlocks channel closed")
				return
			}
			if blk == nil {
				continue
			}

			processed, err := processBlock(blk)
			if err != nil {
				errCh <- fmt.Errorf("block processing error: %w", err)
				return
			}

			select {
			case <-ctx.Done():
				logger.Info("blockProcessor stopping before send")
				return
			case out <- processed:
			}
		}
	}
}

// processBlock parses a raw block into structured data.
func processBlock(blk *common.Block) (*types.ProcessedBlock, error) {
	number := blk.GetHeader().GetNumber()
	txCount := len(blk.GetData().GetData())

	parsedData, blockInfo, err := parser.Parse(blk)
	if err != nil {
		return nil, err
	}

	return &types.ProcessedBlock{
		Number:    number,
		Txns:      txCount,
		Data:      parsedData,
		BlockInfo: blockInfo,
	}, nil
}
