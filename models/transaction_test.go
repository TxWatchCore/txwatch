package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransaction_Validate(t *testing.T) {
	t.Run("valid transaction", func(t *testing.T) {
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    100.50,
			Timestamp: time.Now(),
			Metadata:  map[string]interface{}{"merchant": "Store A"},
		}
		err := tx.Validate()
		assert.NoError(t, err)
	})

	t.Run("missing ID", func(t *testing.T) {
		tx := Transaction{
			AccountID: "acc-456",
			Amount:    100.50,
			Timestamp: time.Now(),
		}
		err := tx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "transaction ID is required")
	})

	t.Run("missing account ID", func(t *testing.T) {
		tx := Transaction{
			ID:        "tx-123",
			Amount:    100.50,
			Timestamp: time.Now(),
		}
		err := tx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "account ID is required")
	})

	t.Run("negative amount", func(t *testing.T) {
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    -50.00,
			Timestamp: time.Now(),
		}
		err := tx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "amount must be non-negative")
	})

	t.Run("zero timestamp", func(t *testing.T) {
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    100.50,
		}
		err := tx.Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "timestamp is required")
	})

	t.Run("zero amount is valid", func(t *testing.T) {
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    0,
			Timestamp: time.Now(),
		}
		err := tx.Validate()
		assert.NoError(t, err)
	})
}

func TestTransaction_NormalizeTimestamp(t *testing.T) {
	t.Run("converts to UTC", func(t *testing.T) {
		// Create a timestamp in a non-UTC timezone
		loc, err := time.LoadLocation("America/New_York")
		require.NoError(t, err)

		localTime := time.Date(2024, 1, 1, 12, 0, 0, 0, loc)
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    100.00,
			Timestamp: localTime,
		}

		assert.Equal(t, "America/New_York", tx.Timestamp.Location().String())

		tx.NormalizeTimestamp()

		assert.Equal(t, "UTC", tx.Timestamp.Location().String())
	})

	t.Run("already UTC remains UTC", func(t *testing.T) {
		utcTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
		tx := Transaction{
			ID:        "tx-123",
			AccountID: "acc-456",
			Amount:    100.00,
			Timestamp: utcTime,
		}

		tx.NormalizeTimestamp()

		assert.Equal(t, "UTC", tx.Timestamp.Location().String())
		assert.Equal(t, utcTime.Unix(), tx.Timestamp.Unix())
	})
}
