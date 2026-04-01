package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/internal/domain"
)

type compsLoadedMsg struct {
	pkgs []*domain.EmployeeCompensationPackage
	err  error
}

type (
	saveCompDoneMsg   struct{ err error }
	deleteCompDoneMsg struct{ err error }
	renameCompDoneMsg struct{ err error }
)

type compsDeletedLoadedMsg struct {
	pkgs []*domain.EmployeeCompensationPackage
	err  error
}

type (
	restoreCompDoneMsg    struct{ err error }
	hardDeleteCompDoneMsg struct{ err error }
)

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

func loadDeletedCompsCmd(svc *application.CompensationPackageService, orgID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		if orgID == uuid.Nil {
			return compsDeletedLoadedMsg{}
		}
		pkgs, err := svc.ListCompensationPackagesIncludingDeleted(context.Background(), orgID)
		return compsDeletedLoadedMsg{pkgs: pkgs, err: err}
	}
}

func restoreCompCmd(svc *application.CompensationPackageService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.RestoreCompensationPackage(context.Background(), id)
		return restoreCompDoneMsg{err: err}
	}
}

func hardDeleteCompCmd(svc *application.CompensationPackageService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.HardDeleteCompensationPackage(context.Background(), id)
		return hardDeleteCompDoneMsg{err: err}
	}
}

func renameCompCmd(svc *application.CompensationPackageService, id uuid.UUID, name string) tea.Cmd {
	return func() tea.Msg {
		err := svc.RenameCompensationPackage(context.Background(), id, name)
		return renameCompDoneMsg{err: err}
	}
}

type compUsageLoadedMsg struct {
	empCount    int64
	resultCount int64
	err         error
}

func loadCompUsageCmd(svc *application.CompensationPackageService, pkgID uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		empCount, resultCount, err := svc.GetCompensationPackageUsageCount(context.Background(), pkgID)
		return compUsageLoadedMsg{empCount: empCount, resultCount: resultCount, err: err}
	}
}
