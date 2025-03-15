package domain_test

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
)

func TestSystemTransaction(t *testing.T) {
	amount := decimal.NewFromFloat(100.50)
	txTime, _ := time.Parse(time.RFC3339, "2025-01-01T01:01:01Z")

	tx := domain.SystemTransaction{
		TrxID:           "SYS-TXN-123456",
		Amount:          amount,
		Type:            domain.Credit,
		TransactionTime: txTime,
	}

	if tx.TrxID != "SYS-TXN-123456" {
		t.Errorf("Expected TrxID to be 'SYS-TXN-123456', got '%s'", tx.TrxID)
	}

	if !tx.Amount.Equal(amount) {
		t.Errorf("Expected Amount to be %s, got %s", amount, tx.Amount)
	}

	if tx.Type != domain.Credit {
		t.Errorf("Expected Type to be Credit, got %s", tx.Type)
	}

	if !tx.TransactionTime.Equal(txTime) {
		t.Errorf("Expected TransactionTime to be %v, got %v", txTime, tx.TransactionTime)
	}
}
