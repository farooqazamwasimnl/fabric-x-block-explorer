/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"fmt"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/logging"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/cenkalti/backoff/v4"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
)

var receiverLogger = logging.New("block-receiver")

// BlockReceiver starts a long-running loop that connects to the Sidecar stream,
// forwards received Fabric blocks to the 'out' channel and handles reconnection
// with backoff. Fatal errors and panics are reported on errCh.
func BlockReceiver(ctx context.Context, streamer *sidecarstream.Streamer, out chan<- *common.Block, errCh chan<- error, channelSize int) {
	defer func() {
		if r := recover(); r != nil {
			errCh <- fmt.Errorf("blockReceiver panic: %v", r)
		}
	}()

	receiverLogger.Info("blockReceiver started")
	backoffObj := NewBackoff()

	for {
		select {
		case <-ctx.Done():
			receiverLogger.Info("blockreceiver stopping")
			return
		default:
		}

		blockCh := make(chan *common.Block, channelSize)

		receiverLogger.Info("blockreceiver: starting Sidecar stream")
		streamer.StartDeliver(ctx, blockCh)
		backoffObj.Reset()

		if err := consumeBlocks(ctx, blockCh, out); err != nil {
			receiverLogger.Warnf("blockreceiver stream error: %v", err)
		}

		wait := backoffObj.NextBackOff()
		if wait == backoff.Stop {
			wait = 30 * time.Second
		}
		receiverLogger.Infof("blockreceiver: reconnecting after %v", wait)

		select {
		case <-ctx.Done():
			receiverLogger.Info("blockreceiver stopping before reconnect")
			return
		case <-time.After(wait):
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
				continue
			}

			select {
			case <-ctx.Done():
				return nil
			case out <- blk:
			}
		}
	}
}
