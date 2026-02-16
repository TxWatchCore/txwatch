package models

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorTypes(t *testing.T) {
	t.Run("all errors are distinct", func(t *testing.T) {
		assert.NotEqual(t, ErrThresholdExceeded, ErrRepositoryUnavailable)
		assert.NotEqual(t, ErrThresholdExceeded, ErrInvalidTransaction)
		assert.NotEqual(t, ErrThresholdExceeded, ErrAIModelFailure)
		assert.NotEqual(t, ErrThresholdExceeded, ErrInvalidScore)
	})

	t.Run("errors have descriptive messages", func(t *testing.T) {
		assert.Contains(t, ErrThresholdExceeded.Error(), "threshold")
		assert.Contains(t, ErrRepositoryUnavailable.Error(), "repository")
		assert.Contains(t, ErrInvalidTransaction.Error(), "transaction")
		assert.Contains(t, ErrAIModelFailure.Error(), "AI")
		assert.Contains(t, ErrInvalidScore.Error(), "score")
	})
}

func TestErrorWrapping(t *testing.T) {
	t.Run("WrapRepositoryError preserves error type", func(t *testing.T) {
		wrapped := WrapRepositoryError(ErrRepositoryUnavailable, "fetching historical transactions")
		require.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, ErrRepositoryUnavailable))
		assert.Contains(t, wrapped.Error(), "fetching historical transactions")
	})

	t.Run("WrapRepositoryError returns nil for nil error", func(t *testing.T) {
		wrapped := WrapRepositoryError(nil, "operation")
		assert.NoError(t, wrapped)
	})

	t.Run("WrapAIModelError preserves error type", func(t *testing.T) {
		wrapped := WrapAIModelError(ErrAIModelFailure, "tx-123")
		require.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, ErrAIModelFailure))
		assert.Contains(t, wrapped.Error(), "tx-123")
		assert.Contains(t, wrapped.Error(), "AI scoring failed")
	})

	t.Run("WrapAIModelError returns nil for nil error", func(t *testing.T) {
		wrapped := WrapAIModelError(nil, "tx-123")
		assert.NoError(t, wrapped)
	})

	t.Run("WrapValidationError preserves error type", func(t *testing.T) {
		wrapped := WrapValidationError(ErrInvalidTransaction, "accountID")
		require.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, ErrInvalidTransaction))
		assert.Contains(t, wrapped.Error(), "accountID")
		assert.Contains(t, wrapped.Error(), "validation failed")
	})

	t.Run("WrapValidationError returns nil for nil error", func(t *testing.T) {
		wrapped := WrapValidationError(nil, "field")
		assert.NoError(t, wrapped)
	})

	t.Run("WrapRuleError preserves error type", func(t *testing.T) {
		originalErr := errors.New("insufficient data")
		wrapped := WrapRuleError(originalErr, "rule-1", "High Amount Rule")
		require.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, originalErr))
		assert.Contains(t, wrapped.Error(), "rule-1")
		assert.Contains(t, wrapped.Error(), "High Amount Rule")
	})

	t.Run("WrapRuleError returns nil for nil error", func(t *testing.T) {
		wrapped := WrapRuleError(nil, "rule-1", "Test Rule")
		assert.NoError(t, wrapped)
	})

	t.Run("WrapAlertError preserves error type", func(t *testing.T) {
		originalErr := errors.New("network timeout")
		wrapped := WrapAlertError(originalErr, "tx-456")
		require.Error(t, wrapped)
		assert.True(t, errors.Is(wrapped, originalErr))
		assert.Contains(t, wrapped.Error(), "tx-456")
		assert.Contains(t, wrapped.Error(), "alert failed")
	})

	t.Run("WrapAlertError returns nil for nil error", func(t *testing.T) {
		wrapped := WrapAlertError(nil, "tx-456")
		assert.NoError(t, wrapped)
	})
}

func TestNestedErrorWrapping(t *testing.T) {
	t.Run("double wrapped errors preserve original", func(t *testing.T) {
		err := ErrRepositoryUnavailable
		wrapped1 := WrapRepositoryError(err, "operation 1")
		wrapped2 := WrapRepositoryError(wrapped1, "operation 2")

		assert.True(t, errors.Is(wrapped2, ErrRepositoryUnavailable))
		assert.Contains(t, wrapped2.Error(), "operation 2")
		assert.Contains(t, wrapped2.Error(), "operation 1")
	})

	t.Run("unwrap retrieves original error", func(t *testing.T) {
		original := ErrInvalidScore
		wrapped := WrapValidationError(original, "score")

		unwrapped := errors.Unwrap(wrapped)
		assert.Equal(t, original, unwrapped)
	})
}
