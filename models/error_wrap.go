package models

import "fmt"

func WrapRepositoryError(err error, operation string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func WrapAIModelError(err error, transactionID string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("AI scoring failed for transaction %s: %w", transactionID, err)
}

func WrapValidationError(err error, field string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("validation failed for %s: %w", field, err)
}

func WrapRuleError(err error, ruleID, ruleName string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("rule %s (%s) failed: %w", ruleID, ruleName, err)
}

func WrapAlertError(err error, transactionID string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("alert failed for transaction %s: %w", transactionID, err)
}
