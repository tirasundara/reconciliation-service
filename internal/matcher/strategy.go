package matcher

import (
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
)

// MatchingStrategy defines a strategy for matching system transactions to bank transactions
type MatchingStrategy interface {
	Match(sysTxn domain.SystemTransaction, bankTxns []domain.BankTransaction) (domain.BankTransaction, bool)
}

// ExactMatchStrategy matches transactions based on exact date and amount
type ExactMatchStrategy struct{}

// NewExactMatchStrategy creates a new ExactMatchStrategy
func NewExactMatchStrategy() *ExactMatchStrategy {
	return &ExactMatchStrategy{}
}

// Match implements the MatchingStrategy interface
func (s *ExactMatchStrategy) Match(sysTxn domain.SystemTransaction, bankTxns []domain.BankTransaction) (domain.BankTransaction, bool) {
	sysTxnDate := sysTxn.TransactionTime.Truncate(24 * time.Hour)
	sysAmount := getNormalizedAmount(sysTxn)

	for _, bankTxn := range bankTxns {
		bankTxnDate := bankTxn.Date.Truncate(24 * time.Hour)

		// Check if the dates match (same day)
		if !bankTxnDate.Equal(sysTxnDate) {
			continue
		}

		// Check if the amounts match exactly
		if !bankTxn.Amount.Equal(sysAmount) {
			continue
		}

		return bankTxn, true
	}

	return domain.BankTransaction{}, false
}

// FuzzyMatchStrategy matches transactions based on date and amount within a threshold
type FuzzyMatchStrategy struct {
	AmountThreshold decimal.Decimal
}

// NewFuzzyMatchStrategy creates a new FuzzyMatchStrategy with the given threshold
func NewFuzzyMatchStrategy(threshold float64) *FuzzyMatchStrategy {
	return &FuzzyMatchStrategy{
		AmountThreshold: decimal.NewFromFloat(threshold),
	}
}

// Match implements the MatchingStrategy interface
func (s *FuzzyMatchStrategy) Match(sysTxn domain.SystemTransaction, bankTxns []domain.BankTransaction) (domain.BankTransaction, bool) {
	sysTxnDate := sysTxn.TransactionTime.Truncate(24 * time.Hour)
	sysAmount := getNormalizedAmount(sysTxn)

	for _, bankTxn := range bankTxns {
		bankTxnDate := bankTxn.Date.Truncate(24 * time.Hour)

		// Check if the dates match (same day)
		if !bankTxnDate.Equal(sysTxnDate) {
			continue
		}

		// Check if the amounts are within the threshold
		diff := sysAmount.Sub(bankTxn.Amount).Abs()
		if diff.GreaterThan(s.AmountThreshold) {
			continue
		}

		return bankTxn, true
	}

	return domain.BankTransaction{}, false
}

// DateBufferMatchStrategy matches transactions with a date buffer
type DateBufferMatchStrategy struct {
	BufferDays int
}

// NewDateBufferMatchStrategy creates a new DateBufferMatchStrategy with the given buffer
func NewDateBufferMatchStrategy(bufferDays int) *DateBufferMatchStrategy {
	return &DateBufferMatchStrategy{
		BufferDays: bufferDays,
	}
}

// Match implements the MatchingStrategy interface
func (s *DateBufferMatchStrategy) Match(sysTxn domain.SystemTransaction, bankTxns []domain.BankTransaction) (domain.BankTransaction, bool) {
	sysAmount := getNormalizedAmount(sysTxn)

	// Calculate date range with buffer
	minDate := sysTxn.TransactionTime.AddDate(0, 0, -s.BufferDays).Truncate(24 * time.Hour)
	maxDate := sysTxn.TransactionTime.AddDate(0, 0, s.BufferDays).Truncate(24 * time.Hour)

	for _, bankTxn := range bankTxns {
		bankTxnDate := bankTxn.Date.Truncate(24 * time.Hour)

		// Check if the date is within buffer range
		if bankTxnDate.Before(minDate) || bankTxnDate.After(maxDate) {
			continue
		}

		// Check if the amounts match exactly
		if !bankTxn.Amount.Equal(sysAmount) {
			continue
		}

		return bankTxn, true
	}

	return domain.BankTransaction{}, false
}

// getNormalizedAmount returns the amount with the sign adjusted for debit/credit
func getNormalizedAmount(sysTxn domain.SystemTransaction) decimal.Decimal {
	if sysTxn.Type == domain.Debit {
		return sysTxn.Amount.Neg() // Return negative amount for debits
	}
	return sysTxn.Amount
}
