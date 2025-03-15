package repository

import (
	"fmt"
	"strings"
)

// createHeaderMap creates a map of column names to their indices
func crateHeaderMap(header []string, expectedHeader []string) (map[string]int, error) {
	columnMap := make(map[string]int)

	for _, column := range expectedHeader {
		found := false
		for i, field := range header {
			if strings.EqualFold(column, field) {
				columnMap[column] = i
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("required field '%s' not found in CSV header", column)
		}
	}

	return columnMap, nil
}
