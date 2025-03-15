package repository

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
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
	NumWorkers     int
	BatchSize      int
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
		NumWorkers:     4,
		BatchSize:      1000,
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

// GetTransactionsInRangeConcurrently reads and parse CSV rows concurrently, good for handling CSV with huge rows
func (r *CSVBankRepository) GetTransactionsInRangeConcurrently(startDate, endDate time.Time) ([]domain.BankTransaction, error) {
	f, err := os.Open(r.FilePath)
	if err != nil {
		return nil, fmt.Errorf("opening csv file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading system transaction header: %w", err)
	}

	columnMap, err := crateHeaderMap(header, bankHeaderFields)
	if err != nil {
		return nil, fmt.Errorf("mapping CSV column: %w", err)
	}

	// Set up concurrent processing
	jobs := make(chan [][]string, r.NumWorkers)
	results := make(chan []domain.BankTransaction, r.NumWorkers)
	errChan := make(chan error, r.NumWorkers)

	// Start the worker pool
	var wg sync.WaitGroup
	startBankWorkers(r.NumWorkers, &wg, jobs, results, columnMap, r.DateFormat, r.BankIdentifier, startDate, endDate)

	// Start a goroutine to close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Read and distribute batches of CSV records to workers
	go func() {
		defer close(jobs) // Close jobs channel when done reading

		err := readAndDistributeBankStatements(reader, jobs, r.BatchSize)
		if err != nil {
			errChan <- err
		}
	}()

	// Collect results from workers
	txns, err := collectBankTxnResults(results, errChan)
	if err != nil {
		return nil, err
	}

	return txns, nil
}

// readAndDistributeBankStatements reads statement row from CSV then distribute them to Go workers
func readAndDistributeBankStatements(csvReader *csv.Reader, jobs chan<- [][]string, batchSize int) error {
	batch := make([][]string, 0, batchSize)

	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading CSV record: %w", err)
		}

		batch = append(batch, record)

		// When batch is full, send it to a worker
		if len(batch) >= batchSize {
			jobs <- batch
			batch = make([][]string, 0, batchSize)
		}
	}

	// Send any remaining records in the last batch
	if len(batch) > 0 {
		jobs <- batch
	}

	return nil
}

// startBankWorkers creates a pool of worker goroutines to process batches of CSV rows
func startBankWorkers(numWorkers int, wg *sync.WaitGroup,
	jobs <-chan [][]string, results chan<- []domain.BankTransaction,
	columnMap map[string]int, dateFormat, bankID string, startDate, endDate time.Time) {

	// Find the highest column index needed
	maxIndex := -1
	for _, idx := range columnMap {
		if idx > maxIndex {
			maxIndex = idx
		}
	}

	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for batch := range jobs {
				batchResults := make([]domain.BankTransaction, 0, len(batch))

				for _, row := range batch {

					// Skip if row doesn't have enough fields
					if len(row) <= maxIndex {
						fmt.Printf("Warning: Invalid row: %v\n", row)
						continue // Resilient. We try to process as much row as possible
					}

					txDate, err := time.Parse(dateFormat, row[columnMap["date"]])
					if err != nil {
						// Log but continue processing other rows
						fmt.Printf("Warning: Invalid date format: %v\n", err)
						continue
					}

					amount, err := decimal.NewFromString(row[columnMap["amount"]])
					if err != nil {
						fmt.Printf("Warning: Invalid amount format: %v\n", err)
						continue
					}

					txn := domain.BankTransaction{
						UniqID: row[columnMap["unique_identifier"]],
						Amount: amount,
						Date:   txDate,
						BankID: bankID,
					}

					batchResults = append(batchResults, txn)
				}

				// Send the batch results if any transactions were found
				if len(batchResults) > 0 {
					results <- batchResults
				}
			}
		}()
	}

}

// collectBankTxnResults gathers processed bank transactions from all workers
func collectBankTxnResults(results <-chan []domain.BankTransaction, errChan <-chan error) ([]domain.BankTransaction, error) {
	var txns []domain.BankTransaction

	for batch := range results {
		// Check for errors (non-blocking)
		select {
		case err := <-errChan:
			return nil, err
		default:
			// Continue if no errors
		}

		txns = append(txns, batch...)
	}

	// Final error check after all results are collected
	select {
	case err := <-errChan:
		return nil, err
	default:
		// No errors
	}

	return txns, nil
}
