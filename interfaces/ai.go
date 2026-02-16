package interfaces

import (
	"context"

	"github.com/TxWatchCore/txwatch/models"
)

// AIModel provides AI-based fraud scoring.
// Implementations can use any ML framework (TensorFlow, PyTorch, custom algorithms, etc.).
type AIModel interface {
	// Score computes a risk score for the transaction based on current and historical data.
	// The score MUST be between 0.0 (no risk) and 1.0 (maximum risk) inclusive.
	// Returns an error if scoring fails or if the score is outside the valid range.
	//
	// The context can be used for cancellation and timeout control.
	// Implementations SHOULD be stateless and thread-safe for concurrent scoring.
	Score(ctx context.Context, tx models.Transaction, historical []models.Transaction) (float64, error)
}
