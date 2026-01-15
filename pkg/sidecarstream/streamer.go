/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sidecarstream

import (
	"context"
	"log"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/config"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-committer/service/sidecar/sidecarclient"
	"github.com/hyperledger/fabric-x-committer/utils/connection"
)

// Streamer wraps a sidecar client and configuration for delivering blocks.
type Streamer struct {
	cfg    config.SidecarConfig
	client *sidecarclient.Client
}

// NewStreamer creates and returns a configured Streamer.
func NewStreamer(cfg config.SidecarConfig) (*Streamer, error) {
	cc := &connection.ClientConfig{
		Endpoint: &connection.Endpoint{
			Host: cfg.Host,
			Port: cfg.Port,
		},
	}

	params := &sidecarclient.Parameters{
		Client:    cc,
		ChannelID: cfg.ChannelID,
	}

	client, err := sidecarclient.New(params)
	if err != nil {
		return nil, err
	}

	s := &Streamer{
		cfg:    cfg,
		client: client,
	}

	log.Printf("sidecarstream: created streamer for %s:%d channel=%s", cfg.Host, cfg.Port, cfg.ChannelID)
	return s, nil
}

// StartDeliver starts a goroutine that calls the sidecar client's Deliver method.
// Blocks received from the sidecar are forwarded to the provided out channel.
// The goroutine logs when it exits and reports any Deliver error.
func (s *Streamer) StartDeliver(ctx context.Context, out chan<- *common.Block) {
	log.Printf("sidecarstream: StartDeliver channel=%s start=%d end=%d", s.cfg.ChannelID, s.cfg.StartBlk, s.cfg.EndBlk)

	go func() {
		defer log.Println("sidecarstream: StartDeliver goroutine exiting")

		deliverParams := &sidecarclient.DeliverParameters{
			StartBlkNum: int64(s.cfg.StartBlk),
			EndBlkNum:   s.cfg.EndBlk,
			OutputBlock: out,
		}

		if err := s.client.Deliver(ctx, deliverParams); err != nil {
			log.Printf("sidecarstream: Deliver returned error: %v", err)
		}
	}()
}

// CloseConnections closes any underlying connections held by the sidecar client.
func (s *Streamer) CloseConnections() {
	if s.client != nil {
		s.client.CloseConnections()
	}
}
