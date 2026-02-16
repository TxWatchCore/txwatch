// Package core implements the core fraud detection engine components.
package core

import (
	"context"
	"fmt"
	"sync"

	"github.com/TxWatchCore/txwatch/interfaces"
	"github.com/TxWatchCore/txwatch/models"
)

// RuleRegistry manages a collection of fraud detection rules.
// It provides thread-safe registration, retrieval, and execution of rules.
type RuleRegistry struct {
	mu    sync.RWMutex
	rules map[string]*ruleEntry
}

// ruleEntry wraps a rule with its enabled state.
type ruleEntry struct {
	rule    interfaces.Rule
	enabled bool
}

func NewRuleRegistry() *RuleRegistry {
	return &RuleRegistry{
		rules: make(map[string]*ruleEntry),
	}
}

func (r *RuleRegistry) Register(rule interfaces.Rule) error {
	if rule == nil {
		return fmt.Errorf("cannot register nil rule")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	id := rule.ID()
	if _, exists := r.rules[id]; exists {
		return fmt.Errorf("rule with ID %s already registered", id)
	}

	r.rules[id] = &ruleEntry{
		rule:    rule,
		enabled: true,
	}

	return nil
}

func (r *RuleRegistry) Unregister(ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.rules[ruleID]; !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	delete(r.rules, ruleID)
	return nil
}

func (r *RuleRegistry) Enable(ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	entry.enabled = true
	return nil
}

func (r *RuleRegistry) Disable(ruleID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	entry, exists := r.rules[ruleID]
	if !exists {
		return fmt.Errorf("rule with ID %s not found", ruleID)
	}

	entry.enabled = false
	return nil
}

func (r *RuleRegistry) IsEnabled(ruleID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.rules[ruleID]
	if !exists {
		return false
	}

	return entry.enabled
}

func (r *RuleRegistry) Get(ruleID string) interfaces.Rule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entry, exists := r.rules[ruleID]
	if !exists {
		return nil
	}

	return entry.rule
}

// List returns all registered rule IDs.
func (r *RuleRegistry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.rules))
	for id := range r.rules {
		ids = append(ids, id)
	}

	return ids
}

// ListEnabled returns IDs of all enabled rules.
func (r *RuleRegistry) ListEnabled() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ids := make([]string, 0, len(r.rules))
	for id, entry := range r.rules {
		if entry.enabled {
			ids = append(ids, id)
		}
	}

	return ids
}

func (r *RuleRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.rules)
}

// EvaluateAll evaluates all enabled rules against a transaction.
// Returns aggregated results from all rules.
// Continues evaluation even if individual rules fail.
func (r *RuleRegistry) EvaluateAll(ctx context.Context, tx models.Transaction, historical []models.Transaction) ([]models.RuleContribution, error) {
	r.mu.RLock()
	enabledRules := make([]interfaces.Rule, 0, len(r.rules))
	for _, entry := range r.rules {
		if entry.enabled {
			enabledRules = append(enabledRules, entry.rule)
		}
	}
	r.mu.RUnlock()

	if len(enabledRules) == 0 {
		return []models.RuleContribution{}, nil
	}

	contributions := make([]models.RuleContribution, 0, len(enabledRules))
	var evalErrors []error

	for _, rule := range enabledRules {
		result, err := rule.Evaluate(ctx, tx, historical)
		if err != nil {
			evalErrors = append(evalErrors, models.WrapRuleError(err, rule.ID(), rule.Name()))
			continue
		}

		contributions = append(contributions, models.RuleContribution{
			RuleID:       rule.ID(),
			RuleName:     rule.Name(),
			Flagged:      result.Flagged,
			Reason:       result.Reason,
			ScoreContrib: result.ScoreContrib,
		})
	}

	// If all rules failed, return error
	if len(contributions) == 0 && len(evalErrors) > 0 {
		return nil, models.ErrRuleEngineFailure
	}

	return contributions, nil
}
