// Package models defines the core data models for TxWatch fraud detection.
package models

import (
	"errors"
	"time"
)

type Transaction struct {
	ID        string
	AccountID string
	Amount    float64
	Timestamp time.Time

	// Metadata contains additional transaction data (e.g., merchant, location, category).
	// This can be used by custom rules or AI models for fraud detection.
	Metadata map[string]interface{}
}

func (t *Transaction) Validate() error {
	if t.ID == "" {
		return errors.New("transaction ID is required")
	}
	if t.AccountID == "" {
		return errors.New("account ID is required")
	}
	if t.Amount < 0 {
		return errors.New("transaction amount must be non-negative")
	}
	if t.Timestamp.IsZero() {
		return errors.New("transaction timestamp is required")
	}
	return nil
}

func (t *Transaction) NormalizeTimestamp() {
	t.Timestamp = t.Timestamp.UTC()
}
