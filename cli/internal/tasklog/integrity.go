package tasklog

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// DanglingEntry is a task session whose opening row was never closed (no
// row anywhere in the file gives its UUID a `to`), and which isn't simply
// the file's last row — the one case given the benefit of the doubt as
// "still being actively tracked right now". Anything else with no `to` is
// presumed abandoned: a crash-orphaned row (v1's documented limitation) that
// got buried under later, unrelated sessions instead of being the most
// recent thing logged.
type DanglingEntry struct {
	Project string
	Line    int
	UUID    string
	Task    string
	From    string // RFC3339, kept as a plain string for display
}

// CheckDataDirectory scans every project's task_log.csv for dangling
// entries. Called on every CLI startup — see main.go — because invoice
// generation (and everything else) trusts task_log.csv to mean what it
// says: a `to`-less row is either "in progress" or a data problem, never
// both, and only position (last row wins) can currently tell them apart.
func CheckDataDirectory(dataDir string) ([]DanglingEntry, error) {
	projectsDir := filepath.Join(dataDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", projectsDir, err)
	}

	var dangling []DanglingEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		slug := e.Name()
		path := filepath.Join(projectsDir, slug, "task_log.csv")
		rows, err := ReadRows(path)
		if err != nil {
			return nil, err
		}
		dangling = append(dangling, danglingInProject(slug, rows)...)
	}

	sort.Slice(dangling, func(i, j int) bool {
		if dangling[i].Project != dangling[j].Project {
			return dangling[i].Project < dangling[j].Project
		}
		return dangling[i].Line < dangling[j].Line
	})
	return dangling, nil
}

func danglingInProject(project string, rows []Row) []DanglingEntry {
	if len(rows) == 0 {
		return nil
	}
	lastLine := rows[len(rows)-1].Line

	closed := make(map[string]bool, len(rows))
	for _, r := range rows {
		if r.To != nil {
			closed[r.UUID] = true
		}
	}

	var out []DanglingEntry
	for _, r := range rows {
		if r.To != nil {
			continue // this row itself is closed
		}
		if closed[r.UUID] {
			continue // some other row closes this uuid — a normal open/close pair
		}
		if r.Line == lastLine {
			continue // benefit of the doubt: presumed still actively tracked
		}
		out = append(out, DanglingEntry{
			Project: project,
			Line:    r.Line,
			UUID:    r.UUID,
			Task:    r.Task,
			From:    r.From.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return out
}
