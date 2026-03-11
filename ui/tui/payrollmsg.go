package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// periodsLoadedMsg is sent when payroll periods for an org are loaded.
type periodsLoadedMsg struct {
	periods []*domain.PayrollPeriod
	counts  map[uuid.UUID]int // result count per period ID
	err     error
}

// resultsLoadedMsg is sent when results for a period are loaded.
type resultsLoadedMsg struct {
	results   []*domain.PayrollResult
	employees map[uuid.UUID]string // empID → full name
	err       error
}

type createPeriodDoneMsg struct{ err error }
type generateResultsDoneMsg struct {
	count int
	err   error
}
type finalizePeriodDoneMsg struct{ err error }
type unfinalizePeriodDoneMsg struct{ err error }
type deletePeriodDoneMsg struct{ err error }

func loadPeriodsCmd(svc *application.PayrollService, orgID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		if orgID == uuid.Nil {
			return periodsLoadedMsg{}
		}
		periods, err := svc.ListPayrollPeriodsByOrganization(context.Background(), orgID)
		if err != nil {
			return periodsLoadedMsg{err: err}
		}
		counts := make(map[uuid.UUID]int, len(periods))
		for _, p := range periods {
			results, rerr := svc.ListPayrollResultsByPeriod(context.Background(), p.ID)
			if rerr == nil {
				counts[p.ID] = len(results)
			}
		}
		return periodsLoadedMsg{periods: periods, counts: counts}
	}
}

func loadResultsCmd(
	payrollSvc *application.PayrollService,
	empSvc *application.EmployeeService,
	periodID uuid.UUID,
	orgID uuid.UUID,
) tea.Cmd {
	return func() tea.Msg {
		results, err := payrollSvc.ListPayrollResultsByPeriod(context.Background(), periodID)
		if err != nil {
			return resultsLoadedMsg{err: err}
		}
		empList, err := empSvc.ListEmployeesByOrganization(context.Background(), orgID)
		if err != nil {
			return resultsLoadedMsg{results: results, err: err}
		}
		names := make(map[uuid.UUID]string, len(empList))
		for _, e := range empList {
			names[e.ID] = e.FullName
		}
		return resultsLoadedMsg{results: results, employees: names}
	}
}

func createPeriodCmd(svc *application.PayrollService, period *domain.PayrollPeriod) tea.Cmd {
	return func() tea.Msg {
		err := svc.CreatePayrollPeriod(context.Background(), period)
		return createPeriodDoneMsg{err: err}
	}
}

func generateResultsCmd(svc *application.PayrollService, periodID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.GeneratePayrollResults(context.Background(), periodID)
		if err != nil {
			return generateResultsDoneMsg{err: err}
		}
		results, err := svc.ListPayrollResultsByPeriod(context.Background(), periodID)
		if err != nil {
			return generateResultsDoneMsg{err: err}
		}
		return generateResultsDoneMsg{count: len(results)}
	}
}

func finalizePeriodCmd(svc *application.PayrollService, periodID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.FinalizePayrollPeriod(context.Background(), periodID)
		return finalizePeriodDoneMsg{err: err}
	}
}

func unfinalizePeriodCmd(svc *application.PayrollService, periodID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.UnfinalizePayrollPeriod(context.Background(), periodID)
		return unfinalizePeriodDoneMsg{err: err}
	}
}

func deletePeriodCmd(svc *application.PayrollService, periodID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.DeletePayrollPeriod(context.Background(), periodID)
		return deletePeriodDoneMsg{err: err}
	}
}
