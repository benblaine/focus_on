package pdfgen

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-pdf/fpdf"

	"github.com/ckritzinger/focus_on/cli/internal/invoicing"
	"github.com/ckritzinger/focus_on/cli/internal/manifest"
)

// TestHeaderAdvancesPastTheLongerColumn is a regression test for a real
// layout bug: the header draws business info (left) and invoice metadata
// (right) as two independent columns starting at the same Y, but only ever
// continued layout from wherever the *last-drawn* column's cursor ended up —
// which is the right column, since it's drawn second. A business block with
// more lines than the invoice metadata (name+address+email+reg
// number+VAT note is easily 5+ lines; the metadata is often just 2-3) meant
// "Bill To" started drawing while the business address was still going,
// producing visibly overlapping/interleaved text.
func TestHeaderAdvancesPastTheLongerColumn(t *testing.T) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginMM, marginMM, marginMM)
	pdf.AddPage()
	r := renderer{pdf: pdf, tr: pdf.UnicodeTranslatorFromDescriptor("")}

	top := pdf.GetY()
	business := manifest.Business{
		Name:               "Intuitably (Pty) Ltd",
		Address:            "8 Kort Street; The Island; Sedgefield; 6573",
		Email:              "accounts@intuitably.com",
		RegistrationNumber: "2017/360065/07",
		VATNote:            "Not registered for VAT",
	}
	// Deliberately short right column: no period, no payment terms.
	inv := invoicing.Invoice{Number: "INV-0045", GeneratedAt: time.Now()}

	r.header(inv, business)
	gotY := pdf.GetY()

	// The right column alone (INVOICE + number + date: 3 lines) would leave
	// Y at roughly top+18+6(Ln)+4(Ln) = top+28. The left column's 7 lines
	// (name at 8mm + 6 more at 5mm) push it to roughly top+43+6+4 = top+53.
	// Assert we're well past where the short-column bug would have left us.
	minExpectedY := top + 40
	if gotY < minExpectedY {
		t.Fatalf("header ended at Y=%.1f, expected >= %.1f (past the longer business column, not just the short invoice-metadata column)", gotY, minExpectedY)
	}
}

func TestRenderProducesAValidLookingPDF(t *testing.T) {
	inv := invoicing.Invoice{
		Number:      "INV-0042",
		Project:     "acme-website",
		Client:      "acme-corp",
		GeneratedAt: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
		Currency:    "USD",
		Rate:        120,
		TotalHours:  3.5,
		TotalAmount: 420,
		LineItems: []invoicing.LineItem{
			{UUID: "aaa", Task: "Homepage redesign with a genuinely quite long description that should truncate", From: time.Date(2026, 9, 1, 9, 0, 0, 0, time.UTC), To: time.Date(2026, 9, 1, 11, 0, 0, 0, time.UTC), Hours: 2},
			{UUID: "bbb", Task: "Café menu page (accented chars)", From: time.Date(2026, 9, 2, 9, 0, 0, 0, time.UTC), To: time.Date(2026, 9, 2, 10, 30, 0, 0, time.UTC), Hours: 1.5},
		},
	}
	business := manifest.Business{
		Name:               "Carl Kritzinger",
		Address:            "1 Example Street; Cape Town; South Africa",
		Email:              "carl@example.com",
		RegistrationNumber: "2017/360065/07",
		VATNote:            "Not registered for VAT",
		PaymentTerms:       "Due upon receipt",
		PaymentDetails:     "Bank: Example Bank\nAccount: 123456789",
	}
	client := manifest.Client{
		Slug:         "acme-corp",
		Name:         "Acme Corp",
		Address:      "500 Business Ave; Denver, CO",
		ContactEmail: "billing@acme.example",
	}

	outPath := filepath.Join(t.TempDir(), "INV-0042.pdf")
	if err := Render(inv, business, client, outPath); err != nil {
		t.Fatalf("Render: %v", err)
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("reading rendered PDF: %v", err)
	}
	if len(data) < 500 {
		t.Fatalf("rendered PDF suspiciously small: %d bytes", len(data))
	}
	if string(data[:5]) != "%PDF-" {
		t.Fatalf("output doesn't start with a PDF header: %q", data[:5])
	}
}

func TestRenderHandlesEmptyOptionalFields(t *testing.T) {
	inv := invoicing.Invoice{
		Number:      "INV-0001",
		GeneratedAt: time.Now(),
		Currency:    "USD",
		LineItems: []invoicing.LineItem{
			{UUID: "aaa", Task: "Only task", From: time.Now(), To: time.Now().Add(time.Hour), Hours: 1},
		},
	}
	outPath := filepath.Join(t.TempDir(), "INV-0001.pdf")
	if err := Render(inv, manifest.Business{}, manifest.Client{}, outPath); err != nil {
		t.Fatalf("Render with empty business/client: %v", err)
	}
}
