package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// formField is one labeled input. readOnly fields are displayed but never
// focused or edited — used for a slug once a client/project has been
// created, since other records may reference it by slug.
type formField struct {
	label    string
	input    textinput.Model
	readOnly bool
}

// form is a small reusable multi-field text form: tab/arrows move between
// fields, enter on the last editable field submits, esc cancels.
type form struct {
	title  string
	fields []formField
	focus  int
	err    error
}

func newForm(title string, labels []string, values []string, readOnly []bool) form {
	fields := make([]formField, len(labels))
	firstEditable := -1
	for i, label := range labels {
		ti := textinput.New()
		ti.Placeholder = label
		ti.CharLimit = 256
		ti.Width = 50
		if i < len(values) {
			ti.SetValue(values[i])
		}
		ro := i < len(readOnly) && readOnly[i]
		fields[i] = formField{label: label, input: ti, readOnly: ro}
		if !ro && firstEditable == -1 {
			firstEditable = i
		}
	}
	f := form{title: title, fields: fields, focus: firstEditable}
	if f.focus >= 0 {
		f.fields[f.focus].input.Focus()
	}
	return f
}

func (f form) values() []string {
	vals := make([]string, len(f.fields))
	for i, field := range f.fields {
		vals[i] = strings.TrimSpace(field.input.Value())
	}
	return vals
}

func (f form) lastEditable() int {
	for i := len(f.fields) - 1; i >= 0; i-- {
		if !f.fields[i].readOnly {
			return i
		}
	}
	return 0
}

func (f *form) moveFocus(dir int) tea.Cmd {
	if f.focus < 0 {
		return nil
	}
	f.fields[f.focus].input.Blur()
	n := len(f.fields)
	i := f.focus
	for k := 0; k < n; k++ {
		i = (i + dir + n) % n
		if !f.fields[i].readOnly {
			break
		}
	}
	f.focus = i
	return f.fields[f.focus].input.Focus()
}

// update handles one key event. The two bools returned are (submitted,
// cancelled); at most one is ever true.
func (f form) update(msg tea.KeyMsg) (form, tea.Cmd, bool, bool) {
	if f.focus < 0 {
		return f, nil, false, false
	}
	switch msg.Type {
	case tea.KeyEsc:
		return f, nil, false, true
	case tea.KeyEnter:
		if f.focus == f.lastEditable() {
			return f, nil, true, false
		}
		cmd := f.moveFocus(1)
		return f, cmd, false, false
	case tea.KeyTab, tea.KeyDown:
		cmd := f.moveFocus(1)
		return f, cmd, false, false
	case tea.KeyShiftTab, tea.KeyUp:
		cmd := f.moveFocus(-1)
		return f, cmd, false, false
	}
	var cmd tea.Cmd
	f.fields[f.focus].input, cmd = f.fields[f.focus].input.Update(msg)
	return f, cmd, false, false
}

func (f form) View() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(f.title) + "\n")
	for i, field := range f.fields {
		if field.readOnly {
			b.WriteString(dimStyle.Render(field.label+": "+field.input.Value()) + "\n")
			continue
		}
		marker := "  "
		if i == f.focus {
			marker = "▸ "
		}
		b.WriteString(marker + field.label + ": " + field.input.View() + "\n")
	}
	if f.err != nil {
		b.WriteString("\n" + errorStyle.Render(f.err.Error()) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render("tab/↑↓ to move · enter to confirm/submit · esc to cancel"))
	return b.String()
}

// renderList draws a simple cursor-navigable list, shared by the main menu
// and the Clients/Projects screens.
func renderList(title string, subtitle string, items []string, cursor int, footer string) string {
	var b strings.Builder
	b.WriteString(titleStyle.Render(title) + "\n")
	if subtitle != "" {
		b.WriteString(dimStyle.Render(subtitle) + "\n")
	}
	b.WriteString("\n")
	for i, item := range items {
		marker := "  "
		style := lipgloss.NewStyle()
		if i == cursor {
			marker = "▸ "
			style = selectedStyle
		}
		b.WriteString(marker + style.Render(item) + "\n")
	}
	b.WriteString("\n" + dimStyle.Render(footer))
	return b.String()
}
