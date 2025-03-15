package domain

import "github.com/shopspring/decimal"

// Match represents a match between a system transaction and a bank transaction
type Match struct {
	SystemTxn   SystemTransaction
	BankTxn     BankTransaction
	AmmountDiff decimal.Decimal
}
