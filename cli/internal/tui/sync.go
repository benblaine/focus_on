package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ckritzinger/focus_on/cli/internal/gitsync"
)

func (m Model) enterSync() (tea.Model, tea.Cmd) {
	m.syncDone = false
	m.syncResult = gitsync.Result{}
	m.syncStatusLines, m.syncErr = gitsync.Status(m.dataDir)
	m.screen = screenSync
	return m, nil
}

func (m Model) updateSync(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc", "q":
		m.screen = screenMenu
	case "enter":
		if m.syncDone || m.syncErr != nil {
			return m, nil
		}
		result, err := gitsync.Commit(m.dataDir, "")
		if err != nil {
			m.syncErr = err
			return m, nil
		}
		m.syncResult = result
		m.syncDone = true
		m.syncStatusLines = nil
	}
	return m, nil
}

func (m Model) viewSync() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Sync") + "\n")
	b.WriteString(dimStyle.Render(m.dataDir) + "\n\n")

	if m.syncErr != nil {
		b.WriteString(errorStyle.Render(m.syncErr.Error()) + "\n\n")
		b.WriteString(dimStyle.Render("esc back to menu"))
		return b.String()
	}
	if m.syncDone {
		if m.syncResult.Committed {
			b.WriteString(selectedStyle.Render("Committed.") + "\n")
		} else {
			b.WriteString("Nothing new to commit.\n")
		}
		switch {
		case m.syncResult.Pushed:
			b.WriteString(selectedStyle.Render("Pushed to origin.") + "\n")
		case m.syncResult.PushError != nil:
			b.WriteString(errorStyle.Render("Push failed: "+m.syncResult.PushError.Error()) + "\n")
			b.WriteString(dimStyle.Render("(local commit is still safe — this just means it isn't on the remote yet)") + "\n")
		default:
			b.WriteString(dimStyle.Render("(no \"origin\" remote configured — commit is local only)") + "\n")
		}
		b.WriteString("\n" + dimStyle.Render("esc back to menu"))
		return b.String()
	}
	if len(m.syncStatusLines) == 0 {
		b.WriteString("Working tree is clean.\n\n")
		b.WriteString(dimStyle.Render("enter to push anyway (in case anything's committed but unpushed) · esc to cancel"))
		return b.String()
	}

	for _, line := range m.syncStatusLines {
		b.WriteString("  " + line + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("enter to commit + push · esc to cancel"))
	return b.String()
}
