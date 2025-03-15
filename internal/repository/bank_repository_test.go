package repository_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/repository"
)

func TestCSVBankRepository_GetTransactionsInRange(t *testing.T) {
	repo := repository.NewCSVBankRepository("../../test/testdata/bank_statements.csv", "")

	startDate, _ := time.Parse("2006-01-02", "2025-01-16")
	endDate, _ := time.Parse("2006-01-02", "2025-01-18")

	// Should return transactions from Jan 16-18 (3 transactions)
	transactions, err := repo.GetTransactionsInRange(startDate, endDate)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(transactions) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(transactions))
	}

	// Verify first transaction in range
	expectedAmount := decimal.NewFromFloat(-50000.00)
	if !transactions[0].Amount.Equal(expectedAmount) {
		t.Errorf("Expected first transaction amount to be %s, got %s",
			expectedAmount, transactions[0].Amount)
	}

	// Verify bank identifier
	bankID := repo.GetBankIdentifier()
	expectedBankID := "bank_statements"

	if bankID != expectedBankID {
		t.Errorf("Expected bank identifier to be %s, got %s",
			expectedBankID, bankID)
	}

	// Test with no transactions in range
	startDate, _ = time.Parse("2006-01-02", "2023-02-01")
	endDate, _ = time.Parse("2006-01-02", "2023-02-28")

	transactions, err = repo.GetTransactionsInRange(startDate, endDate)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(transactions) != 0 {
		t.Errorf("Expected 0 transactions for out-of-range dates, got %d", len(transactions))
	}
}
