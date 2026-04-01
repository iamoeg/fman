package tui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/internal/domain"
	"github.com/iamoeg/fman/pkg/config"
)

// orgState is the internal state machine for the organizations section.
type orgState int

const (
	orgStateList         orgState = iota // browsing the list
	orgStateCreating                     // create form open
	orgStateEditing                      // edit form open
	orgStateDeleting                     // delete confirmation open
	orgStateDetail                       // read-only detail overlay
	orgStateDeleted                      // browsing soft-deleted orgs
	orgStateHardDeleting                 // hard-delete confirmation overlay
)

var orgDetailKey = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "view details"),
)

// orgSection implements sectionModel for the Organizations section.
type orgSection struct {
	svc             *application.OrganizationService
	cfg             *config.Config
	list            list.Model
	state           orgState
	form            orgForm
	pendingDeleteID uuid.UUID
	detailTarget    *domain.Organization
	errMsg          string
	width, height   int
}

func newOrgSection(svc *application.OrganizationService, cfg *config.Config) *orgSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Organizations"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.NoItems = l.Styles.NoItems.PaddingLeft(2)
	return &orgSection{svc: svc, cfg: cfg, list: l}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *orgSection) Init() tea.Cmd {
	return loadOrgsCmd(s.svc)
}

func (s *orgSection) IsOverlay() bool {
	if s.state == orgStateList || s.state == orgStateDeleted {
		return s.list.FilterState() == list.Filtering
	}
	return true
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
	case orgStateDetail:
		return []key.Binding{sectionBackKey}
	case orgStateDeleted:
		return []key.Binding{
			mainKeys.ToggleDeleted,
			mainKeys.Restore,
			mainKeys.HardDelete,
		}
	case orgStateHardDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	default:
		return []key.Binding{
			orgDetailKey,
			mainKeys.New,
			mainKeys.Edit,
			mainKeys.Delete,
			mainKeys.SetActive,
			mainKeys.Filter,
			mainKeys.ToggleDeleted,
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
			s.errMsg = "Could not load organizations — try again"
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
			s.errMsg = "Delete failed — try again"
			return s, nil
		}
		s.errMsg = ""
		cmds := []tea.Cmd{loadOrgsCmd(s.svc)}
		if s.cfg != nil && msg.id.String() == s.cfg.DefaultOrgID {
			cmds = append(cmds, clearActiveOrgCmd(s.cfg))
		}
		return s, tea.Batch(cmds...)

	case orgsDeletedLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load deleted organizations — try again"
			return s, nil
		}
		var items []list.Item
		for _, o := range msg.orgs {
			if o.DeletedAt != nil {
				items = append(items, orgItem{org: o})
			}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case restoreOrgDoneMsg:
		if msg.err != nil {
			s.errMsg = "Restore failed — try again"
			return s, nil
		}
		s.errMsg = ""
		return s, loadDeletedOrgsCmd(s.svc)

	case hardDeleteOrgDoneMsg:
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = "Hard delete failed — try again"
			return s, nil
		}
		s.errMsg = ""
		return s, loadDeletedOrgsCmd(s.svc)

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	// Forward other messages (filter ticks, spinner, etc.) to the list.
	if s.state == orgStateList || s.state == orgStateDeleted {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *orgSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case orgStateDeleted:
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}
		switch {
		case key.Matches(msg, mainKeys.ToggleDeleted):
			s.list.Title = "Organizations"
			s.state = orgStateList
			s.errMsg = ""
			return s, loadOrgsCmd(s.svc)

		case key.Matches(msg, mainKeys.Restore):
			selected, ok := s.list.SelectedItem().(orgItem)
			if !ok {
				return s, nil
			}
			return s, restoreOrgCmd(s.svc, selected.org.ID)

		case key.Matches(msg, mainKeys.HardDelete):
			selected, ok := s.list.SelectedItem().(orgItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.org.ID
			s.state = orgStateHardDeleting
			return s, nil
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case orgStateHardDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.pendingDeleteID = uuid.Nil
			s.state = orgStateDeleted
			return s, hardDeleteOrgCmd(s.svc, id)
		case key.Matches(msg, confirmKeys.No):
			s.pendingDeleteID = uuid.Nil
			s.state = orgStateDeleted
			return s, nil
		}
		return s, nil

	case orgStateList:
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}
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

		case key.Matches(msg, orgDetailKey):
			if selected, ok := s.list.SelectedItem().(orgItem); ok {
				s.detailTarget = selected.org
				s.state = orgStateDetail
			}
			return s, nil

		case key.Matches(msg, mainKeys.ToggleDeleted):
			s.list.Title = "Organizations [DELETED]"
			s.state = orgStateDeleted
			s.errMsg = ""
			return s, loadDeletedOrgsCmd(s.svc)
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

	case orgStateDetail:
		if key.Matches(msg, sectionBackKey) {
			s.state = orgStateList
			s.detailTarget = nil
		}
		return s, nil
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
	if statusRow == "" && len(s.list.Items()) == 0 {
		hint := "  Press n to create your first organization."
		if s.state == orgStateDeleted {
			hint = "  No deleted organizations."
		}
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(width).
			Render(hint)
	}
	if statusRow != "" {
		listView = lipgloss.JoinVertical(lipgloss.Left, listView, statusRow)
	}

	switch s.state {
	case orgStateDeleting:
		return s.renderDeleteConfirm(listView, width, height)
	case orgStateHardDeleting:
		return s.renderHardDeleteConfirm(listView, width)
	case orgStateCreating, orgStateEditing:
		return s.renderFormOverlay(width, height)
	case orgStateDetail:
		return s.renderOrgDetail(width, height)
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

func (s *orgSection) renderHardDeleteConfirm(listView string, width int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(orgItem); ok {
		name = selected.org.Name
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Hard-delete %q? This is permanent and cannot be undone. [y] yes  [n/bksp] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
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
		Render(fmt.Sprintf("  Delete %q? [y] yes  [n/bksp] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
}

func (s *orgSection) renderOrgDetail(width, height int) string {
	o := s.detailTarget

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	opt := func(v string) string {
		if v == "" {
			return "—"
		}
		return v
	}
	row := func(label, value string) string {
		return fmt.Sprintf("  %-18s%-22s", label, value)
	}
	divider := func(label string) string {
		return sectionStyle.Render("── " + label + " " + strings.Repeat("─", 20))
	}

	lines := []string{
		titleStyle.Render(o.Name),
		lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Render("ID: " + o.ID.String()),
		"",
		divider("General"),
		row("Legal Form", string(o.LegalForm)),
		row("Activity", opt(o.Activity)),
		row("Address", opt(o.Address)),
		"",
		divider("Moroccan IDs"),
		row("ICE", opt(o.ICENum)),
		row("IF", opt(o.IFNum)),
		row("RC", opt(o.RCNum)),
		row("CNSS", opt(o.CNSSNum)),
		"",
		divider("Banking"),
		row("Bank RIB", opt(o.BankRIB)),
		"",
		divider("Metadata"),
		row("Created", o.CreatedAt.Format("2006-01-02 15:04")),
		row("Updated", o.UpdatedAt.Format("2006-01-02 15:04")),
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
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
		return "Something went wrong — please try again"
	}
}
