/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package api

import (
	"context"
	"encoding/hex"

	pb "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/api/proto"
	dbsqlc "github.com/LF-Decentralized-Trust-labs/fabric-x-block-explorer/pkg/db/sqlc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GRPCServer implements the BlockExplorer gRPC service
type GRPCServer struct {
	pb.UnimplementedBlockExplorerServer
	api *API
}

// NewGRPCServer creates a new gRPC server instance
func NewGRPCServer(api *API) *GRPCServer {
	return &GRPCServer{api: api}
}

// GetBlockHeight returns the current block height
func (s *GRPCServer) GetBlockHeight(ctx context.Context, req *pb.BlockHeightRequest) (*pb.BlockHeightResponse, error) {
	height, err := s.api.q.GetBlockHeight(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get block height: %v", err)
	}

	h := height.(int64)
	return &pb.BlockHeightResponse{Height: h}, nil
}

// GetBlock returns block details by block number
func (s *GRPCServer) GetBlock(ctx context.Context, req *pb.GetBlockRequest) (*pb.BlockResponse, error) {
	blockNum := req.BlockNum

	limitTx := req.LimitTx
	if limitTx == 0 {
		limitTx = 100
	}
	offsetTx := req.OffsetTx

	limitWrites := req.LimitWrites
	if limitWrites == 0 {
		limitWrites = 1000
	}
	offsetWrites := req.OffsetWrites

	block, err := s.api.q.GetBlock(ctx, blockNum)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "block not found: %v", err)
	}

	txs, err := s.api.q.GetTransactionsByBlock(ctx, dbsqlc.GetTransactionsByBlockParams{
		BlockNum: blockNum,
		Limit:    limitTx,
		Offset:   offsetTx,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get transactions: %v", err)
	}

	resp := &pb.BlockResponse{
		BlockNum:     block.BlockNum,
		TxCount:      block.TxCount,
		PreviousHash: hex.EncodeToString(block.PreviousHash),
		DataHash:     hex.EncodeToString(block.DataHash),
		Transactions: make([]*pb.TransactionWithWrites, 0, len(txs)),
	}

	for _, tx := range txs {
		reads, _ := s.api.q.GetReadsByTx(ctx, dbsqlc.GetReadsByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
		})

		endorsements, _ := s.api.q.GetEndorsementsByTx(ctx, dbsqlc.GetEndorsementsByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
		})

		writes, _ := s.api.q.GetWritesByTx(ctx, dbsqlc.GetWritesByTxParams{
			BlockNum: tx.BlockNum,
			TxNum:    tx.TxNum,
			Limit:    limitWrites,
			Offset:   offsetWrites,
		})

		txResp := &pb.TransactionWithWrites{
			Id:             tx.ID,
			TxNum:          tx.TxNum,
			TxId:           hex.EncodeToString(tx.TxID),
			ValidationCode: tx.ValidationCode,
			Reads:          make([]*pb.ReadRecord, 0, len(reads)),
			Writes:         make([]*pb.WriteRecord, 0, len(writes)),
			Endorsements:   make([]*pb.EndorsementRecord, 0, len(endorsements)),
		}

		for _, r := range reads {
			var version *int64
			if r.Version.Valid {
				version = &r.Version.Int64
			}
			txResp.Reads = append(txResp.Reads, &pb.ReadRecord{
				Id:          r.ID,
				NsId:        r.NsID,
				Key:         hex.EncodeToString(r.Key),
				Version:     version,
				IsReadWrite: r.IsReadWrite,
			})
		}

		for _, w := range writes {
			var readVersion *int64
			if w.ReadVersion.Valid {
				readVersion = &w.ReadVersion.Int64
			}
			txResp.Writes = append(txResp.Writes, &pb.WriteRecord{
				Id:           w.ID,
				NsId:         w.NsID,
				Key:          hex.EncodeToString(w.Key),
				Value:        hex.EncodeToString(w.Value),
				IsBlindWrite: w.IsBlindWrite,
				ReadVersion:  readVersion,
			})
		}

		for _, e := range endorsements {
			var mspID *string
			if e.MspID.Valid {
				mspID = &e.MspID.String
			}
			var identity *string
			if len(e.Identity) > 0 {
				id := string(e.Identity)
				identity = &id
			}
			txResp.Endorsements = append(txResp.Endorsements, &pb.EndorsementRecord{
				Id:          e.ID,
				NsId:        e.NsID,
				Endorsement: hex.EncodeToString(e.Endorsement),
				MspId:       mspID,
				Identity:    identity,
			})
		}

		resp.Transactions = append(resp.Transactions, txResp)
	}

	return resp, nil
}

