package tui

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
	Update(msg tea.Msg) (sectionModel, tea.Cmd)
	View(width, height int) string
	ShortHelp() []key.Binding
}

// Model is the root Bubble Tea model. It owns layout, focus, and routing.
type Model struct {
	app      *App
	width    int
	height   int
	focus    focusTarget
	active   sectionIndex
	sections [sectionCount]sectionModel
	sidebar  sidebar
}

// NewModel constructs the root model. Call this from cmd/tui/main.go.
func NewModel(app *App) Model {
	m := Model{
		app:     app,
		focus:   focusSidebar,
		active:  sectionOrganizations,
		sidebar: newSidebar(),
	}
	m.sidebar.focused = true

	// Install placeholder sections for every slot.
	// Steps 2-4 replace these one by one with real implementations.
	for i := sectionIndex(0); i < sectionCount; i++ {
		m.sections[i] = newPlaceholderSection(sectionLabels[i])
	}

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		// Global bindings checked first.
		switch {
		case key.Matches(msg, globalKeys.Quit):
			return m, tea.Quit

		case key.Matches(msg, globalKeys.SwitchPane):
			if m.focus == focusSidebar {
				m.focus = focusMain
				m.sidebar.focused = false
			} else {
				m.focus = focusSidebar
				m.sidebar.focused = true
			}
			return m, nil
		}

		// Route to focused pane.
		if m.focus == focusSidebar {
			return m.updateSidebar(msg)
		}
		return m.updateMain(msg)
	}

	return m, nil
}

func (m Model) updateSidebar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	newSidebar, selected := m.sidebar.update(msg)
	m.sidebar = newSidebar
	if selected {
		m.active = m.sidebar.active
		// Switch focus to main pane after selecting a section.
		m.focus = focusMain
		m.sidebar.focused = false
	}
	return m, nil
}

func (m Model) updateMain(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Check for Esc to return focus to sidebar.
	if km, ok := msg.(tea.KeyMsg); ok {
		if key.Matches(km, mainKeys.Back) {
			m.focus = focusSidebar
			m.sidebar.focused = true
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
	sidebarView := m.sidebar.view(m.height - footerHeight - headerHeight)

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
	if m.focus == focusSidebar {
		bindings = m.sidebar.shortHelp()
	} else {
		bindings = m.sections[m.active].ShortHelp()
	}
	footer := renderFooter(m.width, bindings)

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

// ---------------------------------------------------------------------------
// Placeholder section — used until Steps 2-4 replace each section.
// ---------------------------------------------------------------------------

type placeholderSection struct {
	label string
}

func newPlaceholderSection(label string) sectionModel {
	return &placeholderSection{label: label}
}

func (p *placeholderSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	return p, nil
}

func (p *placeholderSection) View(width, height int) string {
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true).
		Padding(1, 2)
	return style.Render("(" + p.label + " — coming soon)")
}

func (p *placeholderSection) ShortHelp() []key.Binding {
	return []key.Binding{mainKeys.New, mainKeys.Edit, mainKeys.Delete}
}
