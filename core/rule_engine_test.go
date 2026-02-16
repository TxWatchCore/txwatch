package core

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TxWatchCore/txwatch/interfaces"
	"github.com/TxWatchCore/txwatch/models"
)

// mockRule is a test implementation of interfaces.Rule
type mockRule struct {
	id          string
	name        string
	description string
	result      interfaces.RuleResult
	err         error
}

func (m *mockRule) ID() string                        { return m.id }
func (m *mockRule) Name() string                      { return m.name }
func (m *mockRule) Description() string               { return m.description }
func (m *mockRule) Evaluate(ctx context.Context, tx models.Transaction, historical []models.Transaction) (interfaces.RuleResult, error) {
	return m.result, m.err
}

func TestNewRuleRegistry(t *testing.T) {
	registry := NewRuleRegistry()
	assert.NotNil(t, registry)
	assert.Equal(t, 0, registry.Count())
}

func TestRuleRegistry_Register(t *testing.T) {
	t.Run("registers new rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		err := registry.Register(rule)
		assert.NoError(t, err)
		assert.Equal(t, 1, registry.Count())
	})

	t.Run("prevents duplicate registration", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{id: "rule1", name: "Test Rule 1"}
		rule2 := &mockRule{id: "rule1", name: "Test Rule 2"}

		err := registry.Register(rule1)
		require.NoError(t, err)

		err = registry.Register(rule2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
		assert.Equal(t, 1, registry.Count())
	})

	t.Run("rejects nil rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		err := registry.Register(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "nil rule")
	})

	t.Run("rule is enabled by default", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		err := registry.Register(rule)
		require.NoError(t, err)
		assert.True(t, registry.IsEnabled("rule1"))
	})
}

func TestRuleRegistry_Unregister(t *testing.T) {
	t.Run("unregisters existing rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		registry.Register(rule)
		assert.Equal(t, 1, registry.Count())

		err := registry.Unregister("rule1")
		assert.NoError(t, err)
		assert.Equal(t, 0, registry.Count())
	})

	t.Run("returns error for non-existent rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		err := registry.Unregister("nonexistent")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestRuleRegistry_EnableDisable(t *testing.T) {
	t.Run("disables enabled rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		registry.Register(rule)
		assert.True(t, registry.IsEnabled("rule1"))

		err := registry.Disable("rule1")
		assert.NoError(t, err)
		assert.False(t, registry.IsEnabled("rule1"))
	})

	t.Run("enables disabled rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		registry.Register(rule)
		registry.Disable("rule1")
		assert.False(t, registry.IsEnabled("rule1"))

		err := registry.Enable("rule1")
		assert.NoError(t, err)
		assert.True(t, registry.IsEnabled("rule1"))
	})

	t.Run("returns error for non-existent rule", func(t *testing.T) {
		registry := NewRuleRegistry()

		err := registry.Enable("nonexistent")
		require.Error(t, err)

		err = registry.Disable("nonexistent")
		require.Error(t, err)
	})

	t.Run("IsEnabled returns false for non-existent rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		assert.False(t, registry.IsEnabled("nonexistent"))
	})
}

func TestRuleRegistry_Get(t *testing.T) {
	t.Run("retrieves registered rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}

		registry.Register(rule)
		retrieved := registry.Get("rule1")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "rule1", retrieved.ID())
	})

	t.Run("returns nil for non-existent rule", func(t *testing.T) {
		registry := NewRuleRegistry()
		retrieved := registry.Get("nonexistent")
		assert.Nil(t, retrieved)
	})
}

func TestRuleRegistry_List(t *testing.T) {
	t.Run("lists all rule IDs", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{id: "rule1", name: "Test Rule 1"}
		rule2 := &mockRule{id: "rule2", name: "Test Rule 2"}

		registry.Register(rule1)
		registry.Register(rule2)

		ids := registry.List()
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, "rule1")
		assert.Contains(t, ids, "rule2")
	})

	t.Run("returns empty slice for empty registry", func(t *testing.T) {
		registry := NewRuleRegistry()
		ids := registry.List()
		assert.Empty(t, ids)
	})
}

func TestRuleRegistry_ListEnabled(t *testing.T) {
	t.Run("lists only enabled rules", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{id: "rule1", name: "Test Rule 1"}
		rule2 := &mockRule{id: "rule2", name: "Test Rule 2"}
		rule3 := &mockRule{id: "rule3", name: "Test Rule 3"}

		registry.Register(rule1)
		registry.Register(rule2)
		registry.Register(rule3)
		registry.Disable("rule2")

		ids := registry.ListEnabled()
		assert.Len(t, ids, 2)
		assert.Contains(t, ids, "rule1")
		assert.Contains(t, ids, "rule3")
		assert.NotContains(t, ids, "rule2")
	})
}

