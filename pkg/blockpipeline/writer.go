/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package blockpipeline

import (
	"context"
	"fmt"
	"log"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
)

// BlockWriter persists processed blocks to the database.
func BlockWriter(ctx context.Context, writer *db.BlockWriter, in <-chan *types.ProcessedBlock, errCh chan<- error) {
	defer func() {
		if r := recover(); r != nil {
			errCh <- fmt.Errorf("blockWriter panic: %v", r)
		}
	}()

	log.Println("blockWriter started")

	for {
		select {
		case <-ctx.Done():
			log.Println("blockWriter stopping")
			return

		case pb, ok := <-in:
			if !ok {
				errCh <- fmt.Errorf("processedBlocks channel closed")
				return
			}
			if pb == nil {
				continue
			}

			if err := writer.WriteProcessedBlock(ctx, pb); err != nil {
				errCh <- fmt.Errorf("db write error: %w", err)
				return
			}
		}
	}
}
