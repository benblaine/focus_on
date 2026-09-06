package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetBusinessRoundTrips(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	if err := Bootstrap(dataDir); err != nil {
		t.Fatal(err)
	}

	b := Business{
		Name:               "Carl Kritzinger",
		Address:            "1 Example Street; Cape Town",
		RegistrationNumber: "2017/360065/07",
		VATNote:            "Not registered for VAT",
		PaymentTerms:       "Due upon receipt",
		PaymentDetails:     "Bank: Example Bank; Acc #: 12345",
	}
	if err := SetBusiness(dataDir, b); err != nil {
		t.Fatalf("SetBusiness: %v", err)
	}

	m, err := Load(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if m.Business != b {
		t.Fatalf("business didn't round-trip: got %+v, want %+v", m.Business, b)
	}
	// SetBusiness must not disturb the rest of the manifest.
	if len(m.Projects) != 1 || m.Projects[0].Slug != "personal" {
		t.Fatalf("SetBusiness disturbed projects: %+v", m.Projects)
	}
}

func TestAddClientDerivesSlugAndRejectsDuplicates(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	if err := Bootstrap(dataDir); err != nil {
		t.Fatalf("Bootstrap: %v", err)
	}

	c, err := AddClient(dataDir, Client{Name: "Acme Corp", Currency: "USD", Rate: 120})
	if err != nil {
		t.Fatalf("AddClient: %v", err)
	}
	if c.Slug != "acme-corp" {
		t.Fatalf("expected derived slug acme-corp, got %q", c.Slug)
	}

	if _, err := AddClient(dataDir, Client{Name: "Acme Corp"}); err == nil {
		t.Fatalf("expected duplicate slug error, got nil")
	}
}

func TestUpdateClientKeepsSlugImmutable(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	Bootstrap(dataDir)
	c, _ := AddClient(dataDir, Client{Name: "Acme Corp", Rate: 100})

	if err := UpdateClient(dataDir, c.Slug, Client{Slug: "hijacked", Name: "Acme Corporation", Rate: 150}); err != nil {
		t.Fatalf("UpdateClient: %v", err)
	}
	m, _ := Load(dataDir)
	got, ok := m.FindClient("acme-corp")
	if !ok {
		t.Fatalf("client slug changed unexpectedly, want it to stay acme-corp")
	}
	if got.Name != "Acme Corporation" || got.Rate != 150 {
		t.Fatalf("update didn't apply: %+v", got)
	}
}

func TestAddProjectRejectsUnknownClientAndCreatesDir(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	Bootstrap(dataDir)

	if _, err := AddProject(dataDir, Project{Name: "Ghost", Client: "nope"}); err == nil {
		t.Fatalf("expected error referencing unknown client")
	}

	AddClient(dataDir, Client{Name: "Acme Corp", Rate: 100})
	p, err := AddProject(dataDir, Project{Name: "Acme Website", Client: "acme-corp"})
	if err != nil {
		t.Fatalf("AddProject: %v", err)
	}
	if !dirExists(filepath.Join(dataDir, "projects", p.Slug)) {
		t.Fatalf("expected projects/%s directory to exist immediately", p.Slug)
	}
}

func TestUpdateProjectValidatesClientReference(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	Bootstrap(dataDir)
	AddClient(dataDir, Client{Name: "Acme Corp", Rate: 100})
	p, _ := AddProject(dataDir, Project{Name: "Acme Website", Client: "acme-corp"})

	if err := UpdateProject(dataDir, p.Slug, Project{Name: "Acme Website", Client: "does-not-exist"}); err == nil {
		t.Fatalf("expected error for unknown client reference")
	}
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
