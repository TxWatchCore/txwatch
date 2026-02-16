package rules

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TxWatchCore/txwatch/models"
)

func TestHighAmountRule(t *testing.T) {
	ctx := context.Background()

	t.Run("flags high amount transaction", func(t *testing.T) {
		rule := NewHighAmountRule(1000.0)
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    1500.0,
			Timestamp: time.Now(),
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.True(t, result.Flagged)
		assert.Contains(t, result.Reason, "exceeds threshold")
		assert.Greater(t, result.ScoreContrib, 0.0)
	})

	t.Run("passes normal amount transaction", func(t *testing.T) {
		rule := NewHighAmountRule(1000.0)
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    500.0,
			Timestamp: time.Now(),
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
		assert.Equal(t, 0.0, result.ScoreContrib)
	})

	t.Run("passes transaction at threshold", func(t *testing.T) {
		rule := NewHighAmountRule(1000.0)
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    1000.0,
			Timestamp: time.Now(),
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
	})

	t.Run("has correct metadata", func(t *testing.T) {
		rule := NewHighAmountRule(1000.0)
		assert.Equal(t, "high-amount", rule.ID())
		assert.Equal(t, "High Amount Rule", rule.Name())
		assert.NotEmpty(t, rule.Description())
	})
}

func TestFrequencyRule(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("flags high frequency transactions", func(t *testing.T) {
		rule := NewFrequencyRule(3, 1*time.Hour)

		// 4 transactions within 1 hour (including current)
		historical := []models.Transaction{
			{ID: "tx1", Timestamp: now.Add(-10 * time.Minute)},
			{ID: "tx2", Timestamp: now.Add(-20 * time.Minute)},
			{ID: "tx3", Timestamp: now.Add(-30 * time.Minute)},
		}

		tx := models.Transaction{
			ID:        "tx4",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: now,
		}

		result, err := rule.Evaluate(ctx, tx, historical)
		require.NoError(t, err)
		assert.True(t, result.Flagged)
		assert.Contains(t, result.Reason, "Too many transactions")
		assert.Greater(t, result.ScoreContrib, 0.0)
	})

	t.Run("passes normal frequency", func(t *testing.T) {
		rule := NewFrequencyRule(5, 1*time.Hour)

		// 3 transactions within 1 hour (including current)
		historical := []models.Transaction{
			{ID: "tx1", Timestamp: now.Add(-10 * time.Minute)},
			{ID: "tx2", Timestamp: now.Add(-20 * time.Minute)},
		}

		tx := models.Transaction{
			ID:        "tx3",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: now,
		}

		result, err := rule.Evaluate(ctx, tx, historical)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
		assert.Equal(t, 0.0, result.ScoreContrib)
	})

	t.Run("excludes old transactions", func(t *testing.T) {
		rule := NewFrequencyRule(3, 1*time.Hour)

		// 2 recent + 2 old transactions
		historical := []models.Transaction{
			{ID: "tx1", Timestamp: now.Add(-10 * time.Minute)},  // Within window
			{ID: "tx2", Timestamp: now.Add(-30 * time.Minute)},  // Within window
			{ID: "tx3", Timestamp: now.Add(-2 * time.Hour)},     // Outside window
			{ID: "tx4", Timestamp: now.Add(-3 * time.Hour)},     // Outside window
		}

		tx := models.Transaction{
			ID:        "tx5",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: now,
		}

		result, err := rule.Evaluate(ctx, tx, historical)
		require.NoError(t, err)
		assert.False(t, result.Flagged) // Only 3 within window (including current)
	})

	t.Run("has correct metadata", func(t *testing.T) {
		rule := NewFrequencyRule(5, 1*time.Hour)
		assert.Equal(t, "high-frequency", rule.ID())
		assert.Equal(t, "High Frequency Rule", rule.Name())
		assert.NotEmpty(t, rule.Description())
	})
}

func TestLocationRule(t *testing.T) {
	ctx := context.Background()

	t.Run("flags suspicious location", func(t *testing.T) {
		rule := NewLocationRule([]string{"Darknet", "Tor Exit Node"})
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"location": "Darknet",
			},
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.True(t, result.Flagged)
		assert.Contains(t, result.Reason, "suspicious location")
		assert.Greater(t, result.ScoreContrib, 0.0)
	})

	t.Run("passes safe location", func(t *testing.T) {
		rule := NewLocationRule([]string{"Darknet", "Tor Exit Node"})
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"location": "New York",
			},
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
		assert.Equal(t, 0.0, result.ScoreContrib)
	})

	t.Run("passes when no location data", func(t *testing.T) {
		rule := NewLocationRule([]string{"Darknet"})
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: time.Now(),
			Metadata:  nil,
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
		assert.Contains(t, result.Reason, "No location data")
	})

	t.Run("passes when location is not a string", func(t *testing.T) {
		rule := NewLocationRule([]string{"Darknet"})
		tx := models.Transaction{
			ID:        "tx1",
			AccountID: "acc1",
			Amount:    100.0,
			Timestamp: time.Now(),
			Metadata: map[string]interface{}{
				"location": 12345, // Not a string
			},
		}

		result, err := rule.Evaluate(ctx, tx, nil)
		require.NoError(t, err)
		assert.False(t, result.Flagged)
	})

	t.Run("has correct metadata", func(t *testing.T) {
		rule := NewLocationRule([]string{})
		assert.Equal(t, "suspicious-location", rule.ID())
		assert.Equal(t, "Suspicious Location Rule", rule.Name())
		assert.NotEmpty(t, rule.Description())
	})
}
