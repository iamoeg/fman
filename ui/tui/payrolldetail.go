package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/iamoeg/fman/internal/domain"
)

// renderPayrollResultDetail renders the full payroll breakdown for a single result.
// It is shared between the payroll section and the employee history drilldown.
func renderPayrollResultDetail(
	r *domain.PayrollResult,
	empName string,
	period *domain.PayrollPeriod,
	width, height int,
) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
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
		periodTitle = fmt.Sprintf(" — %s %d",
			time.Month(period.Month).String(), period.Year)
	}

	lines := []string{
		titleStyle.Render(empName + periodTitle),
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

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}
