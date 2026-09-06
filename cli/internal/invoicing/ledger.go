package invoicing

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// LineItem is one billed task_log.csv row — a frozen snapshot, not a live
// reference back to the CSV (spec_v2.md, "Invoice Ledger").
type LineItem struct {
	UUID  string    `toml:"uuid"`
	Task  string    `toml:"task"`
	From  time.Time `toml:"from"`
	To    time.Time `toml:"to"`
	Hours float64   `toml:"hours"`
}

// Invoice is one invoices/INV-000N.toml file: a frozen historical record for
// humans and PDF rendering. It is never consulted for double-billing
// decisions — invoiced.csv is (see invoiced.go).
type Invoice struct {
	Number      string     `toml:"number"`
	Project     string     `toml:"project,omitempty"`
	Client      string     `toml:"client,omitempty"`
	GeneratedAt time.Time  `toml:"generated_at"`
	PeriodFrom  string     `toml:"period_from,omitempty"` // RFC3339; blank = unbounded
	PeriodTo    string     `toml:"period_to,omitempty"`
	Currency    string     `toml:"currency,omitempty"`
	Rate        float64    `toml:"rate,omitempty"`
	TotalHours  float64    `toml:"total_hours,omitempty"`
	TotalAmount float64    `toml:"total_amount,omitempty"`
	PDFPath     string     `toml:"pdf_path,omitempty"`
	Placeholder bool       `toml:"placeholder"`
	Note        string     `toml:"note,omitempty"`
	LineItems   []LineItem `toml:"line_item,omitempty"`
}

func invoicesDir(dataDir string) string {
	return filepath.Join(dataDir, "invoices")
}

func ledgerPath(dataDir, number string) string {
	return filepath.Join(invoicesDir(dataDir), number+".toml")
}

// SaveLedger writes a new invoice file. Invoices are immutable once created
// (spec_v2.md, Out of Scope: "fixing a bad invoice means deleting the file
// by hand") — refuses to overwrite an existing one.
func SaveLedger(dataDir string, inv Invoice) error {
	if inv.Number == "" {
		return fmt.Errorf("invoice has no number")
	}
	path := ledgerPath(dataDir, inv.Number)
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("%s already exists — invoices are never overwritten", path)
	}
	if err := os.MkdirAll(invoicesDir(dataDir), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", invoicesDir(dataDir), err)
	}
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(inv); err != nil {
		return fmt.Errorf("encoding %s: %w", path, err)
	}
	return nil
}

// LoadLedger reads one invoice file.
func LoadLedger(path string) (Invoice, error) {
	var inv Invoice
	if _, err := toml.DecodeFile(path, &inv); err != nil {
		return Invoice{}, fmt.Errorf("reading %s: %w", path, err)
	}
	return inv, nil
}

// ListLedgers reads every invoice in the data directory, newest first.
func ListLedgers(dataDir string) ([]Invoice, error) {
	entries, err := os.ReadDir(invoicesDir(dataDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", invoicesDir(dataDir), err)
	}
	var invoices []Invoice
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		inv, err := LoadLedger(filepath.Join(invoicesDir(dataDir), e.Name()))
		if err != nil {
			return nil, err
		}
		invoices = append(invoices, inv)
	}
	sort.Slice(invoices, func(i, j int) bool {
		return invoices[i].GeneratedAt.After(invoices[j].GeneratedAt)
	})
	return invoices, nil
}

const numberPrefix = "INV-"

func formatInvoiceNumber(n int) string {
	return fmt.Sprintf("%s%04d", numberPrefix, n)
}

func parseInvoiceNumber(number string) (int, error) {
	if !strings.HasPrefix(number, numberPrefix) {
		return 0, fmt.Errorf("invoice number %q missing %q prefix", number, numberPrefix)
	}
	return strconv.Atoi(strings.TrimPrefix(number, numberPrefix))
}

// NextInvoiceNumber is max(existing invoice numbers, placeholder or not) + 1
// — see spec_v2.md, "Invoice numbering & baseline". Consulting every ledger
// file (not just counting them) is what lets a deleted/hand-fixed invoice's
// number slot get reused correctly.
func NextInvoiceNumber(dataDir string) (int, error) {
	invoices, err := ListLedgers(dataDir)
	if err != nil {
		return 0, err
	}
	max := 0
	for _, inv := range invoices {
		n, err := parseInvoiceNumber(inv.Number)
		if err != nil {
			return 0, fmt.Errorf("invoice %q has an unparseable number: %w", inv.Number, err)
		}
		if n > max {
			max = n
		}
	}
	return max + 1, nil
}

// SetLast creates a placeholder invoice so the next real invoice continues
// numbering from wherever an old system (e.g. Harvest) left off — see
// spec_v2.md, "Invoice numbering & baseline".
func SetLast(dataDir string, number int, note string) (Invoice, error) {
	inv := Invoice{
		Number:      formatInvoiceNumber(number),
		GeneratedAt: time.Now(),
		Placeholder: true,
		Note:        note,
	}
	if err := SaveLedger(dataDir, inv); err != nil {
		return Invoice{}, err
	}
	return inv, nil
}
