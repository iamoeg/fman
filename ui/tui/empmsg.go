package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/internal/domain"
)

type empHistoryEntry struct {
	result *domain.PayrollResult
	period *domain.PayrollPeriod
}

type empHistoryLoadedMsg struct {
	entries []empHistoryEntry
	err     error
}

func loadEmpHistoryCmd(payrollSvc *application.PayrollService, empID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		results, err := payrollSvc.ListPayrollResultsByEmployee(context.Background(), empID)
		if err != nil {
			return empHistoryLoadedMsg{err: err}
		}
		entries := make([]empHistoryEntry, 0, len(results))
		for _, r := range results {
			period, err := payrollSvc.GetPayrollPeriod(context.Background(), r.PayrollPeriodID)
			if err != nil {
				continue // skip orphaned results
			}
			entries = append(entries, empHistoryEntry{result: r, period: period})
		}
		return empHistoryLoadedMsg{entries: entries}
	}
}

type empsLoadedMsg struct {
	emps []*domain.Employee
	pkgs []*domain.EmployeeCompensationPackage
	err  error
}

type (
	saveEmpDoneMsg   struct{ err error }
	deleteEmpDoneMsg struct{ err error }
)

type empsDeletedLoadedMsg struct {
	emps []*domain.Employee
	err  error
}

type (
	restoreEmpDoneMsg    struct{ err error }
	hardDeleteEmpDoneMsg struct{ err error }
)

func loadEmpsCmd(
	empSvc *application.EmployeeService,
	compSvc *application.CompensationPackageService,
	orgID uuid.UUID,
) tea.Cmd {
	return func() tea.Msg {
		if orgID == uuid.Nil {
			return empsLoadedMsg{}
		}
		emps, err := empSvc.ListEmployeesByOrganization(context.Background(), orgID)
		if err != nil {
			return empsLoadedMsg{err: err}
		}
		pkgs, err := compSvc.ListCompensationPackages(context.Background(), orgID)
		return empsLoadedMsg{emps: emps, pkgs: pkgs, err: err}
	}
}

func createEmpCmd(svc *application.EmployeeService, emp *domain.Employee) tea.Cmd {
	return func() tea.Msg {
		err := svc.CreateEmployee(context.Background(), emp)
		return saveEmpDoneMsg{err: err}
	}
}

func updateEmpCmd(svc *application.EmployeeService, emp *domain.Employee) tea.Cmd {
	return func() tea.Msg {
		err := svc.UpdateEmployee(context.Background(), emp)
		return saveEmpDoneMsg{err: err}
	}
}

func deleteEmpCmd(svc *application.EmployeeService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.DeleteEmployee(context.Background(), id)
		return deleteEmpDoneMsg{err: err}
	}
}

func loadDeletedEmpsCmd(svc *application.EmployeeService, orgID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		if orgID == uuid.Nil {
			return empsDeletedLoadedMsg{}
		}
		emps, err := svc.ListEmployeesByOrganizationIncludingDeleted(context.Background(), orgID)
		return empsDeletedLoadedMsg{emps: emps, err: err}
	}
}

func restoreEmpCmd(svc *application.EmployeeService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.RestoreEmployee(context.Background(), id)
		return restoreEmpDoneMsg{err: err}
	}
}

func hardDeleteEmpCmd(svc *application.EmployeeService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.HardDeleteEmployee(context.Background(), id)
		return hardDeleteEmpDoneMsg{err: err}
	}
}
