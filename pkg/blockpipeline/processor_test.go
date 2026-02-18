/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"testing"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockProcessor(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	in := make(chan *common.Block, 10)
	out := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockProcessor(ctx, in, out, errCh)

	// Send a valid block
	block := &common.Block{
		Header: &common.BlockHeader{
			Number: 1,
		},
		Data: &common.BlockData{
			Data: [][]byte{},
		},
		Metadata: &common.BlockMetadata{
			Metadata: [][]byte{
				{}, // SIGNATURES
				{}, // LAST_CONFIG
				{}, // TRANSACTIONS_FILTER
			},
		},
	}

	in <- block

	// Should receive processed block
	select {
	case pb := <-out:
		assert.Equal(t, uint64(1), pb.Number)
		assert.Equal(t, 0, pb.Txns)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for processed block")
	}

	cancel()
}

func TestBlockProcessorNilBlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in := make(chan *common.Block, 10)
	out := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockProcessor(ctx, in, out, errCh)

	// Send nil block (should be skipped)
	in <- nil

	// Send valid block
	block := &common.Block{
		Header: &common.BlockHeader{
			Number: 2,
		},
		Data: &common.BlockData{
			Data: [][]byte{},
		},
		Metadata: &common.BlockMetadata{
			Metadata: [][]byte{
				{}, // SIGNATURES
				{}, // LAST_CONFIG
				{}, // TRANSACTIONS_FILTER
			},
		},
	}
	in <- block

	// Should only receive the valid block
	select {
	case pb := <-out:
		assert.Equal(t, uint64(2), pb.Number)
	case err := <-errCh:
		t.Fatalf("unexpected error: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for processed block")
	}

	cancel()
}

func TestBlockProcessorContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	in := make(chan *common.Block, 10)
	out := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockProcessor(ctx, in, out, errCh)

	// Cancel immediately
	cancel()

	// Should not receive any errors (clean shutdown)
	select {
	case err := <-errCh:
		t.Fatalf("unexpected error on clean shutdown: %v", err)
	case <-time.After(500 * time.Millisecond):
		// Expected - no error
	}
}

func TestBlockProcessorChannelClosed(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in := make(chan *common.Block, 10)
	out := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockProcessor(ctx, in, out, errCh)

	// Close input channel
	close(in)

	// Should receive error about closed channel
	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "channel closed")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

func TestBlockProcessorInvalidBlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	in := make(chan *common.Block, 10)
	out := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockProcessor(ctx, in, out, errCh)

	// Send block with nil header (should cause parsing error)
	block := &common.Block{
		Header: nil,
		Data: &common.BlockData{
			Data: [][]byte{},
		},
	}

	in <- block

	// Should receive error
	select {
	case err := <-errCh:
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "block processing error")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

func TestProcessBlock(t *testing.T) {
	block := &common.Block{
		Header: &common.BlockHeader{
			Number: 5,
		},
		Data: &common.BlockData{
			Data: [][]byte{
				[]byte("tx1"),
				[]byte("tx2"),
			},
		},
		Metadata: &common.BlockMetadata{
			Metadata: [][]byte{
				{}, // SIGNATURES
				{}, // LAST_CONFIG
				{}, // TRANSACTIONS_FILTER
			},
		},
	}

	processed, err := processBlock(block)
	require.NoError(t, err)

	assert.Equal(t, uint64(5), processed.Number)
	assert.Equal(t, 2, processed.Txns)
	assert.NotNil(t, processed.Data)
}

func TestProcessBlockNilHeader(t *testing.T) {
	block := &common.Block{
		Header: nil,
		Data: &common.BlockData{
			Data: [][]byte{},
		},
	}

	_, err := processBlock(block)
	assert.Error(t, err)
}
