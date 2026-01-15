/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"math"
	"math/rand"
	"time"
)

type Backoff struct {
	base     time.Duration
	max      time.Duration
	attempts int
}

func NewBackoff() *Backoff {
	return &Backoff{
		base: 500 * time.Millisecond,
		max:  30 * time.Second,
	}
}

func (b *Backoff) Reset() {
	b.attempts = 0
}

func (b *Backoff) Next() time.Duration {
	exp := float64(b.base) * math.Pow(2, float64(b.attempts))
	if exp > float64(b.max) {
		exp = float64(b.max)
	}
	b.attempts++

	jitter := rand.Float64()*0.3 + 0.85 // 0.85–1.15
	return time.Duration(exp * jitter)
}
