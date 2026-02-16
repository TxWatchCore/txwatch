package models

import "errors"

var (
	ErrThresholdExceeded     = errors.New("fraud threshold exceeded")
	ErrRepositoryUnavailable = errors.New("transaction repository unavailable")
	ErrInvalidTransaction    = errors.New("invalid transaction")
	ErrAIModelFailure        = errors.New("AI model scoring failed")
	ErrInvalidScore          = errors.New("invalid score: must be between 0.0 and 1.0")
	ErrNoEvaluationMethods   = errors.New("no evaluation methods configured (no rules or AI model)")
	ErrRuleEngineFailure     = errors.New("rule engine failure: all rules failed")
)
