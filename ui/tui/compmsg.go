package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

type compsLoadedMsg struct {
	pkgs []*domain.EmployeeCompensationPackage
	err  error
}

type saveCompDoneMsg struct{ err error }
type deleteCompDoneMsg struct{ err error }
type renameCompDoneMsg struct{ err error }

func loadCompsCmd(svc *application.CompensationPackageService, orgID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		if orgID == uuid.Nil {
			return compsLoadedMsg{}
		}
		pkgs, err := svc.ListCompensationPackages(context.Background(), orgID)
		return compsLoadedMsg{pkgs: pkgs, err: err}
	}
}

func createCompCmd(svc *application.CompensationPackageService, pkg *domain.EmployeeCompensationPackage) tea.Cmd {
	return func() tea.Msg {
		err := svc.CreateCompensationPackage(context.Background(), pkg)
		return saveCompDoneMsg{err: err}
	}
}

func deleteCompCmd(svc *application.CompensationPackageService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.DeleteCompensationPackage(context.Background(), id)
		return deleteCompDoneMsg{err: err}
	}
}

func renameCompCmd(svc *application.CompensationPackageService, id uuid.UUID, name string) tea.Cmd {
	return func() tea.Msg {
		err := svc.RenameCompensationPackage(context.Background(), id, name)
		return renameCompDoneMsg{err: err}
	}
}
