package tui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	calculator "github.com/iamoeg/fman/internal/adapter/payroll"
	"github.com/iamoeg/fman/internal/domain"
)

// simFormPanelWidth is the fixed width of the left (form) panel.
// Key bindings (simRunKey, simFieldUpKey, simFieldDownKey, simCycleKey) are defined in simform.go.
const simFormPanelWidth = 42

type simSection struct {
	calc      *calculator.Calculator
	form      simForm
	hasResult bool
	result    *domain.PayrollResult
	simPeriod *domain.PayrollPeriod
	errMsg    string
	width     int
	height    int
}

func newSimSection() *simSection {
	return &simSection{
		calc: calculator.New(),
		form: newSimForm(),
	}
}

func (s *simSection) Init() tea.Cmd { return nil }

// IsOverlay always returns false: the simulator is a split-pane view, not an overlay.
// Global keys (esc → sidebar, tab → switch pane) are handled by the root model.
// Field navigation uses ↑/↓ instead of tab to avoid the conflict.
func (s *simSection) IsOverlay() bool { return false }

func (s *simSection) ShortHelp() []key.Binding {
	return []key.Binding{simRunKey, simFieldDownKey, simCycleKey}
}

func (s *simSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		return s, nil

	case tea.KeyMsg:
		f, result, cmd := s.form.update(msg)
		s.form = f
		if result == formSubmit {
			emp, pkg, err := s.form.toDomain()
			if err != nil {
				s.errMsg = err.Error()
				return s, nil
			}
			period := &domain.PayrollPeriod{
				Year:  time.Now().Year(),
				Month: int(time.Now().Month()),
			}
			calcResult, calcErr := s.calc.Calculate(context.Background(), period, emp, pkg)
			if calcErr != nil {
				s.errMsg = userFriendlySimError(calcErr)
				return s, nil
			}
			s.result = calcResult
			s.simPeriod = period
			s.hasResult = true
			s.errMsg = ""
		}
		return s, cmd
	}

	return s, nil
}

func (s *simSection) View(width, height int) string {
	formPanel := s.renderFormPanel(simFormPanelWidth, height)
	resultPanelW := width - simFormPanelWidth
	resultPanel := lipgloss.NewStyle().
		BorderStyle(lipgloss.NormalBorder()).
		BorderLeft(true).
		BorderForeground(lipgloss.Color("237")).
		Width(resultPanelW - 1). // -1 for the left border char
		Height(height).
		Render(s.renderResultContent(resultPanelW-1, height))
	return lipgloss.JoinHorizontal(lipgloss.Top, formPanel, resultPanel)
}

func (s *simSection) renderFormPanel(width, height int) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	errLine := ""
	if s.errMsg != "" {
		errLine = "\n\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render(s.errMsg)
	}

	content := titleStyle.Render("Pay Simulator") + "\n\n" + s.form.view() + errLine

	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1, 2).
		Render(content)
}

func (s *simSection) renderResultContent(width, height int) string {
	if !s.hasResult {
		hint := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Render("Fill in the form and press enter")
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, hint)
	}

	r := s.result
	period := s.simPeriod

	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	boldStyle := lipgloss.NewStyle().Bold(true)
	netStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("114"))

	row := func(label, value string) string {
		return fmt.Sprintf("  %-26s%16s", label, value)
	}
	totalRow := func(label, value string) string {
		return boldStyle.Render(fmt.Sprintf("  %-26s%16s", label, value))
	}
	divider := func(label string) string {
		return sectionStyle.Render("── " + label + " " + strings.Repeat("─", 20))
	}

	periodTitle := ""
	if period != nil {
		periodTitle = fmt.Sprintf(" — %s %d", time.Month(period.Month).String(), period.Year)
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	lines := []string{
		titleStyle.Render("Simulation" + periodTitle),
		"",
		divider("Salary"),
		row("Base Salary", r.BaseSalary.String()),
		row(fmt.Sprintf("Seniority Bonus (%dyr · %.0f%%)", r.SeniorityYears, r.SeniorityRate*100), r.SeniorityBonus.String()),
		row("Gross Salary", r.GrossSalary.String()),
		row("Other Bonuses", r.TotalOtherBonus.String()),
		totalRow("Gross (Grand Total)", r.GrossSalaryGrandTotal.String()),
		row("Family Allowance (CNSS)", r.FamilyAllowance.String()),
		"",
		divider("Employee Deductions"),
		row("CNSS - Social Allowance", r.SocialAllowanceEmployeeContrib.String()),
		row("CNSS - Job Loss", r.JobLossCompensationEmployeeContrib.String()),
		totalRow("  Total CNSS", r.TotalCNSSEmployeeContrib.String()),
		row("AMO", r.AMOEmployeeContrib.String()),
		row("Exemptions", r.TotalExemptions.String()),
		row("Taxable Net Salary", r.TaxableNetSalary.String()),
		row("Income Tax", r.IncomeTax.String()),
		"",
		divider("Net Payment"),
		row("Rounding", r.RoundingAmount.String()),
		netStyle.Render(fmt.Sprintf("  %-26s%16s", "Net to Pay", r.NetToPay.String())),
		"",
		divider("Employer Contributions"),
		row("CNSS - Social Allowance", r.SocialAllowanceEmployerContrib.String()),
		row("CNSS - Job Loss", r.JobLossCompensationEmployerContrib.String()),
		row("CNSS - Training Tax", r.TrainingTaxEmployerContrib.String()),
		row("CNSS - Family Benefits", r.FamilyBenefitsEmployerContrib.String()),
		totalRow("  Total CNSS", r.TotalCNSSEmployerContrib.String()),
		row("AMO", r.AMOEmployerContrib.String()),
	}

	return lipgloss.NewStyle().
		Padding(1, 1).
		Render(strings.Join(lines, "\n"))
}

func userFriendlySimError(err error) string {
	switch {
	case errors.Is(err, calculator.ErrUnsupportedPayrollYear):
		return "No tax rates configured for this payroll year"
	case errors.Is(err, calculator.ErrGrossSalaryBelowSMIG):
		return "Salary is below the legal minimum wage (SMIG)"
	default:
		if strings.Contains(err.Error(), "SMIG") || strings.Contains(err.Error(), "minimum wage") {
			return "Salary is below the legal minimum wage (SMIG)"
		}
		return "Calculation failed — check your inputs"
	}
}
