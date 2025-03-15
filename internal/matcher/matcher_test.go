package matcher_test

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/internal/matcher"
)

func TestDefaultMatcher_FindMatches(t *testing.T) {
	// Create a matcher with default strategies
	m := matcher.NewDefaultMatcher()

	// Test data - 3 system transactions
	systemTxns := []domain.SystemTransaction{
		{
			TrxID:           "SYS-TXN-12345",
			Amount:          decimal.NewFromFloat(100000.50),
			Type:            domain.Credit,
			TransactionTime: parseTime(t, "2025-01-15T14:30:00"),
		},
		{
			TrxID:           "SYS-TXN-12346",
			Amount:          decimal.NewFromFloat(50000.25),
			Type:            domain.Debit,
			TransactionTime: parseTime(t, "2025-01-16T09:15:00"),
		},
		{
			TrxID:           "SYS-TXN-12347",
			Amount:          decimal.NewFromFloat(75000.00),
			Type:            domain.Credit,
			TransactionTime: parseTime(t, "2025-01-17T11:45:00"),
		},
	}

	// Test data - 3 matching bank transactions + 1 non-matching
	bankTxns := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98765",
			Amount: decimal.NewFromFloat(100000.50),
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98766",
			Amount: decimal.NewFromFloat(-50000.25),
			Date:   parseTime(t, "2025-01-16"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98767",
			Amount: decimal.NewFromFloat(75000.00),
			Date:   parseTime(t, "2025-01-17"),
			BankID: "Bank-ABC",
		},
		{
			UniqID: "BANK-STMT-98768",
			Amount: decimal.NewFromFloat(200.00),
			Date:   parseTime(t, "2025-01-18"),
			BankID: "Bank-ABC",
		},
	}

	// Find matches
	matches, err := m.FindMatches(systemTxns, bankTxns)

	// Check for errors
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check number of matches
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches, got %d", len(matches))
	}

	// Check specific matches
	for _, match := range matches {
		if match.SystemTxn.TrxID == "SYS-TXN-12345" {
			if match.BankTxn.UniqID != "BANK-STMT-98765" {
				t.Errorf("Expected SYS-TXN-12345 to match with BANK-STMT-98765, got %s",
					match.BankTxn.UniqID)
			}
		} else if match.SystemTxn.TrxID == "SYS-TXN-12346" {
			if match.BankTxn.UniqID != "BANK-STMT-98766" {
				t.Errorf("Expected SYS-TXN-12346 to match with BANK-STMT-98766, got %s",
					match.BankTxn.UniqID)
			}
		} else if match.SystemTxn.TrxID == "SYS-TXN-12347" {
			if match.BankTxn.UniqID != "BANK-STMT-98767" {
				t.Errorf("Expected SYS-TXN-12347 to match with BANK-STMT-98767, got %s",
					match.BankTxn.UniqID)
			}
		}
	}

	// Test avoiding duplicate matches
	systemTxns = append(systemTxns, domain.SystemTransaction{
		TrxID:           "SYS-TXN-12348",
		Amount:          decimal.NewFromFloat(100000.50), // Same amount as SYS-TXN-12345
		Type:            domain.Credit,
		TransactionTime: parseTime(t, "2025-01-15T16:30:00"), // Same day
	})

	matches, err = m.FindMatches(systemTxns, bankTxns)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should still only be 3 matches, as there are only 3 matching bank transactions
	if len(matches) != 3 {
		t.Errorf("Expected 3 matches with duplicate system transaction, got %d", len(matches))
	}

	// Check that BANK-STMT-98765 was only matched once
	matchCount := 0
	for _, match := range matches {
		if match.BankTxn.UniqID == "BANK-STMT-98765" {
			matchCount++
		}
	}

	if matchCount != 1 {
		t.Errorf("Expected bank transaction BANK-STMT-98765 to be matched exactly once, but it was matched %d times", matchCount)
	}
}

// Test with discrepancies in amount
func TestDefaultMatcher_WithDiscrepancies(t *testing.T) {
	// Create a matcher with fuzzy strategy included
	m := matcher.NewDefaultMatcher(
		matcher.NewExactMatchStrategy(),
		matcher.NewFuzzyMatchStrategy(0.10), // 0.10 threshold
	)

	// Test data with slight amount discrepancy
	systemTxns := []domain.SystemTransaction{
		{
			TrxID:           "SYS-TXN-12345",
			Amount:          decimal.NewFromFloat(100000.50),
			Type:            domain.Credit,
			TransactionTime: parseTime(t, "2025-01-15T14:30:00"),
		},
	}

	bankTxns := []domain.BankTransaction{
		{
			UniqID: "BANK-STMT-98765",
			Amount: decimal.NewFromFloat(100000.45),
			Date:   parseTime(t, "2025-01-15"),
			BankID: "Bank-ABC",
		},
	}

	// Find matches
	matches, err := m.FindMatches(systemTxns, bankTxns)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Check that a match was found despite discrepancy
	if len(matches) != 1 {
		t.Fatalf("Expected 1 match with discrepancy, got %d", len(matches))
	}

	// Check that the discrepancy amount is correctly calculated
	expectedDiff := decimal.NewFromFloat(0.05)
	if !matches[0].AmmountDiff.Equal(expectedDiff) {
		t.Errorf("Expected amount difference to be %s, got %s",
			expectedDiff, matches[0].AmmountDiff)
	}
}
