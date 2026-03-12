package tui

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

type empItem struct {
	emp     *domain.Employee
	pkgName string // resolved compensation package name
}

func (i empItem) Title() string {
	return fmt.Sprintf("#%d · %s", i.emp.SerialNum, i.emp.FullName)
}

func (i empItem) Description() string {
	return fmt.Sprintf("%s · %s", i.emp.Position, i.pkgName)
}

func (i empItem) FilterValue() string {
	return i.Title()
}

type empHistoryItem struct{ entry empHistoryEntry }

func (i empHistoryItem) Title() string {
	status := "[DRAFT]"
	if i.entry.period.Status == domain.PayrollPeriodStatusFinalized {
		status = "[FINALIZED]"
	}
	return fmt.Sprintf("%s %d  %s", time.Month(i.entry.period.Month).String(), i.entry.period.Year, status)
}

func (i empHistoryItem) Description() string {
	r := i.entry.result
	return fmt.Sprintf("Net: %s  |  Gross: %s  |  Tax: %s",
		r.NetToPay.String(), r.GrossSalaryGrandTotal.String(), r.IncomeTax.String())
}

func (i empHistoryItem) FilterValue() string {
	return fmt.Sprintf("%d-%02d", i.entry.period.Year, i.entry.period.Month)
}

// pkgNameMap builds a uuid→name lookup from a package slice.
func pkgNameMap(pkgs []*domain.EmployeeCompensationPackage) map[uuid.UUID]string {
	m := make(map[uuid.UUID]string, len(pkgs))
	for _, p := range pkgs {
		m[p.ID] = p.Name
	}
	return m
}
