package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/lipgloss"
)

var (
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205"))
)

// renderFooter renders a single-line help bar from a list of key bindings.
// Always appends the global quit binding at the end.
func renderFooter(width int, bindings []key.Binding) string {
	all := make([]key.Binding, len(bindings), len(bindings)+2)
	copy(all, bindings)
	all = append(all, globalKeys.SwitchPane, globalKeys.Quit)

	parts := make([]string, 0, len(all))
	for _, b := range all {
		if !b.Enabled() {
			continue
		}
		keys := b.Help().Key
		desc := b.Help().Desc
		if keys == "" || desc == "" {
			continue
		}
		parts = append(parts, footerKeyStyle.Render(keys)+" "+footerStyle.Render(desc))
	}

	line := strings.Join(parts, footerStyle.Render("  ·  "))
	return footerStyle.Width(width).Render(line)
}
