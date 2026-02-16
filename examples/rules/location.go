package rules

import (
	"context"

	"github.com/TxWatchCore/txwatch/interfaces"
	"github.com/TxWatchCore/txwatch/models"
)

// LocationRule flags transactions from suspicious locations.
type LocationRule struct {
	suspiciousLocations map[string]bool
}

func NewLocationRule(suspiciousLocations []string) *LocationRule {
	locMap := make(map[string]bool)
	for _, loc := range suspiciousLocations {
		locMap[loc] = true
	}

	return &LocationRule{
		suspiciousLocations: locMap,
	}
}

func (r *LocationRule) ID() string {
	return "suspicious-location"
}

func (r *LocationRule) Name() string {
	return "Suspicious Location Rule"
}

func (r *LocationRule) Description() string {
	return "Flags transactions from known suspicious locations"
}

func (r *LocationRule) Evaluate(ctx context.Context, tx models.Transaction, historical []models.Transaction) (interfaces.RuleResult, error) {
	if tx.Metadata == nil {
		return interfaces.RuleResult{
			Flagged:      false,
			Reason:       "No location data available",
			ScoreContrib: 0.0,
		}, nil
	}

	location, ok := tx.Metadata["location"].(string)
	if !ok {
		return interfaces.RuleResult{
			Flagged:      false,
			Reason:       "No location data available",
			ScoreContrib: 0.0,
		}, nil
	}

	if r.suspiciousLocations[location] {
		return interfaces.RuleResult{
			Flagged:      true,
			Reason:       "Transaction from suspicious location",
			ScoreContrib: 0.9,
		}, nil
	}

	return interfaces.RuleResult{
		Flagged:      false,
		Reason:       "Location is not flagged",
		ScoreContrib: 0.0,
	}, nil
}