func TestRuleRegistry_EvaluateAll(t *testing.T) {
	ctx := context.Background()
	tx := models.Transaction{ID: "tx1", AccountID: "acc1", Amount: 100}

	t.Run("evaluates all enabled rules", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{
			id:   "rule1",
			name: "Rule 1",
			result: interfaces.RuleResult{
				Flagged:      true,
				Reason:       "Test reason 1",
				ScoreContrib: 0.5,
			},
		}
		rule2 := &mockRule{
			id:   "rule2",
			name: "Rule 2",
			result: interfaces.RuleResult{
				Flagged:      false,
				Reason:       "Test reason 2",
				ScoreContrib: 0.0,
			},
		}

		registry.Register(rule1)
		registry.Register(rule2)

		contributions, err := registry.EvaluateAll(ctx, tx, nil)
		assert.NoError(t, err)
		assert.Len(t, contributions, 2)
	})

	t.Run("skips disabled rules", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{id: "rule1", name: "Rule 1", result: interfaces.RuleResult{Flagged: true}}
		rule2 := &mockRule{id: "rule2", name: "Rule 2", result: interfaces.RuleResult{Flagged: false}}

		registry.Register(rule1)
		registry.Register(rule2)
		registry.Disable("rule2")

		contributions, err := registry.EvaluateAll(ctx, tx, nil)
		assert.NoError(t, err)
		assert.Len(t, contributions, 1)
		assert.Equal(t, "rule1", contributions[0].RuleID)
	})

	t.Run("continues evaluation if one rule fails", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{
			id:   "rule1",
			name: "Rule 1",
			err:  assert.AnError,
		}
		rule2 := &mockRule{
			id:   "rule2",
			name: "Rule 2",
			result: interfaces.RuleResult{Flagged: true},
		}

		registry.Register(rule1)
		registry.Register(rule2)

		contributions, err := registry.EvaluateAll(ctx, tx, nil)
		assert.NoError(t, err)
		assert.Len(t, contributions, 1)
		assert.Equal(t, "rule2", contributions[0].RuleID)
	})

	t.Run("returns error if all rules fail", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule1 := &mockRule{id: "rule1", name: "Rule 1", err: assert.AnError}
		rule2 := &mockRule{id: "rule2", name: "Rule 2", err: assert.AnError}

		registry.Register(rule1)
		registry.Register(rule2)

		contributions, err := registry.EvaluateAll(ctx, tx, nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, models.ErrRuleEngineFailure)
		assert.Nil(t, contributions)
	})

	t.Run("returns empty contributions for no rules", func(t *testing.T) {
		registry := NewRuleRegistry()

		contributions, err := registry.EvaluateAll(ctx, tx, nil)
		assert.NoError(t, err)
		assert.Empty(t, contributions)
	})
}

func TestRuleRegistry_Concurrency(t *testing.T) {
	t.Run("concurrent registration", func(t *testing.T) {
		registry := NewRuleRegistry()
		var wg sync.WaitGroup

		for i := 0; i < 100; i++ {
			wg.Add(1)
			go func(n int) {
				defer wg.Done()
				rule := &mockRule{id: string(rune(n)), name: "Rule"}
				registry.Register(rule)
			}(i)
		}

		wg.Wait()
		assert.Equal(t, 100, registry.Count())
	})

	t.Run("concurrent enable/disable", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}
		registry.Register(rule)

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(2)
			go func() {
				defer wg.Done()
				registry.Enable("rule1")
			}()
			go func() {
				defer wg.Done()
				registry.Disable("rule1")
			}()
		}

		wg.Wait()
		// Should not panic or deadlock
	})

	t.Run("concurrent read operations", func(t *testing.T) {
		registry := NewRuleRegistry()
		rule := &mockRule{id: "rule1", name: "Test Rule"}
		registry.Register(rule)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(3)
			go func() {
				defer wg.Done()
				registry.Get("rule1")
			}()
			go func() {
				defer wg.Done()
				registry.List()
			}()
			go func() {
				defer wg.Done()
				registry.IsEnabled("rule1")
			}()
		}

		wg.Wait()
		// Should not panic or deadlock
	})
}
