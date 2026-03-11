package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

type empsLoadedMsg struct {
	emps []*domain.Employee
	pkgs []*domain.EmployeeCompensationPackage
	err  error
}

type saveEmpDoneMsg struct{ err error }
type deleteEmpDoneMsg struct{ err error }

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
