package tui

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

const (
	simFieldSalary     = 0
	simFieldHireDate   = 1
	simFieldMarital    = 2
	simFieldDependents = 3
	simFieldChildren   = 4
	simFieldCount      = 5
)

// Key bindings specific to the simulator section.
// Navigation uses ↑/↓ because tab is reserved for pane switching (IsOverlay = false).
var (
	simRunKey = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "run"),
	)
	simFieldUpKey = key.NewBinding(
		key.WithKeys("up"),
	)
	simFieldDownKey = key.NewBinding(
		key.WithKeys("down"),
	)
	simCycleKey = key.NewBinding(
		key.WithKeys("left", "right", " "),
	)
)

var maritalStatusOptions = []domain.MaritalStatusEnum{
	domain.MaritalStatusSingle,
	domain.MaritalStatusMarried,
	domain.MaritalStatusSeparated,
	domain.MaritalStatusDivorced,
	domain.MaritalStatusWidowed,
}

type simForm struct {
	salaryInput     textinput.Model
	hireDateInput   textinput.Model
	dependentsInput textinput.Model
	childrenInput   textinput.Model
	maritalIdx      int
	focusIdx        int
}

func newSimForm() simForm {
	salary := textinput.New()
	salary.Placeholder = "e.g. 5000.00"
	salary.Width = 20
	salary.CharLimit = 12
	salary.Focus()

	hireDate := textinput.New()
	hireDate.Placeholder = "YYYY-MM-DD"
	hireDate.Width = 20
	hireDate.CharLimit = 10

	dependents := textinput.New()
	dependents.Placeholder = "0"
	dependents.Width = 6
	dependents.CharLimit = 4

	children := textinput.New()
	children.Placeholder = "0"
	children.Width = 6
	children.CharLimit = 4

	return simForm{
		salaryInput:     salary,
		hireDateInput:   hireDate,
		dependentsInput: dependents,
		childrenInput:   children,
	}
}

// advanceFocus blurs the current field, moves focusIdx by delta, and focuses the new field.
func (f simForm) advanceFocus(delta int) simForm {
	switch f.focusIdx {
	case simFieldSalary:
		f.salaryInput.Blur()
	case simFieldHireDate:
		f.hireDateInput.Blur()
	case simFieldDependents:
		f.dependentsInput.Blur()
	case simFieldChildren:
		f.childrenInput.Blur()
	}
	f.focusIdx = (f.focusIdx + delta + simFieldCount) % simFieldCount
	switch f.focusIdx {
	case simFieldSalary:
		f.salaryInput.Focus()
	case simFieldHireDate:
		f.hireDateInput.Focus()
	case simFieldDependents:
		f.dependentsInput.Focus()
	case simFieldChildren:
		f.childrenInput.Focus()
	}
	return f
}

// update processes a key message and returns the updated form, a result signal, and any cmd.
// Navigation uses ↑/↓ (tab is owned by the root model when IsOverlay = false).
func (f simForm) update(msg tea.KeyMsg) (simForm, formResult, tea.Cmd) {
	switch {
	case key.Matches(msg, simRunKey):
		return f, formSubmit, nil
	case key.Matches(msg, simFieldDownKey):
		return f.advanceFocus(1), formContinue, nil
	case key.Matches(msg, simFieldUpKey):
		return f.advanceFocus(-1), formContinue, nil
	}

	if f.focusIdx == simFieldMarital {
		switch msg.String() {
		case "left", "h":
			f.maritalIdx = (f.maritalIdx - 1 + len(maritalStatusOptions)) % len(maritalStatusOptions)
			return f, formContinue, nil
		case "right", "l", " ":
			f.maritalIdx = (f.maritalIdx + 1) % len(maritalStatusOptions)
			return f, formContinue, nil
		}
		return f, formContinue, nil
	}

	var cmd tea.Cmd
	switch f.focusIdx {
	case simFieldSalary:
		f.salaryInput, cmd = f.salaryInput.Update(msg)
	case simFieldHireDate:
		f.hireDateInput, cmd = f.hireDateInput.Update(msg)
	case simFieldDependents:
		f.dependentsInput, cmd = f.dependentsInput.Update(msg)
	case simFieldChildren:
		f.childrenInput, cmd = f.childrenInput.Update(msg)
	}
	return f, formContinue, cmd
}

