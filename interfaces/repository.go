// Package interfaces defines the contracts for pluggable components in TxWatch.
package interfaces

import (
	"context"
	"time"

	"github.com/TxWatchCore/txwatch/models"
)

// TransactionRepository provides access to transaction data.
// Implementations can use any storage backend (SQL, NoSQL, in-memory, etc.).
type TransactionRepository interface {
	// FetchHistorical retrieves historical transactions for an account within a time window.
	// The window parameter specifies how far back to look from the current time.
	// Returns an empty slice if no historical transactions are found.
	FetchHistorical(ctx context.Context, accountID string, window time.Duration) ([]models.Transaction, error)

	// Persist stores flagged transactions for reporting and risk aggregation.
	// Only transactions that are flagged or have elevated risk scores will be persisted.
	// Implementations should handle idempotency to avoid duplicate storage.
	Persist(ctx context.Context, result models.Result) error
}
