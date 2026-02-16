package interfaces

// RiskSignal represents an individual risk score input for aggregation.
type RiskSignal struct {
	Score     float64
	Weight    float64
	Source    string
	Timestamp int64
}

// AggregationStrategy defines how multiple risk signals are combined into a single score.
// Users can implement custom strategies while TxWatch handles the orchestration.
type AggregationStrategy interface {
	// Aggregate combines multiple risk signals into a single risk score.
	// The returned score MUST be between 0.0 and 1.0 inclusive.
	Aggregate(inputs []RiskSignal) float64
}
