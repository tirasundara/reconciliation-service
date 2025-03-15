package matcher_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/internal/matcher"
)

func TestExactMatchStrategy(t *testing.T) {
	strategy := matcher.NewExactMatchStrategy()

	// Setup test data
	sysTxn := domain.SystemTransaction{
		TrxID:           "SYS-TXN-12345",
		Amount:          decimal.NewFromFloat(100000.00),
		Type:            domain.Credit,
		TransactionTime: parseTime(t, "2025-01-15T14:30:00"),
	}

	bankTxns := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98765",
			Amount: decimal.NewFromFloat(100000.00), // Should match
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98766",
			Amount: decimal.NewFromFloat(100000.01), // Not match - amount
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98767",
			Amount: decimal.NewFromFloat(100000.00), // Not match - date
			Date:   parseTime(t, "2025-01-16"),
			BankID: "Bank-ABC",
		},
	}

	// Test match found
	matchedTxn, found := strategy.Match(sysTxn, bankTxns)

	if !found {
		t.Errorf("Expected to find a match, but none was found")
	}

	if matchedTxn.UniqID != "BANK-STMT-98765" {
		t.Errorf("Expected to match transaction BANK-STMT-98765, but matched %s", matchedTxn.UniqID)
	}

	// Test with no match
	sysTxn.Amount = decimal.NewFromFloat(200000.00)
	matchedTxn, found = strategy.Match(sysTxn, bankTxns)

	if found {
		t.Errorf("Expected to find no match, but found transaction %s", matchedTxn.UniqID)
	}

	// Test debit transaction
	sysTxn = domain.SystemTransaction{
		TrxID:           "SYS-TXN-12346",
		Amount:          decimal.NewFromFloat(50.25),
		Type:            domain.Debit,
		TransactionTime: parseTime(t, "2025-01-16T09:15:00"),
	}

	bankTxns = []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98766",
			Amount: decimal.NewFromFloat(-50.25), // Match (negative for debit)
			Date:   parseTime(t, "2025-01-16"),
			BankID: "Bank-ABC",
		},
	}

	matchedTxn, found = strategy.Match(sysTxn, bankTxns)

	if !found {
		t.Errorf("Expected to find a match for debit transaction, but none was found")
	}
}

func TestFuzzyMatchStrategy(t *testing.T) {
	strategy := matcher.NewFuzzyMatchStrategy(0.02) // 0.02 threshold

	// Setup test data
	sysTxn := domain.SystemTransaction{
		TrxID:           "SYS-TXN-12345",
		Amount:          decimal.NewFromFloat(100000.00),
		Type:            domain.Credit,
		TransactionTime: parseTime(t, "2025-01-15T14:30:00"),
	}

	bankTxns := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98765",
			Amount: decimal.NewFromFloat(100000.01), // Within threshold (0.01 diff). Should match
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98766",
			Amount: decimal.NewFromFloat(100000.03), // Outside threshold (0.03 diff). Not a match
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
	}

	// Test match within threshold
	matchedTxn, found := strategy.Match(sysTxn, bankTxns)

	if !found {
		t.Errorf("Expected to find a match within threshold, but none was found")
	}

	if matchedTxn.UniqID != "BANK-STMT-98765" {
		t.Errorf("Expected to match transaction BANK-STMT-98765, but matched %s", matchedTxn.UniqID)
	}

	// Test no match outside threshold
	strategy = matcher.NewFuzzyMatchStrategy(0.005) // 0.005 threshold
	matchedTxn, found = strategy.Match(sysTxn, bankTxns)

	if found {
		t.Errorf("Expected to find no match with tight threshold, but found transaction %s", matchedTxn.UniqID)
	}
}

