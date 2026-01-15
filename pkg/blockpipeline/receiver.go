/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
)

// BlockReceiver starts a long-running loop that connects to the Sidecar stream,
// forwards received Fabric blocks to the 'out' channel and handles reconnection
// with backoff. Fatal errors and panics are reported on errCh.
func BlockReceiver(ctx context.Context, streamer *sidecarstream.Streamer, out chan<- *common.Block, errCh chan<- error, channelSize int) {
	// Recover from panics and report them to errCh.
	defer func() {
		if r := recover(); r != nil {
			errCh <- fmt.Errorf("blockReceiver panic: %v", r)
		}
	}()

	log.Println("blockReceiver started")
	backoff := NewBackoff()

	for {
		// Stop immediately if context is cancelled.
		select {
		case <-ctx.Done():
			log.Println("blockreceiver stopping")
			close(out)
			return
		default:
		}

		// Per-connection channel for Sidecar deliver.
		blockCh := make(chan *common.Block, channelSize)

		log.Println("blockreceiver: starting Sidecar stream")
		streamer.StartDeliver(ctx, blockCh)
		backoff.Reset()

		// Consume blocks from blockCh and forward to out.
		if err := consumeBlocks(ctx, blockCh, out); err != nil {
			log.Printf("blockreceiver stream error: %v", err)
		}

		// Reconnect with backoff delay.
		wait := backoff.Next()
		log.Printf("blockreceiver: reconnecting after %v", wait)

		select {
		case <-ctx.Done():
			log.Println("blockreceiver stopping before reconnect")
			close(out)
			return
		case <-time.After(wait):
			// loop and try again
		}
	}
}

// consumeBlocks reads from the provided blockCh and forwards non-nil blocks to out.
// It returns an error when blockCh is closed unexpectedly. It respects ctx cancellation.
func consumeBlocks(ctx context.Context, blockCh <-chan *common.Block, out chan<- *common.Block) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case blk, ok := <-blockCh:
			if !ok {
				return fmt.Errorf("sidecar block channel closed")
			}
			if blk == nil {
				// skip nil blocks
				continue
			}

			// Respect context cancellation while attempting to forward.
			select {
			case <-ctx.Done():
				return nil
			case out <- blk:
				// forwarded successfully
			}
		}
	}
}
