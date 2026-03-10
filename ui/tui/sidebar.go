package tui

import (
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

type sectionIndex int

const (
	sectionOrganizations sectionIndex = iota
	sectionEmployees
	sectionCompensation
	sectionPayroll
	sectionCount // must be last
)

var sectionLabels = [sectionCount]string{
	sectionOrganizations: "Organizations",
	sectionEmployees:     "Employees",
	sectionCompensation:  "Compensation",
	sectionPayroll:       "Payroll",
}

// sidebar renders the left navigation pane.
type sidebar struct {
	active sectionIndex
}

func newSidebar() sidebar {
	return sidebar{active: sectionOrganizations}
}

func (s sidebar) update(msg interface{ String() string }) (sidebar, bool) {
	switch msg.String() {
	case "k", "up":
		if s.active > 0 {
			s.active--
		}
	case "j", "down":
		if s.active < sectionCount-1 {
			s.active++
		}
	case "enter":
		return s, true // signals selection confirmed
	}
	return s, false
}

func (s sidebar) view(height int, focused bool) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		Padding(0, 1).
		MarginBottom(1)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	activeItemStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Foreground(lipgloss.Color("205")).
		Bold(true)

	focusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))

	unfocusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	title := titleStyle.Render("finmgmt")

	items := title + "\n"
	for i := sectionIndex(0); i < sectionCount; i++ {
		label := sectionLabels[i]
		if i == s.active {
			items += activeItemStyle.Render("> " + label)
		} else {
			items += itemStyle.Render("  " + label)
		}
		if i < sectionCount-1 {
			items += "\n"
		}
	}

	borderStyle := unfocusedBorder
	if focused {
		borderStyle = focusedBorder
	}

	return borderStyle.
		Width(sidebarWidth - 2). // -2 for border chars
		Height(height - 2).      // -2 for border chars
		Render(items)
}

func (s sidebar) shortHelp() []key.Binding {
	return []key.Binding{sidebarKeys.Up, sidebarKeys.Down, sidebarKeys.Select}
}
