package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResult_Validate(t *testing.T) {
	t.Run("valid result", func(t *testing.T) {
		result := Result{
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Flagged:       true,
			Risk: Risk{
				Score:         0.8,
				TransactionID: "tx-123",
				AccountID:     "acc-456",
				Timestamp:     time.Now(),
			},
			Timestamp: time.Now(),
			Reasons:   []string{"High amount", "Suspicious location"},
		}
		err := result.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing transaction ID", func(t *testing.T) {
		result := Result{
			AccountID: "acc-456",
			Timestamp: time.Now(),
			Risk: Risk{
				Score:         0.8,
				TransactionID: "tx-123",
				AccountID:     "acc-456",
				Timestamp:     time.Now(),
			},
		}
		err := result.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("missing account ID", func(t *testing.T) {
		result := Result{
			TransactionID: "tx-123",
			Timestamp:     time.Now(),
			Risk: Risk{
				Score:         0.8,
				TransactionID: "tx-123",
				AccountID:     "acc-456",
				Timestamp:     time.Now(),
			},
		}
		err := result.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("zero timestamp", func(t *testing.T) {
		result := Result{
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Risk: Risk{
				Score:         0.8,
				TransactionID: "tx-123",
				AccountID:     "acc-456",
				Timestamp:     time.Now(),
			},
		}
		err := result.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("invalid risk propagates", func(t *testing.T) {
		result := Result{
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
			Risk: Risk{
				Score:         1.5, // Invalid
				TransactionID: "tx-123",
				AccountID:     "acc-456",
				Timestamp:     time.Now(),
			},
		}
		err := result.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScore)
	})
}

func TestResult_AddReason(t *testing.T) {
	t.Run("adds reason to empty list", func(t *testing.T) {
		result := Result{Reasons: []string{}}
		result.AddReason("High amount")
		require.Len(t, result.Reasons, 1)
		assert.Equal(t, "High amount", result.Reasons[0])
	})

	t.Run("appends to existing reasons", func(t *testing.T) {
		result := Result{Reasons: []string{"Reason 1"}}
		result.AddReason("Reason 2")
		result.AddReason("Reason 3")
		require.Len(t, result.Reasons, 3)
		assert.Equal(t, "Reason 1", result.Reasons[0])
		assert.Equal(t, "Reason 2", result.Reasons[1])
		assert.Equal(t, "Reason 3", result.Reasons[2])
	})
}

func TestResult_FlaggedBy(t *testing.T) {
	t.Run("returns rules that flagged", func(t *testing.T) {
		result := Result{
			Risk: Risk{
				ContributingRules: []RuleContribution{
					{RuleName: "High Amount", Flagged: true},
					{RuleName: "Normal", Flagged: false},
					{RuleName: "Suspicious Location", Flagged: true},
				},
			},
		}

		sources := result.FlaggedBy()
		require.Len(t, sources, 2)
		assert.Contains(t, sources, "High Amount")
		assert.Contains(t, sources, "Suspicious Location")
	})

	t.Run("includes AI when score high", func(t *testing.T) {
		aiScore := 0.9
		result := Result{
			Risk: Risk{
				AIScore: &aiScore,
				ContributingRules: []RuleContribution{
					{RuleName: "High Amount", Flagged: true},
				},
			},
		}

		sources := result.FlaggedBy()
		require.Len(t, sources, 2)
		assert.Contains(t, sources, "High Amount")
		assert.Contains(t, sources, "AI Model")
	})

	t.Run("excludes AI when score low", func(t *testing.T) {
		aiScore := 0.3
		result := Result{
			Risk: Risk{
				AIScore: &aiScore,
				ContributingRules: []RuleContribution{
					{RuleName: "High Amount", Flagged: true},
				},
			},
		}

		sources := result.FlaggedBy()
		require.Len(t, sources, 1)
		assert.Contains(t, sources, "High Amount")
		assert.NotContains(t, sources, "AI Model")
	})

	t.Run("returns empty when nothing flagged", func(t *testing.T) {
		result := Result{
			Risk: Risk{
				ContributingRules: []RuleContribution{},
			},
		}

		sources := result.FlaggedBy()
		assert.Empty(t, sources)
	})
}
