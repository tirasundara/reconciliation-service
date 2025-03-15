package domain

// TransactionMatcher defines the interface for matching system transactions with bank transactions
type TransactionMatcher interface {
	FindMatches(systemTxns []SystemTransaction, bankTxns []BankTransaction) ([]Match, error)
}

// MatchingStrategy defines a specific strategy for matching transactions
type MatchingStrategy interface {
	Match(sysTxn SystemTransaction, bankTxns []BankTransaction) (BankTransaction, bool)
}
