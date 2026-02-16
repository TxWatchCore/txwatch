package rules

import (
	"context"
	"time"

	"github.com/TxWatchCore/txwatch/interfaces"
	"github.com/TxWatchCore/txwatch/models"
)

// FrequencyRule flags accounts with too many transactions in a short time window.
type FrequencyRule struct {
	maxTransactions int
	window          time.Duration
}

func NewFrequencyRule(maxTransactions int, window time.Duration) *FrequencyRule {
	return &FrequencyRule{
		maxTransactions: maxTransactions,
		window:          window,
	}
}

func (r *FrequencyRule) ID() string {
	return "high-frequency"
}

func (r *FrequencyRule) Name() string {
	return "High Frequency Rule"
}

func (r *FrequencyRule) Description() string {
	return "Flags accounts with too many transactions within a short time window"
}

func (r *FrequencyRule) Evaluate(ctx context.Context, tx models.Transaction, historical []models.Transaction) (interfaces.RuleResult, error) {
	// Count transactions within the time window
	cutoff := tx.Timestamp.Add(-r.window)
	count := 1

	for _, h := range historical {
		if h.Timestamp.After(cutoff) {
			count++
		}
	}

	if count > r.maxTransactions {
		return interfaces.RuleResult{
			Flagged:      true,
			Reason:       "Too many transactions in short time window",
			ScoreContrib: 0.6,
		}, nil
	}

	return interfaces.RuleResult{
		Flagged:      false,
		Reason:       "Transaction frequency within normal range",
		ScoreContrib: 0.0,
	}, nil
}