// GetTransaction returns transaction details by transaction ID
func (s *GRPCServer) GetTransaction(ctx context.Context, req *pb.GetTransactionRequest) (*pb.TransactionResponse, error) {
	txBytes, err := hex.DecodeString(req.TxId)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx_id: %v", err)
	}

	tx, err := s.api.q.GetTransactionByTxID(ctx, txBytes)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "transaction not found: %v", err)
	}

	block, err := s.api.q.GetBlock(ctx, tx.BlockNum)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get block: %v", err)
	}

	reads, _ := s.api.q.GetReadsByTx(ctx, dbsqlc.GetReadsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	endorsements, _ := s.api.q.GetEndorsementsByTx(ctx, dbsqlc.GetEndorsementsByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
	})

	writes, _ := s.api.q.GetWritesByTx(ctx, dbsqlc.GetWritesByTxParams{
		BlockNum: tx.BlockNum,
		TxNum:    tx.TxNum,
		Limit:    1000,
		Offset:   0,
	})

	txResp := &pb.TransactionWithWrites{
		Id:             tx.ID,
		TxNum:          tx.TxNum,
		TxId:           hex.EncodeToString(tx.TxID),
		ValidationCode: tx.ValidationCode,
		Reads:          make([]*pb.ReadRecord, 0, len(reads)),
		Writes:         make([]*pb.WriteRecord, 0, len(writes)),
		Endorsements:   make([]*pb.EndorsementRecord, 0, len(endorsements)),
	}

	for _, r := range reads {
		var version *int64
		if r.Version.Valid {
			version = &r.Version.Int64
		}
		txResp.Reads = append(txResp.Reads, &pb.ReadRecord{
			Id:          r.ID,
			NsId:        r.NsID,
			Key:         hex.EncodeToString(r.Key),
			Version:     version,
			IsReadWrite: r.IsReadWrite,
		})
	}

	for _, w := range writes {
		var readVersion *int64
		if w.ReadVersion.Valid {
			readVersion = &w.ReadVersion.Int64
		}
		txResp.Writes = append(txResp.Writes, &pb.WriteRecord{
			Id:           w.ID,
			NsId:         w.NsID,
			Key:          hex.EncodeToString(w.Key),
			Value:        hex.EncodeToString(w.Value),
			IsBlindWrite: w.IsBlindWrite,
			ReadVersion:  readVersion,
		})
	}

	for _, e := range endorsements {
		var mspID *string
		if e.MspID.Valid {
			mspID = &e.MspID.String
		}
		var identity *string
		if len(e.Identity) > 0 {
			id := string(e.Identity)
			identity = &id
		}
		txResp.Endorsements = append(txResp.Endorsements, &pb.EndorsementRecord{
			Id:          e.ID,
			NsId:        e.NsID,
			Endorsement: hex.EncodeToString(e.Endorsement),
			MspId:       mspID,
			Identity:    identity,
		})
	}

	return &pb.TransactionResponse{
		Transaction: txResp,
		Block: &pb.BlockHeader{
			BlockNum:     block.BlockNum,
			TxCount:      block.TxCount,
			PreviousHash: hex.EncodeToString(block.PreviousHash),
			DataHash:     hex.EncodeToString(block.DataHash),
		},
	}, nil
}

// HealthCheck returns service health status
func (s *GRPCServer) HealthCheck(ctx context.Context, req *pb.HealthRequest) (*pb.HealthResponse, error) {
	if s.api.pool != nil {
		if err := s.api.pool.Ping(ctx); err != nil {
			return &pb.HealthResponse{
				Status:  "unavailable",
				Details: "db ping failed: " + err.Error(),
			}, nil
		}
	}

	return &pb.HealthResponse{Status: "ok"}, nil
}
