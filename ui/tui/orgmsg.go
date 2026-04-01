package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
)

// orgsLoadedMsg carries the result of listing all organizations.
type orgsLoadedMsg struct {
	orgs []*domain.Organization
	err  error
}

// saveOrgDoneMsg carries the result of a create or update operation.
type saveOrgDoneMsg struct {
	err error
}

// deleteOrgDoneMsg carries the result of a delete operation.
type deleteOrgDoneMsg struct {
	id  uuid.UUID
	err error
}

// orgsDeletedLoadedMsg carries the result of listing soft-deleted organizations.
type orgsDeletedLoadedMsg struct {
	orgs []*domain.Organization
	err  error
}

// restoreOrgDoneMsg carries the result of a restore operation.
type restoreOrgDoneMsg struct {
	id  uuid.UUID
	err error
}

// hardDeleteOrgDoneMsg carries the result of a hard-delete operation.
type hardDeleteOrgDoneMsg struct {
	id  uuid.UUID
	err error
}

func loadOrgsCmd(svc *application.OrganizationService) tea.Cmd {
	return func() tea.Msg {
		orgs, err := svc.ListOrganizations(context.Background())
		return orgsLoadedMsg{orgs: orgs, err: err}
	}
}

func createOrgCmd(svc *application.OrganizationService, org *domain.Organization) tea.Cmd {
	return func() tea.Msg {
		err := svc.CreateOrganization(context.Background(), org)
		return saveOrgDoneMsg{err: err}
	}
}

func updateOrgCmd(svc *application.OrganizationService, org *domain.Organization) tea.Cmd {
	return func() tea.Msg {
		err := svc.UpdateOrganization(context.Background(), org)
		return saveOrgDoneMsg{err: err}
	}
}

func deleteOrgCmd(svc *application.OrganizationService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.DeleteOrganization(context.Background(), id)
		return deleteOrgDoneMsg{id: id, err: err}
	}
}

func loadDeletedOrgsCmd(svc *application.OrganizationService) tea.Cmd {
	return func() tea.Msg {
		orgs, err := svc.ListOrganizationsIncludingDeleted(context.Background())
		return orgsDeletedLoadedMsg{orgs: orgs, err: err}
	}
}

func restoreOrgCmd(svc *application.OrganizationService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.RestoreOrganization(context.Background(), id)
		return restoreOrgDoneMsg{id: id, err: err}
	}
}

func hardDeleteOrgCmd(svc *application.OrganizationService, id uuid.UUID) tea.Cmd {
	return func() tea.Msg {
		err := svc.HardDeleteOrganization(context.Background(), id)
		return hardDeleteOrgDoneMsg{id: id, err: err}
	}
}

// clearActiveOrgCmd clears the active org from config and notifies the root model.
func clearActiveOrgCmd(cfg *config.Config) tea.Cmd {
	return func() tea.Msg {
		cfg.DefaultOrgID = ""
		cfg.Save("") //nolint:errcheck // best-effort save; sidebar clears regardless
		return activeOrgLoadedMsg{}
	}
}

// setActiveOrgCmd persists the selected org as default_org_id in the config
// and returns an activeOrgLoadedMsg so the sidebar refreshes immediately.
func setActiveOrgCmd(cfg *config.Config, org *domain.Organization) tea.Cmd {
	return func() tea.Msg {
		cfg.DefaultOrgID = org.ID.String()
		if err := cfg.Save(""); err != nil {
			// Non-fatal: config not persisted to disk, but in-memory state is correct.
			return activeOrgLoadedMsg{name: org.Name, orgID: org.ID}
		}
		return activeOrgLoadedMsg{name: org.Name, orgID: org.ID}
	}
}
