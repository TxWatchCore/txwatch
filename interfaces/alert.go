package interfaces

import (
	"context"

	"github.com/TxWatchCore/txwatch/models"
)

// AlertHandler provides notification capabilities for flagged transactions.
// Implementations can use any notification channel (email, Slack, PagerDuty, SMS, etc.).
type AlertHandler interface {
	// Alert sends a notification for a flagged transaction.
	// The result parameter contains comprehensive information about the fraud detection outcome.
	//
	// Implementations SHOULD be idempotent to handle retry scenarios.
	// Returns an error if the alert cannot be delivered.
	Alert(ctx context.Context, result models.Result) error
}
