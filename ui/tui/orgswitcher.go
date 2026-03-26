package tui

import (
	"context"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
)

// orgsForSwitcherLoadedMsg carries the org list for the switch-org overlay.
type orgsForSwitcherLoadedMsg struct {
	orgs []*domain.Organization
}

// orgSwitcher is the root-level switch-org overlay.
// It is owned by the root Model (not a section) and rendered over the full terminal.
type orgSwitcher struct {
	orgs     []*domain.Organization
	cursor   int
	loading  bool
	activeID uuid.UUID
	cfg      *config.Config
}

func newOrgSwitcher(cfg *config.Config, activeID uuid.UUID) orgSwitcher {
	return orgSwitcher{loading: true, activeID: activeID, cfg: cfg}
}

// loadOrgsForSwitcherCmd fetches all active organizations for the switcher overlay.
func loadOrgsForSwitcherCmd(app *App) tea.Cmd {
	return func() tea.Msg {
		orgs, err := app.OrganizationService.ListOrganizations(context.Background())
		if err != nil {
			return orgsForSwitcherLoadedMsg{}
		}
		return orgsForSwitcherLoadedMsg{orgs: orgs}
	}
}

// loaded populates the switcher after the async load completes.
func (s *orgSwitcher) loaded(msg orgsForSwitcherLoadedMsg) {
	s.orgs = msg.orgs
	s.loading = false
	for i, org := range s.orgs {
		if org.ID == s.activeID {
			s.cursor = i
			return
		}
	}
}

// update handles key events while the switcher is open.
// Returns (newState, cmd, done). Root model closes the switcher when done=true.
func (s orgSwitcher) update(msg tea.KeyMsg) (orgSwitcher, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, sidebarKeys.Up):
		if s.cursor > 0 {
			s.cursor--
		}
	case key.Matches(msg, sidebarKeys.Down):
		if s.cursor < len(s.orgs)-1 {
			s.cursor++
		}
	case key.Matches(msg, formKeys.Submit):
		if len(s.orgs) > 0 {
			return s, setActiveOrgCmd(s.cfg, s.orgs[s.cursor]), true
		}
		return s, nil, true
	case key.Matches(msg, formKeys.Cancel):
		return s, nil, true
	}
	return s, nil, false
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

var (
	switcherTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("205")).
				MarginBottom(1)

	switcherActiveStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205"))

	switcherCursorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("205")).
				Bold(true)

	switcherDimStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	switcherBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("205")).
				Padding(1, 2).
				Width(44)
)

func (s orgSwitcher) view(w, h int) string {
	var sb strings.Builder

	sb.WriteString(switcherTitleStyle.Render("Switch Organization"))
	sb.WriteString("\n")

	switch {
	case s.loading:
		sb.WriteString(switcherDimStyle.Render("Loading…"))
	case len(s.orgs) == 0:
		sb.WriteString(switcherDimStyle.Render("No organizations. Press n in the\nOrganizations section to create one.\n"))
	default:
		for i, org := range s.orgs {
			cursor := "  "
			if i == s.cursor {
				cursor = switcherCursorStyle.Render("> ")
			}
			name := org.Name
			if org.ID == s.activeID {
				name = switcherActiveStyle.Render("• " + name)
			} else {
				name = "  " + name
			}
			if i == s.cursor {
				name = lipgloss.NewStyle().Bold(true).Render(name)
			}
			sb.WriteString(cursor + name + "\n")
		}
	}

	sb.WriteString("\n")
	sb.WriteString(switcherDimStyle.Render("enter") + " select  · " + switcherDimStyle.Render("esc") + " cancel")

	box := switcherBoxStyle.Render(sb.String())

	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}
