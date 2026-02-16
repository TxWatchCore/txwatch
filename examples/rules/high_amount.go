// Package rules provides example fraud detection rule implementations.
package rules

import (
	"context"

	"github.com/TxWatchCore/txwatch/interfaces"
	"github.com/TxWatchCore/txwatch/models"
)

// HighAmountRule flags transactions with amounts exceeding a threshold.
type HighAmountRule struct {
	threshold float64
}

func NewHighAmountRule(threshold float64) *HighAmountRule {
	return &HighAmountRule{
		threshold: threshold,
	}
}

func (r *HighAmountRule) ID() string {
	return "high-amount"
}

func (r *HighAmountRule) Name() string {
	return "High Amount Rule"
}

func (r *HighAmountRule) Description() string {
	return "Flags transactions with amounts exceeding the configured threshold"
}

func (r *HighAmountRule) Evaluate(ctx context.Context, tx models.Transaction, historical []models.Transaction) (interfaces.RuleResult, error) {
	if tx.Amount > r.threshold {
		return interfaces.RuleResult{
			Flagged:      true,
			Reason:       "Transaction amount exceeds threshold",
			ScoreContrib: 0.8,
		}, nil
	}

	return interfaces.RuleResult{
		Flagged:      false,
		Reason:       "Amount within normal range",
		ScoreContrib: 0.0,
	}, nil
}
