package domain

import "github.com/shopspring/decimal"

// ReconciliationResult containts the result of a reconciliation process
type ReconciliationResult struct {
	TotalTxnsProcessed  int
	MatchedTxns         []Match
	UnMatchedSystemTxns []SystemTransaction
	UnMatchedBankTxns   map[string][]BankTransaction // Grouped by bank
	TotalDiscrepancies  decimal.Decimal
}
