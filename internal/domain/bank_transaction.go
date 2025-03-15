package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// BankTransaction represents a transaction from a bank statement
type BankTransaction struct {
	UniqID string
	Amount decimal.Decimal
	Date   time.Time
	BankID string // Can use this identifier to track which bank a trasaction belongs to
}
