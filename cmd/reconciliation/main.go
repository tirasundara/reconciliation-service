package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/tirasundara/reconciliation-service/internal/domain"
	"github.com/tirasundara/reconciliation-service/internal/matcher"
	"github.com/tirasundara/reconciliation-service/internal/report"
	"github.com/tirasundara/reconciliation-service/internal/repository"
	"github.com/tirasundara/reconciliation-service/internal/service"
)

const (
	dateFormat     = "2006-01-02"
	sysTimeFormat  = "2006-01-02T15:04:05"
	bankDateFormat = "2006-01-02"
)

func main() {
	// Command-line flags
	var (
		systemFile      string
		bankFiles       string
		startDateStr    string
		endDateStr      string
		outputFormat    string
		outputFile      string
		dateBufferDays  int
		amountThreshold float64
		prettyPrint     bool
	)

	flag.StringVar(&systemFile, "system-file", "", "Path to system transactions CSV file")
	flag.StringVar(&bankFiles, "bank-files", "", "Comma-separated paths to bank statement CSV files")
	flag.StringVar(&startDateStr, "start-date", "", "Start date for reconciliation (YYYY-MM-DD)")
	flag.StringVar(&endDateStr, "end-date", "", "End date for reconciliation (YYYY-MM-DD)")
	flag.StringVar(&outputFormat, "format", "json", "Output format: json only for now")
	flag.StringVar(&outputFile, "output", "", "Path to output file (if empty, writes to stdout)")
	flag.IntVar(&dateBufferDays, "date-buffer", 1, "Number of days to extend search range on both ends for matching")
	flag.Float64Var(&amountThreshold, "amount-threshold", 0.10, "Maximum amount difference to consider transactions matched")
	flag.BoolVar(&prettyPrint, "pretty", true, "Pretty print JSON output")

	flag.Parse()

	// Validate required flags
	if systemFile == "" {
		exitWithError("System transactions file path is required")
	}
	if bankFiles == "" {
		exitWithError("At least one bank statement file path is required")
	}
	if startDateStr == "" {
		exitWithError("Start date is required")
	}
	if endDateStr == "" {
		exitWithError("End date is required")
	}

	// Parse dates
	startDate, err := time.Parse(dateFormat, startDateStr)
	if err != nil {
		exitWithError(fmt.Sprintf("Invalid start date format: %v", err))
	}

	endDate, err := time.Parse(dateFormat, endDateStr)
	if err != nil {
		exitWithError(fmt.Sprintf("Invalid end date format: %v", err))
	}

	// Add a day to end date to make it inclusive
	endDate = endDate.AddDate(0, 0, 1).Add(-time.Second)

	// Ensure dates are in the correct order
	if endDate.Before(startDate) {
		exitWithError("End date must be after start date")
	}

	// Create system repository
	systemRepo := repository.NewCSVSystemRepository(systemFile, sysTimeFormat)

	// Create bank repositories
	bankRepos := make(map[string]domain.BankTransactionRepository)
	for _, bankFile := range strings.Split(bankFiles, ",") {
		bankFile = strings.TrimSpace(bankFile)
		if bankFile == "" {
			continue
		}

		repo := repository.NewCSVBankRepository(bankFile, bankDateFormat)
		bankRepos[repo.GetBankIdentifier()] = repo
	}

	if len(bankRepos) == 0 {
		exitWithError("No valid bank statement files provided")
	}

	// Create matcher with strategies
	matcherWithStrategies := matcher.NewDefaultMatcher(
		matcher.NewExactMatchStrategy(),
		matcher.NewFuzzyMatchStrategy(amountThreshold),
		matcher.NewDateBufferMatchStrategy(dateBufferDays),
	)

	// Create reconciliation service
	reconciliationService := service.NewReconciliationService(systemRepo, bankRepos, matcherWithStrategies, dateBufferDays)

	// Run reconciliation
	result, err := reconciliationService.Reconcile(startDate, endDate)
	if err != nil {
		exitWithError(fmt.Sprintf("Reconciliation failed: %v", err))
	}

	// Format the output
	var formatter report.OutputFormatter
	switch outputFormat {
	case "json":
		formatter = report.NewJSONFormatter(prettyPrint)

	// Can add other formatters later: csv, txt, etc
	default:
		exitWithError(fmt.Sprintf("Unsupported output format: %s", outputFormat))
		return
	}

	output, err := formatter.Format(result)
	if err != nil {
		exitWithError(fmt.Sprintf("Failed to format output: %v", err))
	}

	// Output the result
	if outputFile != "" {
		// If no extension is provided, add the formatter's default extension
		if !strings.Contains(outputFile, ".") {
			outputFile = fmt.Sprintf("%s.%s", outputFile, formatter.FileExtension())
		}

		err := os.WriteFile(outputFile, output, 0644)
		if err != nil {
			exitWithError(fmt.Sprintf("Failed to write output file: %v", err))
		}

	} else {

		// Write output to stdout
		fmt.Println(string(output))
	}
}

func exitWithError(message string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	fmt.Fprintf(os.Stderr, "Run with -h flag for usage information.\n")
	os.Exit(1)
}