func TestDateBufferMatchStrategy(t *testing.T) {
	// Create strategy with 1-day buffer
	strategy := matcher.NewDateBufferMatchStrategy(1)

	// Setup test data for system transaction
	sysTxn := domain.SystemTransaction{
		TrxID:           "SYS-TXN-12345",
		Amount:          decimal.NewFromFloat(100000.50),
		Type:            domain.Credit,
		TransactionTime: parseTime(t, "2025-01-15T23:55:00"), // Late in the day
	}

	// Test cases with various bank transactions
	bankTxns := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98765",
			Amount: decimal.NewFromFloat(100000.50), // Same amount, same day
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98766",
			Amount: decimal.NewFromFloat(100000.50), // Same amount, next day (within buffer)
			Date:   parseTime(t, "2025-01-16"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98767",
			Amount: decimal.NewFromFloat(100000.50), // Same amount, prev day (within buffer)
			Date:   parseTime(t, "2025-01-14"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98768",
			Amount: decimal.NewFromFloat(100000.50), // Same amount, but outside buffer
			Date:   parseTime(t, "2025-01-13"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98769",
			Amount: decimal.NewFromFloat(100000.60), // Different amount, within buffer
			Date:   parseTime(t, "2025-01-16"),
			BankID: "Bank-ABC",
		},
	}

	// Test matching transaction on same day
	matchedTxn, found := strategy.Match(sysTxn, bankTxns)
	if !found {
		t.Errorf("Expected to find a match on the same day, but none was found")
	}
	if found && matchedTxn.UniqID != "BANK-STMT-98765" {
		t.Errorf("Expected to match BANK-STMT-98765 on same day, got %s", matchedTxn.UniqID)
	}

	// Remove the same-day match to test next-day matching
	bankTxnsWithoutSameDay := []domain.BankTransaction{
		bankTxns[1], bankTxns[2], bankTxns[3], bankTxns[4],
	}

	// Test matching transaction on next day (within buffer)
	matchedTxn, found = strategy.Match(sysTxn, bankTxnsWithoutSameDay)
	if !found {
		t.Errorf("Expected to find a match on the next day (within buffer), but none was found")
	}
	if found && matchedTxn.UniqID != "BANK-STMT-98766" {
		t.Errorf("Expected to match BANK-STMT-98766 on next day, got %s", matchedTxn.UniqID)
	}

	// Remove both same-day and next-day matches to test previous-day matching
	bankTxnsWithoutNearDays := []domain.BankTransaction{
		bankTxns[2], bankTxns[3], bankTxns[4],
	}

	// Test matching transaction on previous day (within buffer)
	matchedTxn, found = strategy.Match(sysTxn, bankTxnsWithoutNearDays)
	if !found {
		t.Errorf("Expected to find a match on the previous day (within buffer), but none was found")
	}
	if found && matchedTxn.UniqID != "BANK-STMT-98767" {
		t.Errorf("Expected to match BANK-STMT-98767 on previous day, got %s", matchedTxn.UniqID)
	}

	// Test with transactions outside buffer and with different amounts
	bankTxnsOutsideBuffer := []domain.BankTransaction{
		bankTxns[3], bankTxns[4],
	}

	// Should not find a match
	matchedTxn, found = strategy.Match(sysTxn, bankTxnsOutsideBuffer)
	if found {
		t.Errorf("Expected not to find a match outside buffer or with different amount, but found %s",
			matchedTxn.UniqID)
	}

	// Test with DEBIT transaction type
	sysTxnDebit := domain.SystemTransaction{
		TrxID:           "SYS-TXN-12346",
		Amount:          decimal.NewFromFloat(50000.25),
		Type:            domain.Debit,
		TransactionTime: parseTime(t, "2025-01-15T08:15:00"),
	}

	bankTxnsDebit := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98770",
			Amount: decimal.NewFromFloat(-50000.25), // Negative for DEBIT
			Date:   parseTime(t, "2025-01-16"),      // Next day (within buffer)
			BankID: "Bank-ABC",
		},
	}

	// Test matching DEBIT transaction across days
	matchedTxn, found = strategy.Match(sysTxnDebit, bankTxnsDebit)
	if !found {
		t.Errorf("Expected to find a DEBIT match across days, but none was found")
	}
}

// Helper function to parse time strings
func parseTime(t *testing.T, timeStr string) time.Time {
	var layout string
	if len(timeStr) > 10 {
		layout = "2006-01-02T15:04:05"
	} else {
		layout = "2006-01-02"
	}

	result, err := time.Parse(layout, timeStr)
	if err != nil {
		t.Fatalf("Failed to parse time string '%s': %v", timeStr, err)
	}

	return result
}
