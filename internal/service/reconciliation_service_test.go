package service_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/internal/matcher"
	"github.com/tirasundara/reconciliation-service/internal/service"
)

type MockSystemRepository struct {
	transactions []domain.SystemTransaction
}

func (m *MockSystemRepository) GetTransactionsInRange(startDate, endDate time.Time) ([]domain.SystemTransaction, error) {
	return m.transactions, nil
}

type MockBankRepository struct {
	transactions []domain.BankTransaction
	BankID       string
}

func (m *MockBankRepository) GetTransactionsInRange(startDate, endDate time.Time) ([]domain.BankTransaction, error) {
	return m.transactions, nil
}

func (m *MockBankRepository) GetBankIdentifier() string {
	return m.BankID
}

func TestReconciliationService(t *testing.T) {
	// Create test data
	sysRepo := &MockSystemRepository{
		transactions: []domain.SystemTransaction{
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
			{
				TrxID:           "SYS-TXN-12348",
				Amount:          decimal.NewFromFloat(200000.00),
				Type:            domain.Debit,
				TransactionTime: parseTime(t, "2025-01-18T16:20:00"),
			},
			{
				TrxID:           "SYS-TXN-12349",
				Amount:          decimal.NewFromFloat(25000.00),
				Type:            domain.Credit,
				TransactionTime: parseTime(t, "2025-01-19T10:30:00"),
			},
		},
	}

	bankRepoA := &MockBankRepository{
		transactions: []domain.BankTransaction{
			{
				UniqID: "BANK-STMT-98765",
				Amount: decimal.NewFromFloat(100000.45), // Discrepancy of 0.05
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
				UniqID: "BANK-STMT-98769",
				Amount: decimal.NewFromFloat(300000.00), // Unmatched
				Date:   parseTime(t, "2025-01-19"),
				BankID: "Bank-ABC",
			},
		},
		BankID: "Bank-ABC",
	}

	bankRepoB := &MockBankRepository{
		transactions: []domain.BankTransaction{
			{
				UniqID: "BANK-STMT-88765",
				Amount: decimal.NewFromFloat(-200000.00),
				Date:   parseTime(t, "2025-01-18"),
				BankID: "Bank-BCD",
			},
			{
				UniqID: "BANK-STMT-88766",
				Amount: decimal.NewFromFloat(150000.00), // Unmatched
				Date:   parseTime(t, "2025-01-20"),
				BankID: "Bank-BCD",
			},
		},
		BankID: "Bank-BCD",
	}

	bankRepos := map[string]domain.BankTransactionRepository{
		"Bank-ABC": bankRepoA,
		"Bank-BCD": bankRepoB,
	}

	// Create matcher with strategies
	m := matcher.NewDefaultMatcher(
		matcher.NewExactMatchStrategy(),
		matcher.NewFuzzyMatchStrategy(0.10), // 0.10 threshold
	)

	// Create reconciliation service
	service := service.NewReconciliationService(sysRepo, bankRepos, m, 1)

	// Perform reconciliation
	startDate := parseTime(t, "2025-01-15")
	endDate := parseTime(t, "2025-01-20")
	result, err := service.Reconcile(startDate, endDate)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Test number of matched transactions
	expectedMatches := 4 // SYS-TXN-12345, SYS-TXN-12346, SYS-TXN-12347, SYS-TXN-12348
	if len(result.MatchedTxns) != expectedMatches {
		t.Errorf("Expected %d matches, got %d", expectedMatches, len(result.MatchedTxns))
	}

	// Test unmatched system transactions
	expectedUnmatchedSys := 1 // SYS-TXN-12349
	if len(result.UnMatchedSystemTxns) != expectedUnmatchedSys {
		t.Errorf("Expected %d unmatched system transactions, got %d",
			expectedUnmatchedSys, len(result.UnMatchedSystemTxns))
	}

	// Test unmatched bank transactions
	expectedUnmatchedBankABC := 1 // BANK-STMT-98769
	expectedUnmatchedBankBCD := 1 // BANK-STMT-88766

	if len(result.UnMatchedBankTxns["Bank-ABC"]) != expectedUnmatchedBankABC {
		t.Errorf("Expected %d unmatched Bank-ABC transactions, got %d",
			expectedUnmatchedBankABC, len(result.UnMatchedBankTxns["Bank-ABC"]))
	}

	if len(result.UnMatchedBankTxns["Bank-BCD"]) != expectedUnmatchedBankBCD {
		t.Errorf("Expected %d unmatched Bank-BCD transactions, got %d",
			expectedUnmatchedBankBCD, len(result.UnMatchedBankTxns["Bank-BCD"]))
	}

	// Test total discrepancies
	expectedDiscrepancy := decimal.NewFromFloat(0.05) // From SYS-TXN-12345 matching with BANK-STMT-98765
	if !result.TotalDiscrepancies.Equal(expectedDiscrepancy) {
		t.Errorf("Expected total discrepancies to be %s, got %s",
			expectedDiscrepancy, result.TotalDiscrepancies)
	}

	// Test total transactions processed
	expectedTotal := 7 // 4 matched + 1 unmatched system + 2 unmatched bank
	if result.TotalTxnsProcessed != expectedTotal {
		t.Errorf("Expected %d total transactions processed, got %d",
			expectedTotal, result.TotalTxnsProcessed)
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
