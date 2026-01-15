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

// BlockWriter consumes ProcessedBlock values from 'in' and persists them using
// the provided BlockWriter. Fatal errors and panics are reported on errCh.
func BlockWriter(ctx context.Context, writer *db.BlockWriter, in <-chan *types.ProcessedBlock, errCh chan<- error) {
	// Recover from panics and report them to errCh.
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
				// Input channel closed unexpectedly — report and exit.
				errCh <- fmt.Errorf("processedBlocks channel closed")
				return
			}
			if pb == nil {
				// Skip nil processed blocks.
				continue
			}

			// Persist the processed block. On error, report and exit.
			if err := writer.WriteProcessedBlock(ctx, pb); err != nil {
				errCh <- fmt.Errorf("db write error: %w", err)
				return
			}
		}
	}
}