// toDomain validates form inputs and returns synthetic (never-stored) domain objects for simulation.
func (f simForm) toDomain() (*domain.Employee, *domain.EmployeeCompensationPackage, error) {
	rawSalary := strings.TrimSpace(f.salaryInput.Value())
	if rawSalary == "" {
		return nil, nil, errors.New("Base salary is required")
	}
	salaryVal, err := strconv.ParseFloat(rawSalary, 64)
	if err != nil {
		return nil, nil, errors.New("Base salary must be a number (e.g. 5000.00)")
	}
	salary, err := money.FromMAD(salaryVal)
	if err != nil {
		return nil, nil, errors.New("Base salary must be a positive value")
	}

	rawDate := strings.TrimSpace(f.hireDateInput.Value())
	if rawDate == "" {
		return nil, nil, errors.New("Hire date is required")
	}
	hireDate, err := time.Parse("2006-01-02", rawDate)
	if err != nil {
		return nil, nil, errors.New("Hire date must be in YYYY-MM-DD format")
	}
	if hireDate.After(time.Now()) {
		return nil, nil, errors.New("Hire date cannot be in the future")
	}

	deps := 0
	if raw := strings.TrimSpace(f.dependentsInput.Value()); raw != "" {
		n, parseErr := strconv.Atoi(raw)
		if parseErr != nil || n < 0 {
			return nil, nil, errors.New("Dependents must be a non-negative integer")
		}
		deps = n
	}

	kids := 0
	if raw := strings.TrimSpace(f.childrenInput.Value()); raw != "" {
		n, parseErr := strconv.Atoi(raw)
		if parseErr != nil || n < 0 {
			return nil, nil, errors.New("Children must be a non-negative integer")
		}
		kids = n
	}

	emp := &domain.Employee{
		ID:            uuid.New(),
		HireDate:      hireDate,
		MaritalStatus: maritalStatusOptions[f.maritalIdx],
		NumDependents: deps,
		NumChildren:   kids,
	}
	pkg := &domain.EmployeeCompensationPackage{
		ID:         uuid.New(),
		BaseSalary: salary,
		Currency:   money.MAD,
	}

	return emp, pkg, nil
}

func (f simForm) view() string {
	labelStyle := lipgloss.NewStyle().
		Width(16).
		Foreground(lipgloss.Color("245"))
	activeLabelStyle := lipgloss.NewStyle().
		Width(16).
		Foreground(lipgloss.Color("205")).
		Bold(true)

	label := func(idx int, text string) string {
		if f.focusIdx == idx {
			return activeLabelStyle.Render(text)
		}
		return labelStyle.Render(text)
	}

	maritalVal := string(maritalStatusOptions[f.maritalIdx])
	if f.focusIdx == simFieldMarital {
		maritalVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Render("‹ " + maritalVal + " ›")
	} else {
		maritalVal = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Render("  " + maritalVal + "  ")
	}

	rows := []string{
		lipgloss.JoinHorizontal(lipgloss.Center,
			label(simFieldSalary, "Base Salary *"), f.salaryInput.View()),
		lipgloss.JoinHorizontal(lipgloss.Center,
			label(simFieldHireDate, "Hire Date *"), f.hireDateInput.View()),
		lipgloss.JoinHorizontal(lipgloss.Center,
			label(simFieldMarital, "Marital Status"), maritalVal),
		lipgloss.JoinHorizontal(lipgloss.Center,
			label(simFieldDependents, "Dependents"), f.dependentsInput.View()),
		lipgloss.JoinHorizontal(lipgloss.Center,
			label(simFieldChildren, "Children"), f.childrenInput.View()),
	}
	return strings.Join(rows, "\n")
}
