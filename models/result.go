package models

import "time"

// Result represents the outcome of fraud detection for a transaction.
type Result struct {
	TransactionID string
	AccountID     string
	Flagged       bool
	Risk          Risk
	Timestamp     time.Time
	Reasons       []string
}

// Validate checks if the result has valid data.
func (r *Result) Validate() error {
	if r.TransactionID == "" {
		return ErrInvalidTransaction
	}
	if r.AccountID == "" {
		return ErrInvalidTransaction
	}
	if r.Timestamp.IsZero() {
		return ErrInvalidTransaction
	}
	return r.Risk.Validate()
}

// AddReason appends a reason to the result.
func (r *Result) AddReason(reason string) {
	r.Reasons = append(r.Reasons, reason)
}

// FlaggedBy returns a summary of what flagged this transaction.
func (r *Result) FlaggedBy() []string {
	var sources []string
	for _, rule := range r.Risk.FlaggedRules() {
		sources = append(sources, rule.RuleName)
	}
	if r.Risk.HasAIScore() && *r.Risk.AIScore > 0.5 {
		sources = append(sources, "AI Model")
	}
	return sources
}
