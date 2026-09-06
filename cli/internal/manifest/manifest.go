// Package manifest reads and writes manifest.toml — the CLI-owned config of
// business info, clients, and projects. The widget never touches this file;
// it only ever lists projects/ subdirectories (see spec_v2.md).
package manifest

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const FileName = "manifest.toml"

type Business struct {
	Name               string `toml:"name"`
	Address            string `toml:"address"`
	Email              string `toml:"email"`
	RegistrationNumber string `toml:"registration_number,omitempty"` // company reg number, if any
	VATNote            string `toml:"vat_note,omitempty"`            // e.g. "Not registered for VAT" or "VAT No: ..." — free text, no jurisdiction assumed
	PaymentTerms       string `toml:"payment_terms,omitempty"`       // e.g. "Due upon receipt", "Net 30" — printed as-is, not computed into a date
	PaymentDetails     string `toml:"payment_details"`
}

type Client struct {
	Slug         string  `toml:"slug"`
	Name         string  `toml:"name"`
	Currency     string  `toml:"currency"`
	Rate         float64 `toml:"rate"`
	Address      string  `toml:"address"`
	ContactEmail string  `toml:"contact_email"`
}

type Project struct {
	Slug           string  `toml:"slug"`
	Client         string  `toml:"client"` // blank => non-billable
	Name           string  `toml:"name"`
	Rate           float64 `toml:"rate,omitempty"`             // overrides client rate if set
	LastInvoicedAt string  `toml:"last_invoiced_at,omitempty"` // ISO 8601; scan-skip optimization only, never billing truth
}

type Manifest struct {
	Business Business  `toml:"business"`
	Clients  []Client  `toml:"client"`
	Projects []Project `toml:"project"`
}

func manifestPath(dataDir string) string {
	return filepath.Join(dataDir, FileName)
}

// Load reads manifest.toml from the data directory.
func Load(dataDir string) (Manifest, error) {
	var m Manifest
	if _, err := toml.DecodeFile(manifestPath(dataDir), &m); err != nil {
		return Manifest{}, fmt.Errorf("loading manifest: %w", err)
	}
	return m, nil
}

// Save writes manifest.toml to the data directory.
func Save(dataDir string, m Manifest) error {
	f, err := os.Create(manifestPath(dataDir))
	if err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(m); err != nil {
		return fmt.Errorf("encoding manifest: %w", err)
	}
	return nil
}

// Exists reports whether a manifest.toml already exists in dataDir.
func Exists(dataDir string) bool {
	_, err := os.Stat(manifestPath(dataDir))
	return err == nil
}

// Bootstrap creates a fresh data directory with a minimal manifest (just the
// "personal" project) if one doesn't exist yet. If a manifest already exists,
// the directory is left untouched. Mirrors the widget's own bootstrap logic
// (spec_v2.md, "Data directory instead of a log file") so either side can be
// the first to touch a brand-new data directory.
func Bootstrap(dataDir string) error {
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return fmt.Errorf("creating data directory: %w", err)
	}
	if Exists(dataDir) {
		return nil
	}
	m := Manifest{
		Projects: []Project{
			{Slug: "personal", Client: "", Name: "Personal"},
		},
	}
	if err := Save(dataDir, m); err != nil {
		return err
	}
	// Create the project directory (not the CSV itself — the CLI never
	// writes task_log.csv, only the widget does) so "personal" shows up
	// immediately in the widget's directory-listing project picker.
	if err := os.MkdirAll(filepath.Join(dataDir, "projects", "personal"), 0o755); err != nil {
		return fmt.Errorf("creating personal project directory: %w", err)
	}
	return nil
}

// FindProject returns the project with the given slug, if any.
func (m Manifest) FindProject(slug string) (Project, bool) {
	for _, p := range m.Projects {
		if p.Slug == slug {
			return p, true
		}
	}
	return Project{}, false
}

// FindClient returns the client with the given slug, if any.
func (m Manifest) FindClient(slug string) (Client, bool) {
	for _, c := range m.Clients {
		if c.Slug == slug {
			return c, true
		}
	}
	return Client{}, false
}

// Billable reports whether a project has a client attached.
func (p Project) Billable() bool {
	return p.Client != ""
}

// EffectiveRate returns the project's rate override, falling back to its
// client's rate.
func (m Manifest) EffectiveRate(p Project) (float64, bool) {
	if p.Rate != 0 {
		return p.Rate, true
	}
	if c, ok := m.FindClient(p.Client); ok {
		return c.Rate, true
	}
	return 0, false
}
