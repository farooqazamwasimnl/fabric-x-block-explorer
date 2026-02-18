/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"testing"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/stretchr/testify/assert"
)

func TestConsumeBlocks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockCh := make(chan *common.Block, 10)
	out := make(chan *common.Block, 10)

	// Send some blocks
	go func() {
		for i := 1; i <= 3; i++ {
			blockCh <- &common.Block{
				Header: &common.BlockHeader{
					Number: uint64(i),
				},
			}
		}
	}()

	// Start consuming in background
	errCh := make(chan error, 1)
	go func() {
		errCh <- consumeBlocks(ctx, blockCh, out)
	}()

	// Should receive the blocks
	for i := 1; i <= 3; i++ {
		select {
		case blk := <-out:
			assert.Equal(t, uint64(i), blk.Header.Number)
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout waiting for block %d", i)
		}
	}

	cancel()

	// Should return nil on context cancellation
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for consumeBlocks to return")
	}
}

func TestConsumeBlocksNilBlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockCh := make(chan *common.Block, 10)
	out := make(chan *common.Block, 10)

	// Send nil and valid blocks
	go func() {
		blockCh <- nil // Should be skipped
		blockCh <- &common.Block{
			Header: &common.BlockHeader{
				Number: 1,
			},
		}
	}()

	// Start consuming
	go consumeBlocks(ctx, blockCh, out)

	// Should only receive the valid block
	select {
	case blk := <-out:
		assert.Equal(t, uint64(1), blk.Header.Number)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for block")
	}

	cancel()
}

func TestConsumeBlocksChannelClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockCh := make(chan *common.Block, 10)
	out := make(chan *common.Block, 10)

	// Close the channel immediately
	close(blockCh)

	err := consumeBlocks(ctx, blockCh, out)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "channel closed")
}

func TestConsumeBlocksContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	blockCh := make(chan *common.Block, 10)
	out := make(chan *common.Block, 10)

	errCh := make(chan error, 1)
	go func() {
		errCh <- consumeBlocks(ctx, blockCh, out)
	}()

	// Cancel context
	cancel()

	// Should return nil (clean shutdown)
	select {
	case err := <-errCh:
		assert.NoError(t, err)
	case <-time.After(1 * time.Second):
		t.Fatal("timeout waiting for return")
	}
}

func TestBlockReceiverReconnect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	out := make(chan *common.Block, 10)
	errCh := make(chan error, 1)

	go BlockReceiver(ctx, &sidecarstream.Streamer{}, out, errCh, 10)

	// Note: This test verifies the reconnection logic compiles and runs,
	// but full integration testing requires actual sidecar connection
	// which is not available in unit tests

	cancel()
}

func TestBlockReceiverContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	out := make(chan *common.Block, 10)
	errCh := make(chan error, 1)

	go BlockReceiver(ctx, &sidecarstream.Streamer{}, out, errCh, 10)

	// Cancel immediately
	cancel()

	// Should not receive panic error
	select {
	case err := <-errCh:
		if err != nil {
			assert.NotContains(t, err.Error(), "panic")
		}
	case <-time.After(500 * time.Millisecond):
		// Expected - clean shutdown
	}
}
