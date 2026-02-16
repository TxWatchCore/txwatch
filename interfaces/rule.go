package interfaces

import (
	"context"

	"github.com/TxWatchCore/txwatch/models"
)

// RuleResult represents the outcome of a rule evaluation.
type RuleResult struct {
	Flagged      bool
	Reason       string
	ScoreContrib float64
}

// Rule defines a fraud detection rule.
// Rules evaluate transactions independently and return their assessment.
type Rule interface {
	// ID returns the unique identifier for this rule.
	ID() string

	// Name returns a human-readable name for this rule.
	Name() string

	// Description returns a detailed description of what this rule checks.
	Description() string

	// Evaluate assesses a transaction for fraud based on this rule's logic.
	// The historical parameter provides context about the account's transaction history.
	// Returns a RuleResult indicating whether the transaction is flagged and why.
	Evaluate(ctx context.Context, tx models.Transaction, historical []models.Transaction) (RuleResult, error)
}
