/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package workerpool

import (
	"context"
	"sync"
	"time"

	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/blockpipeline"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/logging"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/sidecarstream"
	"github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/types"
	"github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"
)

var logger = logging.New("workerpool")

// Config controls pool sizes and buffer sizes.
type Config struct {
	ProcessorCount int
	WriterCount    int
	RawBuf         int
	ProcBuf        int
}

// Pool encapsulates channels and configuration.
type Pool struct {
	cfg      Config
	rawCh    chan *common.Block
	procCh   chan *types.ProcessedBlock
	pool     *pgxpool.Pool
	streamer *sidecarstream.Streamer
}

// New constructs a Pool. pool and streamer are injected.
func New(cfg Config, pool *pgxpool.Pool, streamer *sidecarstream.Streamer) *Pool {
	// sensible defaults
	if cfg.RawBuf <= 0 {
		cfg.RawBuf = 64
	}
	if cfg.ProcBuf <= 0 {
		cfg.ProcBuf = 256
	}
	if cfg.ProcessorCount <= 0 {
		cfg.ProcessorCount = 2
	}
	if cfg.WriterCount <= 0 {
		cfg.WriterCount = 2
	}

	return &Pool{
		cfg:      cfg,
		rawCh:    make(chan *common.Block, cfg.RawBuf),
		procCh:   make(chan *types.ProcessedBlock, cfg.ProcBuf),
		pool:     pool,
		streamer: streamer,
	}
}

// Start runs the block processing pipeline.
func (p *Pool) Start(ctx context.Context, errCh chan<- error) *errgroup.Group {
	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		defer close(p.rawCh)
		blockpipeline.BlockReceiver(ctx, p.streamer, p.rawCh, errCh, 0)
		return nil
	})

	var procWg sync.WaitGroup
	for i := 0; i < p.cfg.ProcessorCount; i++ {
		workerID := i
		procWg.Add(1)
		g.Go(func() error {
			defer procWg.Done()
			logger.Infof("processor[%d] started", workerID)
			blockpipeline.BlockProcessor(ctx, p.rawCh, p.procCh, errCh)
			logger.Infof("processor[%d] stopped", workerID)
			return nil
		})
	}

	// Close procCh once all processors are done.
	g.Go(func() error {
		// Wait for processors to finish, then close procCh.
		procWg.Wait()
		// small grace period to allow last pushes
		time.Sleep(50 * time.Millisecond)
		close(p.procCh)
		return nil
	})

	for i := 0; i < p.cfg.WriterCount; i++ {
		workerID := i
		g.Go(func() error {
			logger.Infof("writer[%d] started", workerID)
			conn, err := p.pool.Acquire(context.Background())
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return err
			}
			defer func() {
				conn.Release()
			}()

			// Create a per-connection BlockWriter.
			writer := db.NewBlockWriterFromConn(conn)

			// Consume processed blocks until procCh is closed or ctx cancelled.
			for {
				select {
				case <-ctx.Done():
					// On cancellation, drain remaining items best-effort with a short timeout.
					drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					for {
						select {
						case pb, ok := <-p.procCh:
							if !ok {
								cancel()
								logger.Infof("writer[%d] drained and stopping", workerID)
								return nil
							}
							if err := writer.WriteProcessedBlock(drainCtx, pb); err != nil {
								select {
								case errCh <- err:
								default:
								}
							}
						default:
							cancel()
								logger.Infof("writer[%d] stopping due to context cancellation", workerID)
							return nil
						}
					}
				case pb, ok := <-p.procCh:
					if !ok {
						// Channel closed: no more work
						logger.Infof("writer[%d] finished (procCh closed)", workerID)
						return nil
					}
					// Write the processed block using the per-connection writer.
					if err := writer.WriteProcessedBlock(ctx, pb); err != nil {
						select {
						case errCh <- err:
						default:
						}
						return err
					}
				}
			}
		})
	}

	return g
}
