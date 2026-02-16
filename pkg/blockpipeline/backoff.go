/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"time"

	"github.com/cenkalti/backoff/v4"
)

// NewBackoff creates a new exponential backoff with jitter.
// Base: 500ms, Max: 30s, with exponential multiplier.
func NewBackoff() backoff.BackOff {
	eb := backoff.NewExponentialBackOff()
	eb.InitialInterval = 500 * time.Millisecond
	eb.MaxInterval = 30 * time.Second
	eb.MaxElapsedTime = 0 // no timeout
	return eb
}
