package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfig_Validate(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		config := Config{
			Threshold:           0.7,
			AggregationStrategy: MaxAggregation,
			TimeWindow:          24 * time.Hour,
			MaxRetries:          3,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("threshold below range", func(t *testing.T) {
		config := Config{
			Threshold:  -0.1,
			TimeWindow: 24 * time.Hour,
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "threshold must be between 0.0 and 1.0")
	})

	t.Run("threshold above range", func(t *testing.T) {
		config := Config{
			Threshold:  1.5,
			TimeWindow: 24 * time.Hour,
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "threshold must be between 0.0 and 1.0")
	})

	t.Run("zero time window", func(t *testing.T) {
		config := Config{
			Threshold:  0.7,
			TimeWindow: 0,
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "time window must be positive")
	})

	t.Run("negative time window", func(t *testing.T) {
		config := Config{
			Threshold:  0.7,
			TimeWindow: -1 * time.Hour,
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "time window must be positive")
	})

	t.Run("negative max retries", func(t *testing.T) {
		config := Config{
			Threshold:  0.7,
			TimeWindow: 24 * time.Hour,
			MaxRetries: -1,
		}
		err := config.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "max retries must be non-negative")
	})

	t.Run("zero max retries is valid", func(t *testing.T) {
		config := Config{
			Threshold:  0.7,
			TimeWindow: 24 * time.Hour,
			MaxRetries: 0,
		}
		err := config.Validate()
		assert.NoError(t, err)
	})

	t.Run("threshold at boundaries", func(t *testing.T) {
		// Test 0.0
		config := Config{
			Threshold:  0.0,
			TimeWindow: 24 * time.Hour,
		}
		assert.NoError(t, config.Validate())

		// Test 1.0
		config.Threshold = 1.0
		assert.NoError(t, config.Validate())
	})
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	t.Run("has valid defaults", func(t *testing.T) {
		assert.Equal(t, 0.7, config.Threshold)
		assert.Equal(t, MaxAggregation, config.AggregationStrategy)
		assert.Equal(t, 24*time.Hour, config.TimeWindow)
		assert.False(t, config.AsyncAlerts)
		assert.False(t, config.EnableDeduplication)
		assert.Equal(t, 3, config.MaxRetries)
	})

	t.Run("default config is valid", func(t *testing.T) {
		err := config.Validate()
		assert.NoError(t, err)
	})
}

func TestAggregationType(t *testing.T) {
	t.Run("enum values are unique", func(t *testing.T) {
		assert.NotEqual(t, MaxAggregation, WeightedAverageAggregation)
		assert.NotEqual(t, MaxAggregation, TimeDecayedAggregation)
		assert.NotEqual(t, WeightedAverageAggregation, TimeDecayedAggregation)
	})
}
