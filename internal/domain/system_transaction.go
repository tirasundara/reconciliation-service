package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// TransactionType represents the type of transaction
type TransactionType string

// Transaction types
const (
	Debit  TransactionType = "DEBIT"
	Credit TransactionType = "CREDIT"
)

// SystemTransaction represents a transaction from the internal system
type SystemTransaction struct {
	TrxID           string
	Amount          decimal.Decimal
	Type            TransactionType
	TransactionTime time.Time
}
