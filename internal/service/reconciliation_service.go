package service

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"github.com/tirasundara/reconciliation-service/internal/domain"
)

// ReconciliationService orchestrates the reconciliation process
type ReconciliationService struct {
	systemRepo domain.SystemTransactionRepository
	bankRepos  map[string]domain.BankTransactionRepository
	matcher    domain.TransactionMatcher
	dateBuffer int
}

// NewReconciliationService creates a new ReconciliationService
func NewReconciliationService(
	systemRepo domain.SystemTransactionRepository,
	bankRepos map[string]domain.BankTransactionRepository,
	matcher domain.TransactionMatcher,
	dateBuffer int,
) *ReconciliationService {
	return &ReconciliationService{
		systemRepo: systemRepo,
		bankRepos:  bankRepos,
		matcher:    matcher,
		dateBuffer: dateBuffer,
	}
}

// Reconcile performs the reconciliation process for the given date range
func (s *ReconciliationService) Reconcile(startDate, endDate time.Time) (domain.ReconciliationResult, error) {

	// Calculate effective date range with buffer
	effectiveStartDate := startDate.AddDate(0, 0, -s.dateBuffer)
	effectiveEndDate := endDate.AddDate(0, 0, s.dateBuffer)

	// Get system txns
	systemTxns, err := s.systemRepo.GetTransactionsInRangeConcurrently(effectiveStartDate, effectiveEndDate)
	if err != nil {
		return domain.ReconciliationResult{}, fmt.Errorf("fetching system transactions: %w", err)
	}

	// Get bank txns -- from all bank repositories
	var allBankTxns []domain.BankTransaction
	for _, repo := range s.bankRepos {
		bankTxns, err := repo.GetTransactionsInRangeConcurrently(effectiveStartDate, effectiveEndDate)
		if err != nil {
			return domain.ReconciliationResult{}, fmt.Errorf("fetching bank transactions: %w", err)
		}
		allBankTxns = append(allBankTxns, bankTxns...)
	}

	// Find matches between system and bank txns
	matches, err := s.matcher.FindMatches(systemTxns, allBankTxns)
	if err != nil {
		return domain.ReconciliationResult{}, fmt.Errorf("matching transactions: %w", err)
	}

	// Filter out matches outside the requested date range (not the buffered range)
	filteredMatches := s.filterMatchesByDateRange(matches, startDate, endDate)

	unmatchedSystemTxns := s.findUnmatchedSystemTransactions(systemTxns, filteredMatches, startDate, endDate)
	unmatchedBankTxns := s.findUnmatchedBankTransactions(allBankTxns, filteredMatches, startDate, endDate)

	totalDiscrepancies := s.calculateTotalDiscrepancies(filteredMatches)

	result := domain.ReconciliationResult{
		TotalTxnsProcessed:  len(filteredMatches) + len(unmatchedSystemTxns) + s.countBankTransactions(unmatchedBankTxns),
		MatchedTxns:         filteredMatches,
		UnMatchedSystemTxns: unmatchedSystemTxns,
		UnMatchedBankTxns:   unmatchedBankTxns,
		TotalDiscrepancies:  totalDiscrepancies,
	}

	return result, nil
}

func (s *ReconciliationService) filterMatchesByDateRange(matches []domain.Match, startDate, endDate time.Time) []domain.Match {
	var filtered []domain.Match

	startDay := startDate.Truncate(24 * time.Hour)
	endDay := endDate.Truncate(24 * time.Hour)

	for _, match := range matches {
		txnDay := match.SystemTxn.TransactionTime.Truncate(24 * time.Hour)
		if (txnDay.Equal(startDay) || txnDay.After(startDay)) && (txnDay.Equal(endDay) || txnDay.Before(endDay)) {
			filtered = append(filtered, match)
		}
	}

	return filtered
}

func (s *ReconciliationService) findUnmatchedSystemTransactions(
	systemTxns []domain.SystemTransaction,
	matches []domain.Match,
	startDate, endDate time.Time,
) []domain.SystemTransaction {

	matchedIDs := make(map[string]bool)
	for _, match := range matches {
		matchedIDs[match.SystemTxn.TrxID] = true
	}

	// Find unmatched txns
	var unmatched []domain.SystemTransaction

	startDay := startDate.Truncate(24 * time.Hour)
	endDay := endDate.Truncate(24 * time.Hour)

	for _, txn := range systemTxns {
		// Skip if already matched
		if matchedIDs[txn.TrxID] {
			continue
		}

		// Only include txns within the requested date range
		txnDay := txn.TransactionTime.Truncate(24 * time.Hour)
		if (txnDay.Equal(startDay) || txnDay.After(startDay)) && (txnDay.Equal(endDay) || txnDay.Before(endDay)) {
			unmatched = append(unmatched, txn)
		}
	}

	return unmatched
}

func (s *ReconciliationService) findUnmatchedBankTransactions(
	bankTxns []domain.BankTransaction,
	matches []domain.Match,
	startDate, endDate time.Time,
) map[string][]domain.BankTransaction {

	matchedIDs := make(map[string]bool)
	for _, match := range matches {
		key := fmt.Sprintf("%s-%s", match.BankTxn.BankID, match.BankTxn.UniqID)
		matchedIDs[key] = true
	}

	// Find unmatched txns grouped by bank
	unmatched := make(map[string][]domain.BankTransaction)

	startDay := startDate.Truncate(24 * time.Hour)
	endDay := endDate.Truncate(24 * time.Hour)

	for _, txn := range bankTxns {
		// Skip if already matched
		key := fmt.Sprintf("%s-%s", txn.BankID, txn.UniqID)
		if matchedIDs[key] {
			continue
		}

		txnDay := txn.Date.Truncate(24 * time.Hour)
		if (txnDay.Equal(startDay) || txnDay.After(startDay)) && (txnDay.Equal(endDay) || txnDay.Before(endDay)) {
			unmatched[txn.BankID] = append(unmatched[txn.BankID], txn)
		}
	}

	return unmatched
}

func (s *ReconciliationService) calculateTotalDiscrepancies(matches []domain.Match) decimal.Decimal {
	total := decimal.Zero

	for _, match := range matches {
		total = total.Add(match.AmmountDiff)
	}

	return total
}

func (s *ReconciliationService) countBankTransactions(txnsByBank map[string][]domain.BankTransaction) int {
	count := 0
	for _, txns := range txnsByBank {
		count += len(txns)
	}
	return count
}
