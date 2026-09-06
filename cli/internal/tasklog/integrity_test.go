package tasklog

import (
	"os"
	"path/filepath"
	"testing"
)

func writeCSV(t *testing.T, path string, lines ...string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "uuid,task,from,to,completed\n"
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNormalOpenClosePairIsNotDangling(t *testing.T) {
	dataDir := t.TempDir()
	writeCSV(t, filepath.Join(dataDir, "projects", "acme", "task_log.csv"),
		`aaa,"Task one",2026-09-01T09:00:00Z,,`,
		`aaa,"Task one",2026-09-01T09:00:00Z,2026-09-01T10:00:00Z,true`,
	)

	dangling, err := CheckDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("CheckDataDirectory: %v", err)
	}
	if len(dangling) != 0 {
		t.Fatalf("expected no dangling entries for a matched open/close pair, got %+v", dangling)
	}
}

func TestOpenLastRowIsGivenBenefitOfTheDoubt(t *testing.T) {
	dataDir := t.TempDir()
	writeCSV(t, filepath.Join(dataDir, "projects", "acme", "task_log.csv"),
		`aaa,"Finished task",2026-09-01T09:00:00Z,2026-09-01T10:00:00Z,true`,
		`bbb,"Currently active",2026-09-06T09:00:00Z,,`,
	)

	dangling, err := CheckDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("CheckDataDirectory: %v", err)
	}
	if len(dangling) != 0 {
		t.Fatalf("expected the last, still-open row to be excused, got %+v", dangling)
	}
}

func TestOrphanedOpenRowBuriedByLaterEntriesIsDangling(t *testing.T) {
	dataDir := t.TempDir()
	writeCSV(t, filepath.Join(dataDir, "projects", "acme", "task_log.csv"),
		`bbb,"Crash victim",2026-09-06T09:00:00Z,,`, // never closed
		`ccc,"Started after relaunch",2026-09-06T09:05:00Z,,`,
		`ccc,"Started after relaunch",2026-09-06T09:05:00Z,2026-09-06T09:10:00Z,false`,
	)

	dangling, err := CheckDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("CheckDataDirectory: %v", err)
	}
	if len(dangling) != 1 || dangling[0].UUID != "bbb" {
		t.Fatalf("expected exactly one dangling entry for uuid bbb, got %+v", dangling)
	}
	if dangling[0].Project != "acme" || dangling[0].Line != 2 {
		t.Fatalf("unexpected dangling entry details: %+v", dangling[0])
	}
}

func TestManualEntryRowIsNeverDangling(t *testing.T) {
	dataDir := t.TempDir()
	writeCSV(t, filepath.Join(dataDir, "projects", "acme", "task_log.csv"),
		`aaa,"Backdated single-row entry",2026-09-01T08:00:00Z,2026-09-01T09:00:00Z,true`,
	)

	dangling, err := CheckDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("CheckDataDirectory: %v", err)
	}
	if len(dangling) != 0 {
		t.Fatalf("a single already-closed row should never be dangling, got %+v", dangling)
	}
}

func TestMalformedRowIsAnError(t *testing.T) {
	dataDir := t.TempDir()
	writeCSV(t, filepath.Join(dataDir, "projects", "acme", "task_log.csv"),
		`aaa,"Missing a field",2026-09-01T08:00:00Z,`,
	)

	if _, err := CheckDataDirectory(dataDir); err == nil {
		t.Fatalf("expected an error for a row with the wrong number of fields")
	}
}

func TestNoProjectsDirectoryIsFine(t *testing.T) {
	dataDir := t.TempDir()
	dangling, err := CheckDataDirectory(dataDir)
	if err != nil {
		t.Fatalf("CheckDataDirectory: %v", err)
	}
	if len(dangling) != 0 {
		t.Fatalf("expected no dangling entries when there's no data yet, got %+v", dangling)
	}
}
