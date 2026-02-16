// Package testutil provides common testing utilities for TxWatch.
// Use github.com/stretchr/testify/assert and require packages for assertions.
package testutil

import (
	"time"
)

// FixedTime returns a fixed time for deterministic testing.
func FixedTime() time.Time {
	return time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
}

// TimeRange returns a start and end time for testing time windows.
func TimeRange(start time.Time, duration time.Duration) (time.Time, time.Time) {
	return start, start.Add(duration)
}
