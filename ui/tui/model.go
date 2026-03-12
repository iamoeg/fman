package tui

import (
	"context"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

const (
	sidebarWidth = 22
	footerHeight = 1
	headerHeight = 1
)

// focusTarget identifies which pane currently has keyboard focus.
type focusTarget int

const (
	focusSidebar focusTarget = iota
	focusMain
)

// sectionModel is the interface every section must satisfy.
// The root model dispatches to whichever section is active.
type sectionModel interface {
	// Init is called once at program start; return a Cmd to kick off initial
	// data loading (or nil if the section has nothing to load up-front).
	Init() tea.Cmd
	Update(msg tea.Msg) (sectionModel, tea.Cmd)
	View(width, height int) string
	ShortHelp() []key.Binding
	// IsOverlay returns true when the section has a form or modal open.
	// The root model skips ALL global key bindings (q, tab/shift+tab, esc) in
	// that state so the section owns every keystroke — including tab for
	// field navigation and esc to cancel.
	IsOverlay() bool
}

// activeOrgLoadedMsg carries the display name and ID of the active org (or empty on miss).
type activeOrgLoadedMsg struct {
	name  string
	orgID uuid.UUID
}

// loadActiveOrgCmd looks up the org name for the given UUID.
// Sends an empty name if the ID is nil or the org is not found.
func loadActiveOrgCmd(app *App, idStr string) tea.Cmd {
	return func() tea.Msg {
		id, err := uuid.Parse(idStr)
		if err != nil || id == uuid.Nil {
			return activeOrgLoadedMsg{}
		}
		org, err := app.OrganizationService.GetOrganization(context.Background(), id)
		if err != nil {
			return activeOrgLoadedMsg{}
		}
		return activeOrgLoadedMsg{name: org.Name, orgID: id}
	}
}

// Model is the root Bubble Tea model. It owns layout, focus, and routing.
type Model struct {
	app           *App
	width         int
	height        int
	focus         focusTarget
	active        sectionIndex
	sections      [sectionCount]sectionModel
	sidebar       sidebar
	activeOrgName string
}

// NewModel constructs the root model. Call this from cmd/tui/main.go.
func NewModel(app *App) Model {
	m := Model{
		app:     app,
		focus:   focusSidebar,
		active:  sectionOrganizations,
		sidebar: newSidebar(),
	}

	m.sections[sectionOrganizations] = newOrgSection(app.OrganizationService, app.Config)

	var initialOrgID uuid.UUID
	if app.Config != nil && app.Config.DefaultOrgID != "" {
		initialOrgID, _ = uuid.Parse(app.Config.DefaultOrgID)
	}
	m.sections[sectionCompensation] = newCompSection(app.CompensationService, initialOrgID)
	m.sections[sectionEmployees] = newEmpSection(app.EmployeeService, app.CompensationService, app.PayrollService, initialOrgID)
	m.sections[sectionPayroll] = newPayrollSection(app.PayrollService, app.EmployeeService, initialOrgID)
	return m
}

func (m Model) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, sectionCount+1)
	for i := range m.sections {
		if cmd := m.sections[i].Init(); cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	if m.app.Config != nil && m.app.Config.DefaultOrgID != "" {
		cmds = append(cmds, loadActiveOrgCmd(m.app, m.app.Config.DefaultOrgID))
	}
	return tea.Batch(cmds...)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Forward to all sections so their internal components (list, viewport,
		// table, …) can resize even if they haven't been visited yet.
		var cmds []tea.Cmd
		for i := range m.sections {
			next, cmd := m.sections[i].Update(msg)
			m.sections[i] = next
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
		return m, tea.Batch(cmds...)

	case activeOrgLoadedMsg:
		m.activeOrgName = msg.name
		var cmds []tea.Cmd
		nextComp, cmd := m.sections[sectionCompensation].Update(msg)
		m.sections[sectionCompensation] = nextComp
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		nextEmp, cmd2 := m.sections[sectionEmployees].Update(msg)
		m.sections[sectionEmployees] = nextEmp
		if cmd2 != nil {
			cmds = append(cmds, cmd2)
		}
		nextPayroll, cmd3 := m.sections[sectionPayroll].Update(msg)
		m.sections[sectionPayroll] = nextPayroll
		if cmd3 != nil {
			cmds = append(cmds, cmd3)
		}
		return m, tea.Batch(cmds...)

	case periodsLoadedMsg, resultsLoadedMsg, createPeriodDoneMsg, generateResultsDoneMsg, finalizePeriodDoneMsg, unfinalizePeriodDoneMsg, deletePeriodDoneMsg:
		next, cmd := m.sections[sectionPayroll].Update(msg)
		m.sections[sectionPayroll] = next
		return m, cmd

	case empsLoadedMsg, saveEmpDoneMsg, deleteEmpDoneMsg, empHistoryLoadedMsg:
		next, cmd := m.sections[sectionEmployees].Update(msg)
		m.sections[sectionEmployees] = next
		return m, cmd

	case compsLoadedMsg, saveCompDoneMsg, deleteCompDoneMsg:
		// Route compensation async responses directly to the comp section,
		// regardless of which section is currently active.
		next, cmd := m.sections[sectionCompensation].Update(msg)
		m.sections[sectionCompensation] = next
		// Also forward compsLoadedMsg to the emp section so its package picker stays fresh.
		if _, ok := msg.(compsLoadedMsg); ok {
			nextEmp, _ := m.sections[sectionEmployees].Update(msg)
			m.sections[sectionEmployees] = nextEmp
		}
		return m, cmd

	case tea.KeyMsg:
		// Global bindings are skipped when the active section has a form or modal
		// open — the section owns every keystroke including tab and esc.
		capturing := m.focus == focusMain && m.sections[m.active].IsOverlay()
		if !capturing {
			switch {
			case key.Matches(msg, globalKeys.Quit):
				return m, tea.Quit

			case key.Matches(msg, globalKeys.SwitchPane):
				if m.focus == focusSidebar {
					m.focus = focusMain
				} else {
					m.focus = focusSidebar
				}
				return m, nil
			}
		}

		// Route to focused pane.
		if m.focus == focusSidebar {
			return m.updateSidebar(msg)
		}
		return m.updateMain(msg)

	default:
		// Route async messages (service responses, etc.) to the active section.
		next, cmd := m.sections[m.active].Update(msg)
		m.sections[m.active] = next
		return m, cmd
	}
}

func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newSidebar, selected := m.sidebar.update(msg)
	m.sidebar = newSidebar
	if selected {
		m.active = m.sidebar.active
		// Switch focus to main pane after selecting a section.
		m.focus = focusMain
	}
	return m, nil
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc to return focus to sidebar — but only when the section is
	// not capturing input, so sections can use Esc internally (e.g. cancel form).
	if km, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(km, mainKeys.Back) && !m.sections[m.active].IsOverlay() {
			m.focus = focusSidebar
			return m, nil
		}
	}

	next, cmd := m.sections[m.active].Update(msg)
	m.sections[m.active] = next
	return m, cmd
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading…"
	}

	mainW := m.mainWidth()
	mainH := m.mainContentHeight()

	// Sidebar
	sidebarView := m.sidebar.view(m.height-footerHeight-headerHeight, m.focus == focusSidebar, m.activeOrgName)

	// Header bar
	headerStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("235")).
		Foreground(lipgloss.Color("252")).
		Bold(true).
		Padding(0, 1)
	header := headerStyle.Render("finmgmt  —  " + sectionLabels[m.active])

	// Main pane
	focusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205"))
	unfocusedBorder := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240"))

	borderStyle := unfocusedBorder
	if m.focus == focusMain {
		borderStyle = focusedBorder
	}

	mainContent := m.sections[m.active].View(mainW, mainH)
	mainPane := borderStyle.
		Width(mainW).
		Height(mainH).
		Render(mainContent)

	// Compose sidebar + main side by side
	body := lipgloss.JoinHorizontal(lipgloss.Top, sidebarView, mainPane)

	// Footer
	var bindings []key.Binding
	overlay := false
	if m.focus == focusSidebar {
		bindings = m.sidebar.shortHelp()
	} else {
		overlay = m.sections[m.active].IsOverlay()
		bindings = m.sections[m.active].ShortHelp()
	}
	footer := renderFooter(m.width, bindings, overlay)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

// mainWidth returns the width available for the main content pane (inside its border).
func (m Model) mainWidth() int {
	w := m.width - sidebarWidth - 2 // -2 for border chars, matching sidebar
	if w < 0 {
		return 0
	}
	return w
}

// mainContentHeight returns the height for main pane content (inside its border).
func (m Model) mainContentHeight() int {
	h := m.height - footerHeight - headerHeight - 2 // -2 for border top+bottom
	if h < 0 {
		return 0
	}
	return h
}

