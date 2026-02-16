/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/app"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/config"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Create server
	server, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	// Setup signal handling
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Run server
	if err := server.Run(rootCtx); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

