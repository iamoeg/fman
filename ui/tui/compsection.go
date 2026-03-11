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
	compStateEditing            // rename form open
	compStateDeleting           // delete confirmation open
)

// compForm holds the Name and BaseSalary inputs for the create overlay,
// or just the Name input when editing (renaming) an existing package.
type compForm struct {
	nameInput     textinput.Model
	salaryInput   textinput.Model
	focused       int
	editing       bool      // true = rename mode (salary is immutable, not shown as input)
	editingID     uuid.UUID // ID of the package being renamed
	salaryDisplay string    // formatted salary for display when editing
}

func newCompForm() compForm {
	name := textinput.New()
	name.Placeholder = "e.g. Standard Developer"
	name.Width = 24
	name.CharLimit = 64
	name.Focus() //nolint:errcheck

	salary := textinput.New()
	salary.Placeholder = "e.g. 3422.00"
	salary.Width = 20
	salary.CharLimit = 12

	return compForm{nameInput: name, salaryInput: salary, focused: 0}
}

func newCompFormFromPkg(pkg *domain.EmployeeCompensationPackage) compForm {
	f := newCompForm()
	f.nameInput.SetValue(pkg.Name)
	f.editing = true
	f.editingID = pkg.ID
	f.salaryDisplay = fmt.Sprintf("%.2f %s", pkg.BaseSalary.ToMAD(), pkg.Currency)
	return f
}

func (f compForm) update(msg tea.KeyMsg) (compForm, formResult, tea.Cmd) {
	switch {
	case key.Matches(msg, formKeys.Cancel):
		return f, formCancel, nil
	case key.Matches(msg, formKeys.Submit):
		return f, formSubmit, nil
	case key.Matches(msg, key.NewBinding(key.WithKeys("tab", "shift+tab"))):
		if f.editing {
			// Name-only form: no second field to cycle to
			return f, formContinue, nil
		}
		// Toggle focus between name and salary
		if f.focused == 0 {
			f.focused = 1
			f.nameInput.Blur()
			f.salaryInput.Focus() //nolint:errcheck
		} else {
			f.focused = 0
			f.salaryInput.Blur()
			f.nameInput.Focus() //nolint:errcheck
		}
		return f, formContinue, nil
	default:
		var cmd tea.Cmd
		if f.focused == 0 {
			f.nameInput, cmd = f.nameInput.Update(msg)
		} else {
			f.salaryInput, cmd = f.salaryInput.Update(msg)
		}
		return f, formContinue, cmd
	}
}

// toDomain parses the form inputs into a domain.EmployeeCompensationPackage.
// The caller is responsible for setting OrgID.
func (f compForm) toDomain(orgID uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
	name := strings.TrimSpace(f.nameInput.Value())
	if name == "" {
		return nil, errors.New("Name is required")
	}

	raw := strings.TrimSpace(f.salaryInput.Value())
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
		OrgID:      orgID,
		Name:       name,
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
	dimLabel := lipgloss.NewStyle().Width(14).Foreground(lipgloss.Color("245"))

	rows := []string{
		lipgloss.JoinHorizontal(lipgloss.Center,
			labelStyle.Render("Name *"), f.nameInput.View()),
	}

	if f.editing {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Center,
				dimLabel.Render("Base Salary"),
				staticStyle.Render(f.salaryDisplay+" (immutable)")),
		)
	} else {
		rows = append(rows,
			lipgloss.JoinHorizontal(lipgloss.Center,
				labelStyle.Render("Base Salary *"), f.salaryInput.View()),
			lipgloss.JoinHorizontal(lipgloss.Center,
				dimLabel.Render("Currency"),
				staticStyle.Render("MAD  (only supported currency)")),
		)
	}
	return strings.Join(rows, "\n")
}

// ---------------------------------------------------------------------------
// compSection
// ---------------------------------------------------------------------------

type compSection struct {
	svc             *application.CompensationPackageService
	orgID           uuid.UUID
	list            list.Model
	state           compState
	form            compForm
	pendingDeleteID uuid.UUID
	errMsg          string
	width, height   int
}

func newCompSection(svc *application.CompensationPackageService, orgID uuid.UUID) *compSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Compensation Packages"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	return &compSection{svc: svc, orgID: orgID, list: l}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *compSection) Init() tea.Cmd {
	return loadCompsCmd(s.svc, s.orgID)
}

func (s *compSection) IsOverlay() bool {
	return s.state == compStateCreating || s.state == compStateEditing || s.state == compStateDeleting
}

func (s *compSection) ShortHelp() []key.Binding {
	switch s.state {
	case compStateCreating, compStateEditing:
		return []key.Binding{formKeys.Submit, formKeys.Cancel}
	case compStateDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	default:
		return []key.Binding{mainKeys.New, mainKeys.Edit, mainKeys.Delete}
	}
}

func (s *compSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		return s, nil

	case activeOrgLoadedMsg:
		s.orgID = msg.orgID
		return s, loadCompsCmd(s.svc, s.orgID)

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
		return s, loadCompsCmd(s.svc, s.orgID)

	case deleteCompDoneMsg:
		s.state = compStateList
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = userFriendlyCompError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadCompsCmd(s.svc, s.orgID)

	case renameCompDoneMsg:
		s.state = compStateList
		if msg.err != nil {
			s.errMsg = userFriendlyCompError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadCompsCmd(s.svc, s.orgID)

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
			if s.orgID == uuid.Nil {
				s.errMsg = "Select an active organization first"
				return s, nil
			}
			s.form = newCompForm()
			s.state = compStateCreating
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Edit):
			selected, ok := s.list.SelectedItem().(compItem)
			if !ok {
				return s, nil
			}
			s.form = newCompFormFromPkg(selected.pkg)
			s.state = compStateEditing
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

	case compStateEditing:
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			name := strings.TrimSpace(s.form.nameInput.Value())
			if name == "" {
				s.errMsg = "Name is required"
				return s, nil
			}
			id := s.form.editingID
			s.state = compStateList
			s.errMsg = ""
			return s, renameCompCmd(s.svc, id, name)
		case formCancel:
			s.state = compStateList
			s.errMsg = ""
			return s, nil
		default:
			return s, cmd
		}

	case compStateCreating:
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			pkg, err := s.form.toDomain(s.orgID)
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
		return s.renderFormOverlay("New Compensation Package", width, height)
	case compStateEditing:
		return s.renderFormOverlay("Edit Package Name", width, height)
	}
	return listView
}

func (s *compSection) renderFormOverlay(title string, width, height int) string {
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
		Width(52).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

func (s *compSection) renderDeleteConfirm(listView string, width int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(compItem); ok {
		name = selected.Title()
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Delete package %q? [y] yes  [n/esc] cancel", name))
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
