package repository

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"sync"
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
	NumWorkers int
	BatchSize  int
}

// NewCSVSystemRepository creates a new CSVSystemRepository
func NewCSVSystemRepository(fp, dateFormat string) *CSVSystemRepository {
	if dateFormat == "" {
		dateFormat = "2006-01-02T15:04:05"
	}

	return &CSVSystemRepository{
		FilePath:   fp,
		DateFormat: dateFormat,
		NumWorkers: 4,    // Default to 4 workers
		BatchSize:  1000, // Default to 1000 records per batch
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

// GetTransactionsInRangeConcurrently reads and parse CSV rows concurrently, good for handling CSV with huge rows
func (r *CSVSystemRepository) GetTransactionsInRangeConcurrently(startDate, endDate time.Time) ([]domain.SystemTransaction, error) {
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

	columnMap, err := crateHeaderMap(header, systemHeaderFields)
	if err != nil {
		return nil, fmt.Errorf("mapping CSV column: %w", err)
	}

	// Set up concurrent processing
	jobs := make(chan [][]string, r.NumWorkers)
	results := make(chan []domain.SystemTransaction, r.NumWorkers)
	errChan := make(chan error, r.NumWorkers)

	// Start the worker pool
	var wg sync.WaitGroup
	startWorkers(r.NumWorkers, &wg, jobs, results, columnMap, r.DateFormat, startDate, endDate)

	// Start a goroutine to close results channel when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	// Read and distribute batches of CSV records to workers
	go func() {
		defer close(jobs) // Close jobs channel when done reading

		err := readAndDistributeSystemTxns(reader, jobs, r.BatchSize)
		if err != nil {
			errChan <- err
		}
	}()

	// Collect results from workers
	transactions, err := collectResults(results, errChan)
	if err != nil {
		return nil, err
	}

	return transactions, nil
}

// readAndDistributeSystemTxns reads system transaction from CSV then distribute them to Go workers
func readAndDistributeSystemTxns(csvReader *csv.Reader, jobs chan<- [][]string, batchSize int) error {
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

// startWorkers creates a pool of worker goroutines to process batches of CSV rows
func startWorkers(numWorkers int, wg *sync.WaitGroup,
	jobs <-chan [][]string, results chan<- []domain.SystemTransaction,
	columnMap map[string]int, dateFormat string, startDate, endDate time.Time) {

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
				batchResults := make([]domain.SystemTransaction, 0, len(batch))

				for _, row := range batch {

					// Skip if row doesn't have enough fields
					if len(row) <= maxIndex {
						fmt.Printf("Warning: Invalid row: %v\n", row)
						continue // Resilient. We try to process as much row as possible
					}

					// Parse the transaction date/time
					txTime, err := time.Parse(dateFormat, row[columnMap["transactionTime"]])
					if err != nil {
						// Log warning but continue processing other records
						fmt.Printf("Warning: Invalid date format: %v\n", err)
						continue
					}

					// Filter by date range - only row within specified date range will be included
					txnDay := txTime.Truncate(24 * time.Hour)
					startDay := startDate.Truncate(24 * time.Hour)
					endDay := endDate.Truncate(24 * time.Hour)

					if txnDay.Before(startDay) || txnDay.After(endDay) {
						continue
					}

					// Parse amount
					amount, err := decimal.NewFromString(row[columnMap["amount"]])
					if err != nil {
						fmt.Printf("Warning: Invalid amount format: %v\n", err)
						continue
					}

					// Validate transaction type
					txnType := domain.TransactionType(row[columnMap["type"]])
					if txnType != domain.Debit && txnType != domain.Credit {
						fmt.Printf("Warning: Invalid transaction type: %s\n", txnType)
						continue
					}

					txn := domain.SystemTransaction{
						TrxID:           row[columnMap["trxID"]],
						Amount:          amount,
						Type:            txnType,
						TransactionTime: txTime,
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

// collectResults gathers processed transactions from all workers
func collectResults(results <-chan []domain.SystemTransaction, errChan <-chan error) ([]domain.SystemTransaction, error) {
	var txns []domain.SystemTransaction

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
