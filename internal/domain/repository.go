package domain

import "time"

// SystemTransactionRepository defines the interface for accessing system transactions
type SystemTransactionRepository interface {
	// GetTransactionsInRange gets system transactions for specified date between startDate and endDate
	GetTransactionsInRange(startDate, endDate time.Time) ([]SystemTransaction, error)

	// GetTransactionsInRangeConcurrently is a concurrent version of GetTransactionsInRange()
	GetTransactionsInRangeConcurrently(startDate, endDate time.Time) ([]SystemTransaction, error)
}

// BankTransactionRepository defines the interface for accessing bank transactions
type BankTransactionRepository interface {
	// GetTransactionsInRange gets bank transactions for specified date between startDate and endDate
	GetTransactionsInRange(startDate, endDate time.Time) ([]BankTransaction, error)

	// GetTransactionsInRangeConcurrently is a concurrent version of GetTransactionsInRange()
	GetTransactionsInRangeConcurrently(startDate, endDate time.Time) ([]BankTransaction, error)

	// GetBankIdentifier returns bank identifier
	GetBankIdentifier() string
}
