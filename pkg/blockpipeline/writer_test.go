/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	dbtest "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlockWriter(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Send a processed block
	pb := &types.ProcessedBlock{
		Number: 1,
		Txns:   0,
		Data:   &types.ParsedBlockData{},
		BlockInfo: &types.BlockInfo{
			Number:       1,
			PreviousHash: []byte("prevhash"),
			DataHash:     []byte("hash1"),
		},
	}

	in <- pb

	// Wait a bit for write to complete
	time.Sleep(500 * time.Millisecond)

	// Verify block was written
	env.AssertBlockExists(t, 1)

	cancel()
}

func TestBlockWriterMultipleBlocks(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Send multiple blocks
	for i := uint64(1); i <= 3; i++ {
		pb := &types.ProcessedBlock{
			Number: i,
			Txns:   0,
			Data:   &types.ParsedBlockData{},
			BlockInfo: &types.BlockInfo{
				Number:       i,
				PreviousHash: []byte("prevhash"),
				DataHash:     []byte(fmt.Sprintf("hash%d", i)),
			},
		}
		in <- pb
	}

	// Wait for writes to complete
	time.Sleep(1 * time.Second)

	// Verify all blocks were written
	count := env.GetBlockCount(t)
	assert.Equal(t, int64(3), count)

	cancel()
}

func TestBlockWriterNilBlock(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Send nil block (should be skipped)
	in <- nil

	// Send valid block
	pb := &types.ProcessedBlock{
		Number: 1,
		Txns:   0,
		Data:   &types.ParsedBlockData{},
		BlockInfo: &types.BlockInfo{
			Number:       1,
			PreviousHash: []byte("prevhash"),
			DataHash:     []byte("hash1"),
		},
	}
	in <- pb

	// Wait for write
	time.Sleep(500 * time.Millisecond)

	// Should only have 1 block
	count := env.GetBlockCount(t)
	assert.Equal(t, int64(1), count)

	cancel()
}

func TestBlockWriterContextCancellation(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithCancel(context.Background())

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Cancel immediately
	cancel()

	// Should not receive any errors (clean shutdown)
	select {
	case err := <-errCh:
		// Panic recovery might send error, but not a write error
		if err != nil {
			assert.NotContains(t, err.Error(), "db write error")
		}
	case <-time.After(500 * time.Millisecond):
		// Expected - no error
	}
}

func TestBlockWriterChannelClosed(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Close input channel
	close(in)

	// Should receive error about closed channel
	select {
	case err := <-errCh:
		require.Error(t, err)
		assert.Contains(t, err.Error(), "channel closed")
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}

func TestBlockWriterDatabaseError(t *testing.T) {
	env := dbtest.NewDatabaseTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	writer := db.NewBlockWriter(env.Pool)
	in := make(chan *types.ProcessedBlock, 10)
	errCh := make(chan error, 1)

	go BlockWriter(ctx, writer, in, errCh)

	// Send block with nil BlockInfo (should cause write error or panic)
	pb := &types.ProcessedBlock{
		Number:    1,
		Txns:      0,
		Data:      &types.ParsedBlockData{},
		BlockInfo: nil, // This will cause an error
	}

	in <- pb

	// Should receive database write error or panic error
	select {
	case err := <-errCh:
		require.Error(t, err)
		// Accept either db write error or panic
		assert.True(t, 
			strings.Contains(err.Error(), "db write error") || 
			strings.Contains(err.Error(), "panic"),
			"expected db write error or panic, got: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for error")
	}
}
