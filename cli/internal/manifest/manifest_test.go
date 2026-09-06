package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBootstrapCreatesMinimalManifest(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "focuson-data")

	if err := Bootstrap(dataDir); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	m, err := Load(dataDir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(m.Projects) != 1 || m.Projects[0].Slug != "personal" {
		t.Fatalf("expected single personal project, got %+v", m.Projects)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "projects", "personal")); err != nil {
		t.Fatalf("expected projects/personal directory: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "projects", "personal", "task_log.csv")); !os.IsNotExist(err) {
		t.Fatalf("Bootstrap must never create task_log.csv (widget-only), got err=%v", err)
	}
}

func TestBootstrapLeavesExistingManifestUntouched(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "focuson-data")

	if err := Bootstrap(dataDir); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}
	m, _ := Load(dataDir)
	m.Business.Name = "Carl Kritzinger"
	m.Clients = append(m.Clients, Client{Slug: "acme", Name: "Acme Corp", Currency: "USD", Rate: 120})
	if err := Save(dataDir, m); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := Bootstrap(dataDir); err != nil {
		t.Fatalf("second Bootstrap: %v", err)
	}

	reloaded, err := Load(dataDir)
	if err != nil {
		t.Fatalf("Load after second Bootstrap: %v", err)
	}
	if reloaded.Business.Name != "Carl Kritzinger" || len(reloaded.Clients) != 1 {
		t.Fatalf("Bootstrap overwrote existing manifest: %+v", reloaded)
	}
}

func TestEffectiveRate(t *testing.T) {
	m := Manifest{
		Clients: []Client{{Slug: "acme", Rate: 100}},
		Projects: []Project{
			{Slug: "acme-website", Client: "acme"},
			{Slug: "acme-support", Client: "acme", Rate: 150},
			{Slug: "personal"},
		},
	}

	if rate, ok := m.EffectiveRate(m.Projects[0]); !ok || rate != 100 {
		t.Fatalf("expected fallback to client rate 100, got %v ok=%v", rate, ok)
	}
	if rate, ok := m.EffectiveRate(m.Projects[1]); !ok || rate != 150 {
		t.Fatalf("expected project override 150, got %v ok=%v", rate, ok)
	}
	if _, ok := m.EffectiveRate(m.Projects[2]); ok {
		t.Fatalf("expected no rate for non-billable personal project")
	}
	if m.Projects[2].Billable() {
		t.Fatalf("personal project should not be billable")
	}
	if !m.Projects[0].Billable() {
		t.Fatalf("acme-website should be billable")
	}
}
