.PHONY: build test clean lint run help

APP_NAME=reconcile
BUILD_DIR=./bin

help:
	@echo "Available commands:"
	@echo "  make build    - Build the application"
	@echo "  make test     - Run all tests"
	@echo "  make lint     - Run linters"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make run      - Run with sample data"

build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/reconciliation

test:
	@echo "Running tests..."
	@go test -v ./...

lint:
	@echo "Running linters..."
	@golangci-lint run ./...

clean:
	@echo "Cleaning up..."
	@rm -rf $(BUILD_DIR)

run: build
	@echo "Running with sample data..."
	@$(BUILD_DIR)/$(APP_NAME) \
		--system-file ./test/testdata/integrated/system_transactions.csv \
		--bank-files ./test/testdata/integrated/bank_abc.csv,./test/testdata/integrated/bank_bcd.csv \
		--start-date 2025-01-01 \
		--end-date 2025-01-31 \
		--amount-threshold 1000 \
		--format json
