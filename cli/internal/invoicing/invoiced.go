package invoicing

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// InvoicedRow is one line of a project's invoiced.csv — the append-only map
// from a task_log.csv UUID to the invoice that billed it (spec_v2.md,
// "invoiced.csv"). Written only here, by the CLI, never by the widget.
type InvoicedRow struct {
	UUID          string
	InvoiceNumber string
	InvoicedAt    time.Time
}

func invoicedPath(dataDir, projectSlug string) string {
	return filepath.Join(dataDir, "projects", projectSlug, "invoiced.csv")
}

// readInvoiced reads a project's invoiced.csv. A missing file just means
// nothing has ever been invoiced for that project yet.
func readInvoiced(path string) ([]InvoicedRow, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	if _, err := r.Read(); err != nil { // header
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("reading %s header: %w", path, err)
	}

	var rows []InvoicedRow
	line := 1
	for {
		fields, err := r.Read()
		line++
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s:%d: %w", path, line, err)
		}
		if len(fields) != 3 {
			return nil, fmt.Errorf("%s:%d: expected 3 fields, got %d", path, line, len(fields))
		}
		at, err := time.Parse(time.RFC3339, fields[2])
		if err != nil {
			return nil, fmt.Errorf("%s:%d: invalid invoiced_at %q: %w", path, line, fields[2], err)
		}
		rows = append(rows, InvoicedRow{UUID: fields[0], InvoiceNumber: fields[1], InvoicedAt: at})
	}
	return rows, nil
}

// billedUUIDs returns the set of task_log.csv UUIDs already claimed by some
// invoice for this project — the actual double-billing prevention mechanism
// (spec_v2.md, "Double-billing prevention"). Never trust timestamps alone.
func billedUUIDs(dataDir, projectSlug string) (map[string]bool, error) {
	rows, err := readInvoiced(invoicedPath(dataDir, projectSlug))
	if err != nil {
		return nil, err
	}
	billed := make(map[string]bool, len(rows))
	for _, r := range rows {
		billed[r.UUID] = true
	}
	return billed, nil
}

// appendInvoiced appends one row per newly-billed UUID. Append-only, same as
// task_log.csv — existing rows are never touched.
func appendInvoiced(dataDir, projectSlug string, rows []InvoicedRow) error {
	if len(rows) == 0 {
		return nil
	}
	path := invoicedPath(dataDir, projectSlug)
	isNew := false
	if _, err := os.Stat(path); os.IsNotExist(err) {
		isNew = true
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	if isNew {
		if err := w.Write([]string{"uuid", "invoice_number", "invoiced_at"}); err != nil {
			return err
		}
	}
	for _, r := range rows {
		if err := w.Write([]string{r.UUID, r.InvoiceNumber, r.InvoicedAt.Format(time.RFC3339)}); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
