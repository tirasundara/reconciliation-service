package domain

import "time"

// SystemTransactionRepository defines the interface for accessing system transactions
type SystemTransactionRepository interface {
	GetTransactionsInRange(startDate, endDate time.Time) ([]SystemTransaction, error)
}

// BankTransactionRepository defines the interface for accessing bank transactions
type BankTransactionRepository interface {
	GetTransactionsInRange(startDate, endDate time.Time) ([]BankTransaction, error)
	GetBankIdentifier() string
}
