package fileutil

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

// CSVReader provides a helper/utility to read CSV file(s)
type CSVReader struct {
	FilePath string
}

// NewCSVReader returns a CSVReader instance for a specified CSV file
func NewCSVReader(fp string) *CSVReader {
	return &CSVReader{
		FilePath: fp,
	}
}

// ReadHeader reads ONLY the header of the specified CSV file
func (r *CSVReader) ReadHeader() ([]string, error) {
	f, err := os.Open(r.FilePath)
	if err != nil {
		return nil, fmt.Errorf("opening a csv file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("reading CSV header: %w", err)
	}

	return header, nil
}

// ReadAndProcessByRow reads and processes a CSV file row by row, allows for streaming large file(s)
func (r *CSVReader) ReadAndProcessByRow(processorFn func([]string) error) error {
	f, err := os.Open(r.FilePath)
	if err != nil {
		return fmt.Errorf("opening a csv file: %w", err)
	}
	defer f.Close()

	reader := csv.NewReader(f)

	// Skip header
	_, err = reader.Read()
	if err != nil {
		return fmt.Errorf("reading CSV header: %w", err)
	}

	// read and process row by row
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break // end of file, stop
		}
		if err != nil {
			return fmt.Errorf("reading CSV row: %w", err)
		}

		if err = processorFn(row); err != nil {
			return err
		}
	}

	return nil
}
