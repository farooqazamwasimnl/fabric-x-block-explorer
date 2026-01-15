/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/api"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/config"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/workerpool"
)

func main() {
	// Root context cancelled on SIGINT / SIGTERM
	rootCtx, stopSignals := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stopSignals()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// Initialize DB (shared DB handle)
	sqlDB, err := db.NewPostgres(db.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DBName:   cfg.DB.DBName,
		SSLMode:  cfg.DB.SSLMode,
	})
	if err != nil {
		log.Fatalf("failed to init postgres: %v", err)
	}
	defer func() {
		_ = sqlDB.Close()
	}()

	// API server (used both for HTTP and for programmatic calls)
	apiServer := api.NewAPI(sqlDB)
	srv := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: apiServer.Router(),
	}

	// Query current block height and adjust sidecar start block if needed
	currentBlockHeight, err := apiServer.GetBlockHeightValue(rootCtx)
	if err != nil {
		log.Fatalf("failed to get block height: %v", err)
	}
	if currentBlockHeight > 0 {
		cfg.Sidecar.StartBlk = uint64(currentBlockHeight) + 1
	}

	// Create sidecar streamer (concrete type)
	streamer, err := sidecarstream.NewStreamer(cfg.Sidecar)
	if err != nil {
		log.Fatalf("failed to create streamer: %v", err)
	}

	// Ensure sensible defaults for workerpool config
	wpCfg := workerpool.Config{
		ProcessorCount: cfg.Workers.ProcessorCount,
		WriterCount:    cfg.Workers.WriterCount,
		RawBuf:         cfg.Buffer.RawChannelSize,
		ProcBuf:        cfg.Buffer.ProcessChannelSize,
	}
	// Pass concrete streamer pointer to workerpool
	wp := workerpool.New(wpCfg, sqlDB, streamer)

	// Central error channel (buffered to avoid blocking)
	errCh := make(chan error, 1)

	// Start HTTP server
	startHTTPServer(srv, errCh)

	// Start workerpool
	ctx, cancel := context.WithCancel(rootCtx)
	defer cancel()

	// Start workerpool and get an errgroup to wait on
	g := wp.Start(ctx, errCh)

	// Supervisor: wait for signal or first fatal error
	select {
	case <-rootCtx.Done():
		log.Println("shutdown requested by signal")
		cancel()
	case <-ctx.Done():
		log.Println("shutdown requested by context cancellation")
	case err := <-errCh:
		log.Printf("fatal error reported: %v", err)
		cancel()
	}

	// Begin shutdown sequence
	shutdownTimeout := time.Duration(cfg.Server.ShutdownTimeoutSec) * time.Second
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	} else {
		log.Println("http server shutdown complete")
	}

	// Wait for workerpool to finish (bounded by context cancellation)
	if err := g.Wait(); err != nil {
		log.Printf("workerpool exited with error: %v", err)
	} else {
		log.Println("workerpool exited cleanly")
	}

	log.Println("exiting")
}

// startHTTPServer runs the HTTP server in a goroutine and reports errors to errCh.
func startHTTPServer(srv *http.Server, errCh chan<- error) {
	go func() {
		log.Printf("REST API running on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			// non-blocking send
			select {
			case errCh <- err:
			default:
			}
		}
	}()
}
