package manifest

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Slugify turns a display name into a filesystem-and-TOML-safe slug:
// lowercase, alphanumerics and hyphens only, no leading/trailing/doubled
// hyphens.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// These Add/Update functions are the single implementation of "how a
// client/project gets created or edited" — both the TUI forms and the
// scripting flags (spec_v2.md Phase 5) call these directly rather than each
// re-implementing validation.

// AddClient validates and appends a new client, deriving its slug from Name
// if Slug is blank. Slug is immutable after creation.
func AddClient(dataDir string, c Client) (Client, error) {
	m, err := Load(dataDir)
	if err != nil {
		return Client{}, err
	}
	if c.Slug == "" {
		c.Slug = Slugify(c.Name)
	}
	if c.Slug == "" {
		return Client{}, fmt.Errorf("client needs a name")
	}
	if _, exists := m.FindClient(c.Slug); exists {
		return Client{}, fmt.Errorf("a client with slug %q already exists", c.Slug)
	}
	m.Clients = append(m.Clients, c)
	if err := Save(dataDir, m); err != nil {
		return Client{}, err
	}
	return c, nil
}

// UpdateClient replaces the fields of an existing client, keyed by its
// (immutable) slug.
func UpdateClient(dataDir, slug string, updated Client) error {
	m, err := Load(dataDir)
	if err != nil {
		return err
	}
	for i, c := range m.Clients {
		if c.Slug == slug {
			updated.Slug = slug
			m.Clients[i] = updated
			return Save(dataDir, m)
		}
	}
	return fmt.Errorf("no client with slug %q", slug)
}

// AddProject validates and appends a new project, deriving its slug from
// Name if Slug is blank, and creates projects/<slug>/ immediately so it
// shows up in the widget's directory-listing project picker right away.
func AddProject(dataDir string, p Project) (Project, error) {
	m, err := Load(dataDir)
	if err != nil {
		return Project{}, err
	}
	if p.Slug == "" {
		p.Slug = Slugify(p.Name)
	}
	if p.Slug == "" {
		return Project{}, fmt.Errorf("project needs a name")
	}
	if _, exists := m.FindProject(p.Slug); exists {
		return Project{}, fmt.Errorf("a project with slug %q already exists", p.Slug)
	}
	if p.Client != "" {
		if _, ok := m.FindClient(p.Client); !ok {
			return Project{}, fmt.Errorf("no client with slug %q", p.Client)
		}
	}
	m.Projects = append(m.Projects, p)
	if err := Save(dataDir, m); err != nil {
		return Project{}, err
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "projects", p.Slug), 0o755); err != nil {
		return Project{}, fmt.Errorf("creating project directory: %w", err)
	}
	return p, nil
}

// SetBusiness replaces the [business] block wholesale — unlike clients and
// projects it's a singleton, so there's no slug/identity to preserve.
func SetBusiness(dataDir string, b Business) error {
	m, err := Load(dataDir)
	if err != nil {
		return err
	}
	m.Business = b
	return Save(dataDir, m)
}

// SetLastInvoicedAt bumps a project's scan-skip cursor after a successful
// invoice run. This is purely a performance optimization for large logs —
// invoicing's actual double-billing prevention is invoiced.csv, never this
// timestamp (spec_v2.md, "Double-billing prevention").
func SetLastInvoicedAt(dataDir, slug string, at time.Time) error {
	m, err := Load(dataDir)
	if err != nil {
		return err
	}
	for i, p := range m.Projects {
		if p.Slug == slug {
			m.Projects[i].LastInvoicedAt = at.Format(time.RFC3339)
			return Save(dataDir, m)
		}
	}
	return fmt.Errorf("no project with slug %q", slug)
}

// UpdateProject replaces the fields of an existing project, keyed by its
// (immutable) slug.
func UpdateProject(dataDir, slug string, updated Project) error {
	m, err := Load(dataDir)
	if err != nil {
		return err
	}
	if updated.Client != "" {
		if _, ok := m.FindClient(updated.Client); !ok {
			return fmt.Errorf("no client with slug %q", updated.Client)
		}
	}
	for i, p := range m.Projects {
		if p.Slug == slug {
			updated.Slug = slug
			m.Projects[i] = updated
			return Save(dataDir, m)
		}
	}
	return fmt.Errorf("no project with slug %q", slug)
}
