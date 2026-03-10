package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// compState is the internal state machine for the compensation section.
type compState int

const (
	compStateList     compState = iota
	compStateCreating           // create form open
	compStateDeleting           // delete confirmation open
)

// compForm holds the single BaseSalary input for the create overlay.
type compForm struct {
	input textinput.Model
}

func newCompForm() compForm {
	t := textinput.New()
	t.Placeholder = "e.g. 3422.00"
	t.Width = 20
	t.CharLimit = 12
	t.Focus() //nolint:errcheck
	return compForm{input: t}
}

func (f compForm) update(msg tea.KeyMsg) (compForm, formResult, tea.Cmd) {
	switch {
	case key.Matches(msg, formKeys.Cancel):
		return f, formCancel, nil
	case key.Matches(msg, formKeys.Submit):
		return f, formSubmit, nil
	default:
		var cmd tea.Cmd
		f.input, cmd = f.input.Update(msg)
		return f, formContinue, cmd
	}
}

// toDomain parses the salary input into a domain.EmployeeCompensationPackage.
func (f compForm) toDomain() (*domain.EmployeeCompensationPackage, error) {
	raw := strings.TrimSpace(f.input.Value())
	if raw == "" {
		return nil, errors.New("Base salary is required")
	}
	val, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, errors.New("Base salary must be a number (e.g. 3422.00)")
	}
	salary, err := money.FromMAD(val)
	if err != nil {
		return nil, fmt.Errorf("Invalid salary amount: %w", err)
	}
	return &domain.EmployeeCompensationPackage{
		Currency:   money.MAD,
		BaseSalary: salary,
	}, nil
}

func (f compForm) view() string {
	labelStyle := lipgloss.NewStyle().
		Width(14).
		Foreground(lipgloss.Color("205")).
		Bold(true)
	staticStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	rows := []string{
		lipgloss.JoinHorizontal(lipgloss.Center,
			labelStyle.Render("Base Salary *"), f.input.View()),
		lipgloss.JoinHorizontal(lipgloss.Center,
			lipgloss.NewStyle().Width(14).Foreground(lipgloss.Color("245")).Render("Currency"),
			staticStyle.Render("MAD  (only supported currency)")),
	}
	return strings.Join(rows, "\n")
}

// ---------------------------------------------------------------------------
// compSection
// ---------------------------------------------------------------------------

type compSection struct {
	svc             *application.CompensationPackageService
	list            list.Model
	state           compState
	form            compForm
	pendingDeleteID uuid.UUID
	errMsg          string
	width, height   int
}

func newCompSection(svc *application.CompensationPackageService) *compSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Compensation Packages"
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	return &compSection{svc: svc, list: l}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *compSection) Init() tea.Cmd {
	return loadCompsCmd(s.svc)
}

func (s *compSection) IsOverlay() bool {
	return s.state == compStateCreating || s.state == compStateDeleting
}

func (s *compSection) ShortHelp() []key.Binding {
	switch s.state {
	case compStateCreating:
		return []key.Binding{formKeys.Submit, formKeys.Cancel}
	case compStateDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	default:
		return []key.Binding{mainKeys.New, mainKeys.Delete}
	}
}

func (s *compSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		return s, nil

	case compsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "load error: " + msg.err.Error()
			return s, nil
		}
		items := make([]list.Item, len(msg.pkgs))
		for i, p := range msg.pkgs {
			items[i] = compItem{pkg: p}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case saveCompDoneMsg:
		s.state = compStateList
		if msg.err != nil {
			s.errMsg = userFriendlyCompError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadCompsCmd(s.svc)

	case deleteCompDoneMsg:
		s.state = compStateList
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = userFriendlyCompError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadCompsCmd(s.svc)

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	if s.state == compStateList {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *compSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case compStateList:
		switch {
		case key.Matches(msg, mainKeys.New):
			s.form = newCompForm()
			s.state = compStateCreating
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Delete):
			selected, ok := s.list.SelectedItem().(compItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.pkg.ID
			s.state = compStateDeleting
			return s, nil
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case compStateCreating:
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			pkg, err := s.form.toDomain()
			if err != nil {
				s.errMsg = err.Error()
				return s, nil
			}
			s.state = compStateList
			s.errMsg = ""
			return s, createCompCmd(s.svc, pkg)
		case formCancel:
			s.state = compStateList
			s.errMsg = ""
			return s, nil
		default:
			return s, cmd
		}

	case compStateDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.state = compStateList
			s.pendingDeleteID = uuid.Nil
			return s, deleteCompCmd(s.svc, id)
		case key.Matches(msg, confirmKeys.No):
			s.state = compStateList
			s.pendingDeleteID = uuid.Nil
			return s, nil
		}
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *compSection) View(width, height int) string {
	listView := s.list.View()

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
	case compStateDeleting:
		return s.renderDeleteConfirm(listView, width)
	case compStateCreating:
		return s.renderFormOverlay(width, height)
	}
	return listView
}

func (s *compSection) renderFormOverlay(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1).
		Foreground(lipgloss.Color("205"))

	errorLine := ""
	if s.errMsg != "" {
		errorLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render("  "+s.errMsg)
	}

	inner := titleStyle.Render("New Compensation Package") + "\n" + s.form.view() + errorLine

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(48).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

func (s *compSection) renderDeleteConfirm(listView string, width int) string {
	salary := ""
	if selected, ok := s.list.SelectedItem().(compItem); ok {
		salary = selected.Title()
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Delete package %q? [y] yes  [n/esc] cancel", salary))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
}

func (s *compSection) listHeight() int {
	if s.height <= 1 {
		return s.height
	}
	return s.height - 1
}

func userFriendlyCompError(err error) string {
	switch {
	case errors.Is(err, application.ErrCompensationPackageInUse):
		return "Package is in use by employees or payroll — reassign them first"
	case errors.Is(err, application.ErrCompensationPackageNotFound):
		return "Package not found"
	default:
		return err.Error()
	}
}
