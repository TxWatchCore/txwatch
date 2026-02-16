package models

import "time"

type Risk struct {
	Score             float64
	TransactionID     string
	AccountID         string
	Timestamp         time.Time
	ContributingRules []RuleContribution
	AIScore           *float64
}

type RuleContribution struct {
	RuleID       string
	RuleName     string
	Flagged      bool
	Reason       string
	ScoreContrib float64
}

func (r *Risk) Validate() error {
	if r.Score < 0.0 || r.Score > 1.0 {
		return ErrInvalidScore
	}
	if r.TransactionID == "" {
		return ErrInvalidTransaction
	}
	if r.AccountID == "" {
		return ErrInvalidTransaction
	}
	if r.AIScore != nil && (*r.AIScore < 0.0 || *r.AIScore > 1.0) {
		return ErrInvalidScore
	}
	return nil
}

func (r *Risk) HasAIScore() bool {
	return r.AIScore != nil
}

// FlaggedRules returns the list of rules that flagged this transaction.
func (r *Risk) FlaggedRules() []RuleContribution {
	var flagged []RuleContribution
	for _, rule := range r.ContributingRules {
		if rule.Flagged {
			flagged = append(flagged, rule)
		}
	}
	return flagged
}
