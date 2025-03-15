package repository

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/pkg/fileutil"
)

var systemHeaderFields = []string{"trxID", "amount", "type", "transactionTime"}

// CSVSystemRepository implements the SystemTransactionRepository interface for CSV file(s)
type CSVSystemRepository struct {
	FilePath   string
	DateFormat string
}

// NewCSVSystemRepository creates a new CSVSystemRepository
func NewCSVSystemRepository(fp, dateFormat string) *CSVSystemRepository {
	if dateFormat == "" {
		dateFormat = "2006-01-02T15:04:05"
	}

	return &CSVSystemRepository{
		FilePath:   fp,
		DateFormat: dateFormat,
	}
}

func (r *CSVSystemRepository) GetTransactionsInRange(startDate, endDate time.Time) ([]domain.SystemTransaction, error) {
	reader := fileutil.NewCSVReader(r.FilePath)

	header, err := reader.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("reading system transaction header: %w", err)
	}

	columnMap, err := crateHeaderMap(header, systemHeaderFields)
	if err != nil {
		return nil, fmt.Errorf("mapping CSV column: %w", err)
	}

	var txns []domain.SystemTransaction
	var rowProcessorFn = func(row []string) error {
		// Find the highest column index needed
		maxIndex := -1
		for _, idx := range columnMap {
			if idx > maxIndex {
				maxIndex = idx
			}
		}

		// Skip if row doesn't have enough fields
		if len(row) <= maxIndex {
			return nil
		}

		txTime, err := time.Parse(r.DateFormat, row[columnMap["transactionTime"]])
		if err != nil {
			// Log but continue processing other rows
			fmt.Printf("Warning: Invalid date format: %v\n", err)
			return nil
		}

		amount, err := decimal.NewFromString(row[columnMap["amount"]])
		if err != nil {
			fmt.Printf("Warning: Invalid amount format: %v\n", err)
			return nil
		}

		txnType := domain.TransactionType(row[columnMap["type"]])
		if txnType != domain.Debit && txnType != domain.Credit {
			fmt.Printf("Warning: Invalid transaction type: %s\n", txnType)
			return nil
		}

		txn := domain.SystemTransaction{
			TrxID:           row[columnMap["trxID"]],
			Amount:          amount,
			Type:            txnType,
			TransactionTime: txTime,
		}

		txns = append(txns, txn)
		return nil
	}

	// Read and process row by row
	if err := reader.ReadAndProcessByRow(rowProcessorFn); err != nil {
		return nil, fmt.Errorf("reading and processing system transaction: %w", err)
	}

	// Apply filtering bt date range after loading all transactions
	var filteredTxns []domain.SystemTransaction
	for _, txn := range txns {
		txnDay := txn.TransactionTime.Truncate(24 * time.Hour)
		startDay := startDate.Truncate(24 * time.Hour)
		endDay := endDate.Truncate(24 * time.Hour)

		if (txnDay.Equal(startDay) || txnDay.After(startDay)) &&
			(txnDay.Equal(endDay) || txnDay.Before(endDay)) {
			filteredTxns = append(filteredTxns, txn)
		}
	}

	return filteredTxns, nil
}
