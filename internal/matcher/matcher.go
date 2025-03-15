package matcher

import (
	"fmt"

	"github.com/tirasundara/reconciliation-service/internal/domain"
)

const (
	defaultAmountThreshold = 0.01 // 0.01 threshold
	defaultDaysBuffer      = 1    // 1 day buffer
)

// DefaultMatcher implements the TransactionMatcher interface
type DefaultMatcher struct {
	strategies []MatchingStrategy
}

// NewDefaultMatcher creates a new DefaultMatcher with the given strategies
func NewDefaultMatcher(strategies ...MatchingStrategy) *DefaultMatcher {
	if len(strategies) == 0 {

		// Default strategies
		strategies = []MatchingStrategy{
			NewExactMatchStrategy(),
			NewFuzzyMatchStrategy(defaultAmountThreshold),
			NewDateBufferMatchStrategy(defaultDaysBuffer),
		}
	}

	return &DefaultMatcher{
		strategies: strategies,
	}
}

func (m *DefaultMatcher) FindMatches(systemTxns []domain.SystemTransaction, bankTxns []domain.BankTransaction) ([]domain.Match, error) {
	matches := make([]domain.Match, 0)

	// Print info
	fmt.Printf("Matching %d system transactions with %d bank transactions\n", len(systemTxns), len(bankTxns))

	matchedBankTxns := make(map[string]bool)

	// For each system transaction, try to find a match
	for _, sysTxn := range systemTxns {
		var matched bool
		var matchedBankTxn domain.BankTransaction

		// Try each strategy in order until a match is found
		for _, strategy := range m.strategies {

			// Filter out already matched bank transactions
			availableBankTxns := make([]domain.BankTransaction, 0)
			for _, bankTxn := range bankTxns {

				key := fmt.Sprintf("%s-%s", bankTxn.BankID, bankTxn.UniqID)
				if matchedBankTxns[key] {
					continue
				}

				availableBankTxns = append(availableBankTxns, bankTxn)
			}

			bankTxn, found := strategy.Match(sysTxn, availableBankTxns)
			if found {
				matched = true
				matchedBankTxn = bankTxn
				break
			}
		}

		if matched {
			// Mark bank transaction as matched
			key := fmt.Sprintf("%s-%s", matchedBankTxn.BankID, matchedBankTxn.UniqID)
			matchedBankTxns[key] = true

			sysAmount := getNormalizedAmount(sysTxn)
			amountDiff := sysAmount.Sub(matchedBankTxn.Amount).Abs()

			match := domain.Match{
				SystemTxn:   sysTxn,
				BankTxn:     matchedBankTxn,
				AmmountDiff: amountDiff,
			}

			matches = append(matches, match)
		}
	}

	return matches, nil
}
