package ingestion

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pipdaniels/geochem-agent/internal/models"
	"github.com/xuri/excelize/v2"
)

// knownMetaCols are columns that map to Assay fields; everything else is an element.
var knownMetaCols = map[string]bool{
	"sample_id":     true,
	"latitude":      true,
	"lat":           true,
	"longitude":     true,
	"lon":           true,
	"long":          true,
	"elevation":     true,
	"elev":          true,
	"lab_name":      true,
	"lab":           true,
	"collected_at":  true,
	"date":          true,
	"sampling_date": true,
	"comments":      true,
	"comment":       true,
}

// ParseCSV parses a CSV reader into a slice of Assay structs.
// labName and collectedAt are fallback values used when the file doesn't
// contain those columns.
func ParseCSV(r io.Reader, labName string, collectedAt time.Time) ([]models.Assay, error) {
	rows, err := readCSVRows(r)
	if err != nil {
		return nil, err
	}
	return rowsToAssays(rows, labName, collectedAt)
}

// ParseXLSX parses the first sheet of an XLSX file into a slice of Assay structs.
func ParseXLSX(r io.Reader, labName string, collectedAt time.Time) ([]models.Assay, error) {
	// excelize needs a ReadSeeker or reads from bytes; buffer the reader.
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("reading xlsx data: %w", err)
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parsing xlsx: %w", err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("xlsx file has no sheets")
	}

	rows, err := f.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("reading xlsx sheet: %w", err)
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("xlsx sheet is empty")
	}

	return rowsToAssays(rows, labName, collectedAt)
}

// readCSVRows reads all rows (including the header) from a CSV reader.
func readCSVRows(r io.Reader) ([][]string, error) {
	cr := csv.NewReader(r)
	cr.TrimLeadingSpace = true
	cr.FieldsPerRecord = -1 // allow variable field count
	rows, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parsing csv: %w", err)
	}
	if len(rows) < 2 {
		return nil, fmt.Errorf("csv file must have a header row and at least one data row")
	}
	return rows, nil
}

// rowsToAssays converts a 2-D slice of strings (first row = headers) to []models.Assay.
func rowsToAssays(rows [][]string, defaultLabName string, defaultCollectedAt time.Time) ([]models.Assay, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("no rows provided")
	}

	// Normalise headers
	headers := make([]string, len(rows[0]))
	for i, h := range rows[0] {
		headers[i] = strings.ToLower(strings.TrimSpace(h))
	}

	// Map header name → column index
	colIdx := make(map[string]int, len(headers))
	for i, h := range headers {
		colIdx[h] = i
	}

	// Check required column
	if _, ok := colIdx["sample_id"]; !ok {
		return nil, fmt.Errorf("required column 'sample_id' not found in file")
	}

	// Identify element columns (not a known meta column)
	type elemCol struct {
		name string
		idx  int
	}
	var elemCols []elemCol
	for i, h := range headers {
		if !knownMetaCols[h] {
			elemCols = append(elemCols, elemCol{name: h, idx: i})
		}
	}

	assays := make([]models.Assay, 0, len(rows)-1)

	for rowNum, row := range rows[1:] {
		// Pad short rows
		for len(row) < len(headers) {
			row = append(row, "")
		}

		sampleID := strings.TrimSpace(row[colIdx["sample_id"]])
		if sampleID == "" {
			continue // skip blank sample rows
		}

		// Location
		loc := models.GeoLocation{
			Latitude:  parseFloat(row, colIdx, "latitude", "lat"),
			Longitude: parseFloat(row, colIdx, "longitude", "lon", "long"),
			Elevation: parseFloat(row, colIdx, "elevation", "elev"),
		}

		// Lab name (column overrides parameter)
		labName := defaultLabName
		if v := cellValue(row, colIdx, "lab_name", "lab"); v != "" {
			labName = v
		}

		// Collected at (column overrides parameter)
		collectedAt := defaultCollectedAt
		if v := cellValue(row, colIdx, "collected_at", "date", "sampling_date"); v != "" {
			if t, err := parseDate(v); err == nil {
				collectedAt = t
			}
		}

		comments := cellValue(row, colIdx, "comments", "comment")

		// Elements
		elements := make(map[string]float64, len(elemCols))
		for _, ec := range elemCols {
			if ec.idx >= len(row) {
				continue
			}
			raw := strings.TrimSpace(row[ec.idx])
			if isBelowDetection(raw) || raw == "" {
				continue
			}
			// Strip common suffix characters like "<", ">"
			raw = strings.TrimPrefix(raw, "<")
			raw = strings.TrimPrefix(raw, ">")
			v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
			if err != nil {
				continue // non-numeric; skip silently
			}
			elements[ec.name] = v
		}

		assays = append(assays, models.Assay{
			SampleID:    sampleID,
			Location:    loc,
			Elements:    elements,
			LabName:     labName,
			CollectedAt: collectedAt,
			Comments:    comments,
		})

		_ = rowNum // suppress unused warning
	}

	if len(assays) == 0 {
		return nil, fmt.Errorf("no valid assay rows found in file")
	}

	return assays, nil
}

// --- helpers ---

func parseFloat(row []string, colIdx map[string]int, keys ...string) float64 {
	v := cellValue(row, colIdx, keys...)
	if v == "" {
		return 0
	}
	f, err := strconv.ParseFloat(strings.TrimSpace(v), 64)
	if err != nil {
		return 0
	}
	return f
}

func cellValue(row []string, colIdx map[string]int, keys ...string) string {
	for _, k := range keys {
		if idx, ok := colIdx[k]; ok && idx < len(row) {
			v := strings.TrimSpace(row[idx])
			if v != "" {
				return v
			}
		}
	}
	return ""
}

func isBelowDetection(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower == "bdl" || lower == "nd" || lower == "n/a" || lower == "na" ||
		strings.HasPrefix(lower, "<dl") || lower == "<dl"
}

var dateFormats = []string{
	"2006-01-02",
	"02/01/2006",
	"01/02/2006",
	"2006/01/02",
	"02-Jan-2006",
	"January 2, 2006",
}

func parseDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	for _, format := range dateFormats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse date: %q", s)
}
