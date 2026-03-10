package tui

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
)

// orgState is the internal state machine for the organizations section.
type orgState int

const (
	orgStateList     orgState = iota // browsing the list
	orgStateCreating                 // create form open
	orgStateEditing                  // edit form open
	orgStateDeleting                 // delete confirmation open
)

// orgSection implements sectionModel for the Organizations section.
type orgSection struct {
	svc             *application.OrganizationService
	cfg             *config.Config
	list            list.Model
	state           orgState
	form            orgForm
	pendingDeleteID uuid.UUID
	errMsg          string
	width, height   int
}

func newOrgSection(svc *application.OrganizationService, cfg *config.Config) *orgSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Organizations"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	return &orgSection{svc: svc, cfg: cfg, list: l}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *orgSection) Init() tea.Cmd {
	return loadOrgsCmd(s.svc)
}

func (s *orgSection) IsOverlay() bool {
	return s.state == orgStateCreating ||
		s.state == orgStateEditing ||
		s.state == orgStateDeleting
}

func (s *orgSection) ShortHelp() []key.Binding {
	switch s.state {
	case orgStateCreating, orgStateEditing:
		return []key.Binding{
			formKeys.NextField,
			formKeys.PrevField,
			formKeys.Submit,
			formKeys.Cancel,
		}
	case orgStateDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	default:
		return []key.Binding{
			mainKeys.New,
			mainKeys.Edit,
			mainKeys.Delete,
			mainKeys.SetActive,
		}
	}
}

func (s *orgSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		// Match the inner content dimensions computed by the root model:
		//   mainWidth()        = width  - sidebarWidth - 2 (border chars)
		//   mainContentHeight() = height - headerHeight - footerHeight - 2 (border chars)
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		return s, nil

	case orgsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "load error: " + msg.err.Error()
			return s, nil
		}
		items := make([]list.Item, len(msg.orgs))
		for i, o := range msg.orgs {
			items[i] = orgItem{org: o}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case saveOrgDoneMsg:
		s.state = orgStateList
		if msg.err != nil {
			s.errMsg = userFriendlyOrgError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadOrgsCmd(s.svc)

	case deleteOrgDoneMsg:
		s.state = orgStateList
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = "delete failed: " + msg.err.Error()
			return s, nil
		}
		s.errMsg = ""
		return s, loadOrgsCmd(s.svc)

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	// Forward other messages (filter ticks, spinner, etc.) to the list.
	if s.state == orgStateList {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *orgSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case orgStateList:
		switch {
		case key.Matches(msg, mainKeys.New):
			s.form = newOrgForm()
			s.state = orgStateCreating
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Edit):
			selected, ok := s.list.SelectedItem().(orgItem)
			if !ok {
				return s, nil
			}
			s.form = newOrgFormFromOrg(selected.org)
			s.state = orgStateEditing
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Delete):
			selected, ok := s.list.SelectedItem().(orgItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.org.ID
			s.state = orgStateDeleting
			return s, nil

		case key.Matches(msg, mainKeys.SetActive):
			selected, ok := s.list.SelectedItem().(orgItem)
			if !ok {
				return s, nil
			}
			return s, setActiveOrgCmd(s.cfg, selected.org)
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case orgStateCreating, orgStateEditing:
		wasCreating := s.state == orgStateCreating
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			org, err := s.form.toDomain()
			if err != nil {
				s.errMsg = err.Error()
				return s, nil
			}
			s.state = orgStateList
			s.errMsg = ""
			if wasCreating {
				return s, createOrgCmd(s.svc, org)
			}
			return s, updateOrgCmd(s.svc, org)
		case formCancel:
			s.state = orgStateList
			s.errMsg = ""
			return s, nil
		default:
			return s, cmd
		}

	case orgStateDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.state = orgStateList
			s.pendingDeleteID = uuid.Nil
			return s, deleteOrgCmd(s.svc, id)
		case key.Matches(msg, confirmKeys.No):
			s.state = orgStateList
			s.pendingDeleteID = uuid.Nil
			return s, nil
		}
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *orgSection) View(width, height int) string {
	listView := s.list.View()

	// Reserve 1 row at the bottom for an error/status bar.
	statusRow := ""
	if s.errMsg != "" {
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Width(width).
			Render("  " + s.errMsg)
	}
	if statusRow != "" {
		listView = lipgloss.JoinVertical(lipgloss.Left, listView, statusRow)
	}

	switch s.state {
	case orgStateDeleting:
		return s.renderDeleteConfirm(listView, width, height)
	case orgStateCreating, orgStateEditing:
		return s.renderFormOverlay(width, height)
	}
	return listView
}

func (s *orgSection) renderFormOverlay(width, height int) string {
	title := "New Organization"
	if s.state == orgStateEditing {
		title = "Edit Organization"
	}

	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1).
		Foreground(lipgloss.Color("205"))

	errorLine := ""
	if s.errMsg != "" {
		errorLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render("  "+s.errMsg)
	}

	inner := titleStyle.Render(title) + "\n" + s.form.view() + errorLine

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(56).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

func (s *orgSection) renderDeleteConfirm(listView string, width, _ int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(orgItem); ok {
		name = selected.org.Name
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Delete %q? [y] yes  [n/esc] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// listHeight reserves 1 row at the bottom for the error/status bar.
func (s *orgSection) listHeight() int {
	if s.height <= 1 {
		return s.height
	}
	return s.height - 1
}

func userFriendlyOrgError(err error) string {
	switch {
	case errors.Is(err, application.ErrOrganizationExists):
		return "An organization with these identifiers already exists"
	case errors.Is(err, application.ErrOrganizationNotFound):
		return "Organization not found"
	default:
		return err.Error()
	}
}
