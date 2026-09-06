package invoicing

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ckritzinger/focus_on/cli/internal/manifest"
)

func TestReconFindsNothingOnACleanBilledLog(t *testing.T) {
	dataDir, man := setupProject(t,
		`aaa,"Task one",2026-09-01T09:00:00Z,2026-09-01T10:00:00Z,true`,
	)
	if _, err := Commit(dataDir, man, Options{ProjectSlug: "acme-website"}); err != nil {
		t.Fatal(err)
	}

	issues, err := Recon(dataDir, man, 24*time.Hour)
	if err != nil {
		t.Fatalf("Recon: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected no issues on a fully-billed, recent log, got %+v", issues)
	}
}

func TestReconFlagsStaleUnbilledRow(t *testing.T) {
	dataDir, man := setupProject(t,
		`aaa,"Old forgotten task",2026-01-01T09:00:00Z,2026-01-01T10:00:00Z,true`,
	)

	issues, err := Recon(dataDir, man, 24*time.Hour)
	if err != nil {
		t.Fatalf("Recon: %v", err)
	}
	if len(issues) != 1 || issues[0].Kind != "stale" || issues[0].UUID != "aaa" {
		t.Fatalf("expected one stale issue for aaa, got %+v", issues)
	}
}

func TestReconDoesNotFlagRecentUnbilledRow(t *testing.T) {
	dataDir, man := setupProject(t)
	logPath := filepath.Join(dataDir, "projects", "acme-website", "task_log.csv")
	recent := time.Now().Add(-time.Hour)
	line := recent.Format("2006-01-02T15:04:05Z07:00") + "," + time.Now().Format("2006-01-02T15:04:05Z07:00")
	os.WriteFile(logPath, []byte("uuid,task,from,to,completed\naaa,\"Just now\","+line+",true\n"), 0o644)

	issues, err := Recon(dataDir, man, 24*time.Hour)
	if err != nil {
		t.Fatalf("Recon: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected recent unbilled work to not be flagged yet, got %+v", issues)
	}
}

func TestReconFlagsDuplicateBilling(t *testing.T) {
	dataDir, man := setupProject(t,
		`aaa,"Task one",2026-09-01T09:00:00Z,2026-09-01T10:00:00Z,true`,
	)
	if err := appendInvoiced(dataDir, "acme-website", []InvoicedRow{
		{UUID: "aaa", InvoiceNumber: "INV-0001", InvoicedAt: time.Now()},
		{UUID: "aaa", InvoiceNumber: "INV-0002", InvoicedAt: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}

	issues, err := Recon(dataDir, man, 24*time.Hour)
	if err != nil {
		t.Fatalf("Recon: %v", err)
	}
	found := false
	for _, iss := range issues {
		if iss.Kind == "duplicate" && iss.UUID == "aaa" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a duplicate issue for aaa, got %+v", issues)
	}
}

func TestReconIgnoresNonBillableProjects(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "focuson-data")
	if err := manifest.Bootstrap(dataDir); err != nil {
		t.Fatal(err)
	}
	man, _ := manifest.Load(dataDir)
	logPath := filepath.Join(dataDir, "projects", "personal", "task_log.csv")
	os.WriteFile(logPath, []byte("uuid,task,from,to,completed\naaa,\"Old\",2026-01-01T09:00:00Z,2026-01-01T10:00:00Z,true\n"), 0o644)

	issues, err := Recon(dataDir, man, time.Hour)
	if err != nil {
		t.Fatalf("Recon: %v", err)
	}
	if len(issues) != 0 {
		t.Fatalf("expected the non-billable personal project to be skipped entirely, got %+v", issues)
	}
}
