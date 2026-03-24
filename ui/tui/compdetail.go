package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

func renderCompDetail(
	pkg *domain.EmployeeCompensationPackage,
	empCount, payrollCount int64,
	usageLoaded bool,
	width, height int,
) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle := lipgloss.NewStyle().Width(16).Foreground(lipgloss.Color("245"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	row := func(label, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), value)
	}
	divider := func(label string) string {
		return sectionStyle.Render("── " + label + " " + strings.Repeat("─", 18))
	}

	var empValue, payrollValue string
	if !usageLoaded {
		loading := dimStyle.Render("loading…")
		empValue = loading
		payrollValue = loading
	} else {
		if empCount == 0 {
			empValue = dimStyle.Render("not assigned")
		} else {
			empValue = fmt.Sprintf("%d assigned", empCount)
		}
		payrollValue = fmt.Sprintf("%d results", payrollCount)
	}

	lines := []string{
		titleStyle.Render("Package Details"),
		"",
		row("Name", pkg.Name),
		row("Base Salary", fmt.Sprintf("%.2f %s", pkg.BaseSalary.ToMAD(), pkg.Currency)),
		row("Currency", string(pkg.Currency)),
		"",
		divider("Usage"),
		row("Employees", empValue),
		row("Payroll", payrollValue),
		"",
		divider("Dates"),
		row("Created", pkg.CreatedAt.Format("2006-01-02")),
		row("Updated", pkg.UpdatedAt.Format("2006-01-02")),
		"",
		hintStyle.Render("        [backspace] back"),
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(52).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}
