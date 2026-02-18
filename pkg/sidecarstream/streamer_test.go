/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sidecarstream

import (
	"context"
	"testing"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/config"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStreamer(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "localhost",
		Port:      7052,
		ChannelID: "testchannel",
		StartBlk:  0,
		EndBlk:    1000,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	assert.NotNil(t, streamer.client)
	assert.Equal(t, "localhost", streamer.cfg.Host)
	assert.Equal(t, 7052, streamer.cfg.Port)
	assert.Equal(t, "testchannel", streamer.cfg.ChannelID)
	assert.Equal(t, uint64(0), streamer.cfg.StartBlk)
	assert.Equal(t, uint64(1000), streamer.cfg.EndBlk)

	// Clean up
	streamer.CloseConnections()
}

func TestNewStreamerConfiguration(t *testing.T) {
	testCases := []struct {
		name   string
		config config.SidecarConfig
	}{
		{
			name: "default configuration",
			config: config.SidecarConfig{
				Host:      "localhost",
				Port:      7052,
				ChannelID: "mychannel",
				StartBlk:  0,
				EndBlk:    1000,
			},
		},
		{
			name: "specific block range",
			config: config.SidecarConfig{
				Host:      "peer.example.com",
				Port:      8052,
				ChannelID: "businesschannel",
				StartBlk:  100,
				EndBlk:    200,
			},
		},
		{
			name: "start from specific block",
			config: config.SidecarConfig{
				Host:      "192.168.1.100",
				Port:      9052,
				ChannelID: "ledger1",
				StartBlk:  500,
				EndBlk:    5000,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			streamer, err := NewStreamer(tc.config)
			require.NoError(t, err)
			require.NotNil(t, streamer)

			assert.Equal(t, tc.config.Host, streamer.cfg.Host)
			assert.Equal(t, tc.config.Port, streamer.cfg.Port)
			assert.Equal(t, tc.config.ChannelID, streamer.cfg.ChannelID)
			assert.Equal(t, tc.config.StartBlk, streamer.cfg.StartBlk)
			assert.Equal(t, tc.config.EndBlk, streamer.cfg.EndBlk)

			streamer.CloseConnections()
		})
	}
}

func TestStreamerCloseConnections(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "localhost",
		Port:      7052,
		ChannelID: "testchannel",
		StartBlk:  0,
		EndBlk:    1000,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)

	// Should not panic
	assert.NotPanics(t, func() {
		streamer.CloseConnections()
	})

	// Multiple closes should be safe
	assert.NotPanics(t, func() {
		streamer.CloseConnections()
	})
}

func TestStreamerCloseConnectionsNilClient(t *testing.T) {
	// Create a streamer with nil client (edge case)
	streamer := &Streamer{
		cfg: config.SidecarConfig{
			Host:      "localhost",
			Port:      7052,
			ChannelID: "test",
		},
		client: nil,
	}

	// Should not panic with nil client
	assert.NotPanics(t, func() {
		streamer.CloseConnections()
	})
}

func TestStartDeliver(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "localhost",
		Port:      7052,
		ChannelID: "testchannel",
		StartBlk:  0,
		EndBlk:    1000,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	defer streamer.CloseConnections()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	blockCh := make(chan *common.Block, 10)

	// Start deliver - will fail to connect but should not panic
	assert.NotPanics(t, func() {
		streamer.StartDeliver(ctx, blockCh)
	})

	// Give it a moment to start the goroutine
	time.Sleep(100 * time.Millisecond)

	// Cancel context to stop the deliver
	cancel()

	// Should exit gracefully
	time.Sleep(500 * time.Millisecond)
}

func TestStartDeliverContextCancellation(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "localhost",
		Port:      7052,
		ChannelID: "testchannel",
		StartBlk:  0,
		EndBlk:    100,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	defer streamer.CloseConnections()

	ctx, cancel := context.WithCancel(context.Background())
	blockCh := make(chan *common.Block, 10)

	// Start deliver
	streamer.StartDeliver(ctx, blockCh)

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Cancel immediately
	cancel()

	// Should exit without hanging
	time.Sleep(500 * time.Millisecond)
}

func TestStartDeliverMultipleCalls(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "localhost",
		Port:      7052,
		ChannelID: "testchannel",
		StartBlk:  0,
		EndBlk:    1000,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	defer streamer.CloseConnections()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	blockCh1 := make(chan *common.Block, 5)
	blockCh2 := make(chan *common.Block, 5)

	// Multiple StartDeliver calls should not panic
	assert.NotPanics(t, func() {
		streamer.StartDeliver(ctx, blockCh1)
		streamer.StartDeliver(ctx, blockCh2)
	})

	time.Sleep(200 * time.Millisecond)
}

func TestStreamerConfigPreservation(t *testing.T) {
	cfg := config.SidecarConfig{
		Host:      "peer.example.com",
		Port:      7052,
		ChannelID: "mychannel",
		StartBlk:  42,
		EndBlk:    100,
	}

	streamer, err := NewStreamer(cfg)
	require.NoError(t, err)
	require.NotNil(t, streamer)
	defer streamer.CloseConnections()

	// Verify configuration is preserved exactly
	assert.Equal(t, cfg.Host, streamer.cfg.Host)
	assert.Equal(t, cfg.Port, streamer.cfg.Port)
	assert.Equal(t, cfg.ChannelID, streamer.cfg.ChannelID)
	assert.Equal(t, cfg.StartBlk, streamer.cfg.StartBlk)
	assert.Equal(t, cfg.EndBlk, streamer.cfg.EndBlk)
}
