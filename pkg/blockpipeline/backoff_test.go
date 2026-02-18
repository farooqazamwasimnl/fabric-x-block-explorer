/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package blockpipeline

import (
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/stretchr/testify/assert"
)

func TestNewBackoff(t *testing.T) {
	bo := NewBackoff()

	assert.NotNil(t, bo)

	// Verify it's an exponential backoff
	eb, ok := bo.(*backoff.ExponentialBackOff)
	assert.True(t, ok, "should be ExponentialBackOff")

	// Verify configuration
	assert.Equal(t, 500*time.Millisecond, eb.InitialInterval)
	assert.Equal(t, 30*time.Second, eb.MaxInterval)
	assert.Equal(t, time.Duration(0), eb.MaxElapsedTime, "should have no timeout")
}

func TestBackoffProgression(t *testing.T) {
	bo := NewBackoff()

	// First backoff should be around InitialInterval
	first := bo.NextBackOff()
	assert.Greater(t, first, time.Duration(0))
	assert.Less(t, first, 5*time.Second) // reasonable upper bound

	// Second should typically be longer (but with jitter, may vary)
	second := bo.NextBackOff()
	assert.Greater(t, second, time.Duration(0))

	// Should eventually stabilize near MaxInterval
	var last time.Duration
	for i := 0; i < 20; i++ {
		last = bo.NextBackOff()
	}
	// With jitter, should be around MaxInterval (allow up to 1.5x)
	assert.Greater(t, last, 10*time.Second)
	assert.Less(t, last, 60*time.Second)
}

func TestBackoffReset(t *testing.T) {
	bo := NewBackoff()

	// Progress through several intervals
	for i := 0; i < 5; i++ {
		bo.NextBackOff()
	}

	// Reset should start over
	bo.Reset()
	first := bo.NextBackOff()
	assert.Greater(t, first, time.Duration(0))
	assert.Less(t, first, 5*time.Second) // Should be back to initial range
}

func TestBackoffNoStop(t *testing.T) {
	bo := NewBackoff()

	// Since MaxElapsedTime is 0, should never return Stop
	for i := 0; i < 50; i++ {
		next := bo.NextBackOff()
		assert.NotEqual(t, backoff.Stop, next, "should never stop")
	}
}
