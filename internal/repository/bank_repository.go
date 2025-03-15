package repository

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/pkg/fileutil"
)

var bankHeaderFields = []string{"unique_identifier", "amount", "date"}

// CSVBankRepository implements the BankTransactionRepository interface for CSV files
type CSVBankRepository struct {
	FilePath       string
	BankIdentifier string
	DateFormat     string
}

// NewCSVBankRepository creates a new CSVBankRepository
func NewCSVBankRepository(filePath string, dateFormat string) *CSVBankRepository {
	if dateFormat == "" {
		dateFormat = "2006-01-02" // Default format
	}

	// Try to get bankID from its filename for now
	bankID := filepath.Base(filePath)
	bankID = bankID[:len(bankID)-4] // Remove .csv extension

	return &CSVBankRepository{
		FilePath:       filePath,
		BankIdentifier: bankID,
		DateFormat:     dateFormat,
	}
}

func (r *CSVBankRepository) GetBankIdentifier() string {
	return r.BankIdentifier
}

func (r *CSVBankRepository) GetTransactionsInRange(startDate, endDate time.Time) ([]domain.BankTransaction, error) {
	reader := fileutil.NewCSVReader(r.FilePath)

	// Read just the header row
	header, err := reader.ReadHeader()
	if err != nil {
		return nil, fmt.Errorf("reading bank statement header: %w", err)
	}

	columnMap, err := crateHeaderMap(header, bankHeaderFields)
	if err != nil {
		return nil, fmt.Errorf("mapping CSV columns: %w", err)
	}

	var txns []domain.BankTransaction
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

		txDate, err := time.Parse(r.DateFormat, row[columnMap["date"]])
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

		txn := domain.BankTransaction{
			UniqID: row[columnMap["unique_identifier"]],
			Amount: amount,
			Date:   txDate,
			BankID: r.BankIdentifier,
		}

		txns = append(txns, txn)
		return nil
	}

	// Process data row by row
	if err := reader.ReadAndProcessByRow(rowProcessorFn); err != nil {
		return nil, fmt.Errorf("processing bank transactions: %w", err)
	}

	// Apply date filtering after loading all transactions
	var filteredTxns []domain.BankTransaction
	for _, txn := range txns {
		txnDay := txn.Date.Truncate(24 * time.Hour)
		startDay := startDate.Truncate(24 * time.Hour)
		endDay := endDate.Truncate(24 * time.Hour)

		if (txnDay.Equal(startDay) || txnDay.After(startDay)) &&
			(txnDay.Equal(endDay) || txnDay.Before(endDay)) {
			filteredTxns = append(filteredTxns, txn)
		}
	}

	return filteredTxns, nil
}
