package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
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

func (s sidebar) update(msg tea.KeyMsg) (sidebar, bool) {
	switch {
	case key.Matches(msg, sidebarKeys.Up):
		if s.active > 0 {
			s.active--
		}
	case key.Matches(msg, sidebarKeys.Down):
		if s.active < sectionCount-1 {
			s.active++
		}
	case key.Matches(msg, sidebarKeys.Select):
		return s, true // signals selection confirmed
	}
	return s, false
}

func (s sidebar) view(height int, focused bool, activeOrg string) string {
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

	nav := title + "\n"
	for i := sectionIndex(0); i < sectionCount; i++ {
		label := sectionLabels[i]
		if i == s.active {
			nav += activeItemStyle.Render("> " + label)
		} else {
			nav += itemStyle.Render("  " + label)
		}
		if i < sectionCount-1 {
			nav += "\n"
		}
	}

	// Active org indicator at the bottom.
	orgIndicator := ""
	if activeOrg != "" {
		dividerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("237")).
			Padding(0, 1)
		orgLabelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Padding(0, 1)
		orgNameStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Padding(0, 1).
			Width(sidebarWidth - 4). // fit within border + padding
			MaxWidth(sidebarWidth - 4)
		orgIndicator = "\n" + dividerStyle.Render("─────────────────") +
			"\n" + orgLabelStyle.Render("org") +
			"\n" + orgNameStyle.Render(activeOrg)
	}

	content := nav + orgIndicator

	borderStyle := unfocusedBorder
	if focused {
		borderStyle = focusedBorder
	}

	return borderStyle.
		Width(sidebarWidth - 2). // -2 for border chars
		Height(height - 2).      // -2 for border chars
		Render(content)
}

func (s sidebar) shortHelp() []key.Binding {
	return []key.Binding{sidebarKeys.Up, sidebarKeys.Down, sidebarKeys.Select}
}
