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

func loadCompsCmd(svc *application.CompensationPackageService) tea.Cmd {
	return func() tea.Msg {
		pkgs, err := svc.ListCompensationPackages(context.Background())
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
