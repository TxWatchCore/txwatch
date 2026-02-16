package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRisk_Validate(t *testing.T) {
	t.Run("valid risk", func(t *testing.T) {
		aiScore := 0.75
		risk := Risk{
			Score:         0.8,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
			AIScore:       &aiScore,
		}
		err := risk.Validate()
		assert.NoError(t, err)
	})

	t.Run("score below range", func(t *testing.T) {
		risk := Risk{
			Score:         -0.1,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScore)
	})

	t.Run("score above range", func(t *testing.T) {
		risk := Risk{
			Score:         1.5,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScore)
	})

	t.Run("missing transaction ID", func(t *testing.T) {
		risk := Risk{
			Score:     0.5,
			AccountID: "acc-456",
			Timestamp: time.Now(),
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("missing account ID", func(t *testing.T) {
		risk := Risk{
			Score:         0.5,
			TransactionID: "tx-123",
			Timestamp:     time.Now(),
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidTransaction)
	})

	t.Run("AI score below range", func(t *testing.T) {
		aiScore := -0.1
		risk := Risk{
			Score:         0.5,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
			AIScore:       &aiScore,
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScore)
	})

	t.Run("AI score above range", func(t *testing.T) {
		aiScore := 1.5
		risk := Risk{
			Score:         0.5,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
			AIScore:       &aiScore,
		}
		err := risk.Validate()
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidScore)
	})

	t.Run("nil AI score is valid", func(t *testing.T) {
		risk := Risk{
			Score:         0.5,
			TransactionID: "tx-123",
			AccountID:     "acc-456",
			Timestamp:     time.Now(),
			AIScore:       nil,
		}
		err := risk.Validate()
		assert.NoError(t, err)
	})
}

func TestRisk_HasAIScore(t *testing.T) {
	t.Run("returns true when AI score present", func(t *testing.T) {
		aiScore := 0.75
		risk := Risk{AIScore: &aiScore}
		assert.True(t, risk.HasAIScore())
	})

	t.Run("returns false when AI score nil", func(t *testing.T) {
		risk := Risk{AIScore: nil}
		assert.False(t, risk.HasAIScore())
	})
}

func TestRisk_FlaggedRules(t *testing.T) {
	t.Run("returns only flagged rules", func(t *testing.T) {
		risk := Risk{
			ContributingRules: []RuleContribution{
				{RuleID: "rule1", RuleName: "High Amount", Flagged: true},
				{RuleID: "rule2", RuleName: "Normal Transaction", Flagged: false},
				{RuleID: "rule3", RuleName: "Suspicious Location", Flagged: true},
			},
		}

		flagged := risk.FlaggedRules()
		require.Len(t, flagged, 2)
		assert.Equal(t, "rule1", flagged[0].RuleID)
		assert.Equal(t, "rule3", flagged[1].RuleID)
	})

	t.Run("returns empty when no rules flagged", func(t *testing.T) {
		risk := Risk{
			ContributingRules: []RuleContribution{
				{RuleID: "rule1", Flagged: false},
				{RuleID: "rule2", Flagged: false},
			},
		}

		flagged := risk.FlaggedRules()
		assert.Empty(t, flagged)
	})

	t.Run("returns empty when no rules", func(t *testing.T) {
		risk := Risk{ContributingRules: []RuleContribution{}}
		flagged := risk.FlaggedRules()
		assert.Empty(t, flagged)
	})
}
