/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package contracts

import (
	"context"

	"github.com/hyperledger/fabric-protos-go-apiv2/common"
)

// Streamer is the minimal interface workerpool needs from the sidecar wrapper.
type Streamer interface {
	// StartDeliver should start delivering blocks into the provided channel.
	// Implementations should return when ctx is cancelled or on fatal error.
	StartDeliver(ctx context.Context, out chan<- *common.Block) error
	// Close releases resources.
	Close() error
}
