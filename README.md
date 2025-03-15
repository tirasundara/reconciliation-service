# Reconciliation Service

A transaction reconciliation tool that identifies discrepancies between internal system transactions and external bank statements.

## Overview
This service helps financial teams identify:

* Unmatched transactions between internal systems and bank records
* Transactions with amount discrepancies
* Summary statistics for reconciliation periods

## Matching Strategies
The service implements three complementary matching strategies:

* **Exact Match Strategy**: Matches transactions with identical amounts on the same date, converting system DEBIT/CREDIT types to signed amounts for comparison with bank records.
* **Fuzzy Match Strategy**: Matches transactions on the same date with amounts within a configurable threshold (default 0.01), accommodating minor rounding differences or fees.
* **Date Buffer Strategy**: Matches transactions with identical amounts across multiple days (configurable buffer, default 1 day), handling overnight processing delays and transactions near midnight.

These strategies are applied sequentially, providing balance between accuracy and practical reconciliation needs.


## Architecture
The service follows **Clean Architecture** principles:

* **Domain Layer**: Core business entities and interfaces
* **Repository Layer**: Data access with streaming CSV processing
* **Matcher Layer**: Transaction matching algorithms with multiple strategies
* **Service Layer**: Orchestration of the reconciliation process
* **Report Layer**: Formatting reconciliation results
* **CLI Layer**: Command-line interface

## Repository Layer Concurrent Processing

The reconciliation service includes specialized concurrent data loading capabilities at the repository layer. This feature leverages Go's goroutines and channels to read and process CSV data concurrently, improving performance when working with large transaction files.

### Concurrent Repository Methods
The system implements dedicated methods for concurrent data access:
```go
// Standard sequential processing
txns, err := repo.GetTransactionsInRange(startDate, endDate)

// Parallel processing for improved performance
txns, err := repo.GetTransactionsInRangeConcurrently(startDate, endDate)
```

### How Repository Concurrency Works
The concurrent implementation divides CSV processing into separate stages that run concurrently:

* A coordinator goroutine reads CSV records and groups them into batches
* Multiple worker goroutines process these batches simultaneously
* Each worker independently handles parsing, validation, and filtering
* A collector aggregates processed transactions from all workers

This approach maintains the same interface and behavior as the standard methods while providing significant performance improvements.

### Benefits of Repository Concurrency

* Faster Data Loading: Processes large files much more quickly by utilizing multiple CPU cores
* Memory Efficiency: Controls memory usage through batch processing rather than loading entire files
* Resilient Processing: Continues processing valid records even when encountering occasional invalid data
* Same Interface: Requires no changes to higher-level services that consume repository data

### When To Use Concurrent Repository Methods
The concurrent repository methods are particularly beneficial when:

* Processing files with 10,000+ transactions
* Working with multiple bank statements simultaneously
* Running on multi-core systems
* Batch reconciliations need to complete quickly

For smaller datasets or simpler use cases, the standard methods remain available.


### Implementation Details
The concurrent repository implementation uses several Go concurrency patterns:

* Worker pools to distribute processing tasks
* Buffered channels to manage work queues
* WaitGroups to coordinate completion
* Non-blocking error channels for error propagation

The repository layer preserves all business validation rules and date filtering logic while enabling significant performance gains through parallel processing.


## Installation

```bash
# Clone repository
git clone https://github.com/tirasundara/reconciliation-service.git
cd reconciliation-service

# Build
make build
```

## Usage
```bash
./bin/reconcile \
  --system-file path/to/system_transactions.csv \
  --bank-files path/to/bank1.csv,path/to/bank2.csv \
  --start-date 2025-01-01 \
  --end-date 2025-01-31 \
  --format json \
```

## Options
* `--system-file` -- Path to system transactions CSV (required)
* `--bank-files` -- Comma-separated paths to bank statements (required)
* `--start-date` -- Start date (YYYY-MM-DD) (required)
* `--end-date` -- End date (YYYY-MM-DD) (required)
* `--format` -- Output format (json only for now). Default `json`
* `--output` -- Path to output file. Default prints to `stdout`
* `--date-buffer` -- Days to extend search range. Default `1`
* `--amount-threshold` -- Maximum amount difference. Default `0.01`
* `--pretty` -- Pretty print JSON. Default `true`

## Input Format
### System Transactions CSV
```csv
trxID,amount,type,transactionTime
SYS-TXN-001,100000.00,CREDIT,2025-01-15T08:30:00
SYS-TXN-002,51000.00,DEBIT,2025-01-16T09:15:00
```

### Bank Statements CSV
```csv
unique_identifier,amount,date
BABC-STMT-001,100000.00,2025-01-15
BABC-STMT-002,-50000.00,2025-01-16
```

## Development
```bash
# Run tests
make test

# Run with sample data
make run
```

## Extending
* **New Matching Strategies**: Implement the `MatchingStrategy` interface
* **New Output Formats**: Implement the `OutputFormatter` interface
* **New Data Sources**: Implement the repository interfaces
