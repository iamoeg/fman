package tui

import (
	"fmt"
	"time"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// periodItem wraps a PayrollPeriod for display in the period list.
type periodItem struct {
	period *domain.PayrollPeriod
	count  int // result count loaded alongside the period
}

func (i periodItem) Title() string {
	monthName := time.Month(i.period.Month).String()
	status := "[DRAFT]"
	if i.period.Status == domain.PayrollPeriodStatusFinalized {
		status = "[FINALIZED]"
	}
	return fmt.Sprintf("%s %d  %s", monthName, i.period.Year, status)
}

func (i periodItem) Description() string {
	if i.period.Status == domain.PayrollPeriodStatusFinalized && i.period.FinalizedAt != nil {
		return fmt.Sprintf("Finalized %s · %d employee(s)", i.period.FinalizedAt.Format("2006-01-02"), i.count)
	}
	return fmt.Sprintf("%d result(s) — press g to generate", i.count)
}

func (i periodItem) FilterValue() string {
	return fmt.Sprintf("%d-%02d", i.period.Year, i.period.Month)
}

// resultItem wraps a PayrollResult + employee name for display in the results list.
type resultItem struct {
	result  *domain.PayrollResult
	empName string
}

func (i resultItem) Title() string {
	if i.empName == "" {
		return i.result.EmployeeID.String()[:8] + "..."
	}
	return i.empName
}

func (i resultItem) Description() string {
	return fmt.Sprintf("Net: %s  |  Gross: %s  |  Tax: %s",
		i.result.NetToPay.String(),
		i.result.GrossSalaryGrandTotal.String(),
		i.result.IncomeTax.String(),
	)
}

func (i resultItem) FilterValue() string {
	return i.empName
}
