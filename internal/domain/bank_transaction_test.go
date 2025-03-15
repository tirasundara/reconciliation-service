package domain_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
)

func TestBankTransaction(t *testing.T) {
	amount := decimal.NewFromFloat(-50.25)
	txDate, _ := time.Parse("2006-01-02", "2023-01-15")

	tx := domain.BankTransaction{
		UniqID: "BANK-TXN-987654",
		Amount: amount,
		Date:   txDate,
		BankID: "Bank-ABC",
	}

	if tx.UniqID != "BANK-TXN-987654" {
		t.Errorf("Expected UniqID to be 'BANK-TXN-987654', got '%s'", tx.UniqID)
	}

	if !tx.Amount.Equal(amount) {
		t.Errorf("Expected Amount to be %s, got %s", amount, tx.Amount)
	}

	if !tx.Date.Equal(txDate) {
		t.Errorf("Expected Date to be %v, got %v", txDate, tx.Date)
	}

	if tx.BankID != "Bank-ABC" {
		t.Errorf("Expected BankID to be 'Bank-ABC', got '%s'", tx.BankID)
	}
}
