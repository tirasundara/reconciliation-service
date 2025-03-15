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
