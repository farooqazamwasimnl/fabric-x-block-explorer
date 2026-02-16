/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package app

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/api"
	pb "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/api/proto"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/config"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/workerpool"
	"google.golang.org/grpc"
)

// Server manages the block explorer application components.
type Server struct {
	config     *config.Config
	pool       *pgxpool.Pool
	apiServer  *api.API
	httpServer *http.Server
	grpcServer *grpc.Server
	streamer   *sidecarstream.Streamer
	workerPool *workerpool.Pool
}

// New creates a new Server instance.
func New(cfg *config.Config) (*Server, error) {
	// Initialize database
	pool, err := db.NewPostgres(db.Config{
		Host:     cfg.DB.Host,
		Port:     cfg.DB.Port,
		User:     cfg.DB.User,
		Password: cfg.DB.Password,
		DBName:   cfg.DB.DBName,
		SSLMode:  cfg.DB.SSLMode,
	})
	if err != nil {
		return nil, err
	}

	// Create API server
	apiServer := api.NewAPI(pool)

	// Create HTTP server
	httpServer := &http.Server{
		Addr:    cfg.Server.HTTPAddr,
		Handler: apiServer.Router(),
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()
	grpcHandler := api.NewGRPCServer(apiServer)
	pb.RegisterBlockExplorerServer(grpcServer, grpcHandler)

	// Query current block height and adjust sidecar start block if needed
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	currentBlockHeight, err := apiServer.GetBlockHeightValue(ctx)
	if err != nil {
		log.Printf("warning: could not get block height: %v", err)
	} else if currentBlockHeight > 0 {
		cfg.Sidecar.StartBlk = uint64(currentBlockHeight) + 1
	}

	// Create sidecar streamer
	streamer, err := sidecarstream.NewStreamer(cfg.Sidecar)
	if err != nil {
		pool.Close()
		return nil, err
	}

	// Create worker pool
	wpCfg := workerpool.Config{
		ProcessorCount: cfg.Workers.ProcessorCount,
		WriterCount:    cfg.Workers.WriterCount,
		RawBuf:         cfg.Buffer.RawChannelSize,
		ProcBuf:        cfg.Buffer.ProcessChannelSize,
	}
	wp := workerpool.New(wpCfg, pool, streamer)

	return &Server{
		config:     cfg,
		pool:       pool,
		apiServer:  apiServer,
		httpServer: httpServer,
		grpcServer: grpcServer,
		streamer:   streamer,
		workerPool: wp,
	}, nil
}

// Run starts all server components and blocks until shutdown.
func (s *Server) Run(ctx context.Context) error {
	// HTTP server errors
	httpErrCh := make(chan error, 1)
	// gRPC server errors
	grpcErrCh := make(chan error, 1)

	// Start HTTP server
	go func() {
		log.Printf("REST API running on %s", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			select {
			case httpErrCh <- err:
			default:
			}
		}
	}()

	// Start gRPC server
	go func() {
		lis, err := net.Listen("tcp", s.config.Server.GRPCAddr)
		if err != nil {
			select {
			case grpcErrCh <- err:
			default:
			}
			return
		}
		log.Printf("gRPC API running on %s", s.config.Server.GRPCAddr)
		if err := s.grpcServer.Serve(lis); err != nil {
			select {
			case grpcErrCh <- err:
			default:
			}
		}
	}()

	// Start worker pool
	g := s.workerPool.Start(ctx, httpErrCh)

	// Wait for shutdown signal or fatal error
	select {
	case <-ctx.Done():
		log.Println("shutdown requested")
	case err := <-httpErrCh:
		log.Printf("fatal HTTP error: %v", err)
	case err := <-grpcErrCh:
		log.Printf("fatal gRPC error: %v", err)
	}

	// Graceful shutdown
	if err := s.Shutdown(); err != nil {
		return err
	}

	// Wait for worker pool to finish
	if err := g.Wait(); err != nil {
		log.Printf("workerpool exited with error: %v", err)
	} else {
		log.Println("workerpool exited cleanly")
	}

	return nil
}

// Shutdown gracefully shuts down the server components.
func (s *Server) Shutdown() error {
	// gRPC server shutdown
	log.Println("shutting down gRPC server...")
	s.grpcServer.GracefulStop()
	log.Println("gRPC server shutdown complete")

	// HTTP server shutdown
	shutdownTimeout := time.Duration(s.config.Server.ShutdownTimeoutSec) * time.Second
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	} else {
		log.Println("http server shutdown complete")
	}

	// Database cleanup
	s.pool.Close()

	return nil
}
