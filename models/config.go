package models

import (
	"errors"
	"time"
)

// AggregationType defines the strategy for aggregating risk scores.
type AggregationType int

const (
	MaxAggregation AggregationType = iota
	WeightedAverageAggregation
	TimeDecayedAggregation
)

// Config holds the configuration for the fraud detection service.
type Config struct {
	Threshold           float64
	AggregationStrategy AggregationType
	TimeWindow          time.Duration
	AsyncAlerts         bool
	EnableDeduplication bool
	MaxRetries          int
}

func (c *Config) Validate() error {
	if c.Threshold < 0.0 || c.Threshold > 1.0 {
		return errors.New("threshold must be between 0.0 and 1.0")
	}
	if c.TimeWindow <= 0 {
		return errors.New("time window must be positive")
	}
	if c.MaxRetries < 0 {
		return errors.New("max retries must be non-negative")
	}
	return nil
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() Config {
	return Config{
		Threshold:           0.7,
		AggregationStrategy: MaxAggregation,
		TimeWindow:          24 * time.Hour,
		AsyncAlerts:         false,
		EnableDeduplication: false,
		MaxRetries:          3,
	}
}
