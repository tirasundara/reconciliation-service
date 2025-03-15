package report

import (
	"encoding/json"

	"github.com/tirasundara/reconciliation-service/internal/domain"
)

// OutputFormatter defines the interface for formatting reconciliation results
type OutputFormatter interface {
	Format(result domain.ReconciliationResult) ([]byte, error)
	FileExtension() string
}

// JSONFormatter formats reconciliation results as JSON
type JSONFormatter struct {
	PrettyPrint bool
}

func NewJSONFormatter(prettyPrint bool) *JSONFormatter {
	return &JSONFormatter{
		PrettyPrint: prettyPrint,
	}
}

// Format implements the OutputFormatter interface for JSON
func (f *JSONFormatter) Format(result domain.ReconciliationResult) ([]byte, error) {
	if f.PrettyPrint {
		return json.MarshalIndent(result, "", "  ")
	}
	return json.Marshal(result)
}

func (f *JSONFormatter) FileExtension() string {
	return "json"
}
