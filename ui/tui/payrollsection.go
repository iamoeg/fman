package tui

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ---------------------------------------------------------------------------
// State machine
// ---------------------------------------------------------------------------

type payrollState int

const (
	payrollStateList         payrollState = iota
	payrollStateCreating                  // new period form overlay
	payrollStateDeleting                  // delete confirm inline
	payrollStateResults                   // drilled into results for selected period
	payrollStateResultDetail              // drilled into a single result's full breakdown
)

// ---------------------------------------------------------------------------
// Payroll-specific key bindings
// ---------------------------------------------------------------------------

type payrollKeyMap struct {
	Generate    key.Binding
	ViewResults key.Binding
	Finalize    key.Binding
	Unfinalize  key.Binding
}

var payrollKeys = payrollKeyMap{
	Generate: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "generate"),
	),
	ViewResults: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "view results"),
	),
	Finalize: key.NewBinding(
		key.WithKeys("f"),
		key.WithHelp("f", "finalize"),
	),
	Unfinalize: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unfinalize"),
	),
}

// ---------------------------------------------------------------------------
// payrollForm — Year + Month fields
// ---------------------------------------------------------------------------

const (
	payrollFormYear  = 0
	payrollFormMonth = 1
	payrollFormCount = 2
)

type payrollForm struct {
	inputs   [payrollFormCount]textinput.Model
	focusIdx int
}

func newPayrollForm() payrollForm {
	year := textinput.New()
	year.Placeholder = fmt.Sprintf("%d", time.Now().Year())
	year.CharLimit = 4
	year.Width = 10
	year.Focus() //nolint:errcheck

	month := textinput.New()
	month.Placeholder = fmt.Sprintf("%d", int(time.Now().Month()))
	month.CharLimit = 2
	month.Width = 10

	return payrollForm{
		inputs: [payrollFormCount]textinput.Model{year, month},
	}
}

func (f payrollForm) advanceFocus(delta int) payrollForm {
	f.inputs[f.focusIdx].Blur()
	f.focusIdx = (f.focusIdx + delta + payrollFormCount) % payrollFormCount
	f.inputs[f.focusIdx].Focus() //nolint:errcheck
	return f
}

func (f payrollForm) update(msg tea.KeyMsg) (payrollForm, formResult, tea.Cmd) {
	switch {
	case key.Matches(msg, formKeys.Cancel):
		return f, formCancel, nil
	case key.Matches(msg, formKeys.Submit):
		return f, formSubmit, nil
	case key.Matches(msg, formKeys.NextField):
		return f.advanceFocus(1), formContinue, nil
	case key.Matches(msg, formKeys.PrevField):
		return f.advanceFocus(-1), formContinue, nil
	}
	var cmd tea.Cmd
	f.inputs[f.focusIdx], cmd = f.inputs[f.focusIdx].Update(msg)
	return f, formContinue, cmd
}

func (f payrollForm) toDomain(orgID uuid.UUID) (*domain.PayrollPeriod, error) {
	yearStr := strings.TrimSpace(f.inputs[payrollFormYear].Value())
	if yearStr == "" {
		return nil, errors.New("Year is required")
	}
	year, err := strconv.Atoi(yearStr)
	if err != nil || year < 2020 || year > 2050 {
		return nil, errors.New("Year must be between 2020 and 2050")
	}

	monthStr := strings.TrimSpace(f.inputs[payrollFormMonth].Value())
	if monthStr == "" {
		return nil, errors.New("Month is required")
	}
	month, err := strconv.Atoi(monthStr)
	if err != nil || month < 1 || month > 12 {
		return nil, errors.New("Month must be between 1 and 12")
	}

	return &domain.PayrollPeriod{
		OrgID: orgID,
		Year:  year,
		Month: month,
	}, nil
}

func (f payrollForm) view() string {
	labelStyle := lipgloss.NewStyle().Width(10).Foreground(lipgloss.Color("205")).Bold(true)
	labels := [payrollFormCount]string{"Year *", "Month *"}

	var rows []string
	for i := 0; i < payrollFormCount; i++ {
		row := lipgloss.JoinHorizontal(lipgloss.Center,
			labelStyle.Render(labels[i]),
			f.inputs[i].View(),
		)
		rows = append(rows, row)
	}
	return strings.Join(rows, "\n")
}

// ---------------------------------------------------------------------------
// payrollSection
// ---------------------------------------------------------------------------

type payrollSection struct {
	payrollSvc         *application.PayrollService
	empSvc             *application.EmployeeService
	orgID              uuid.UUID
	list               list.Model
	resultList         list.Model
	state              payrollState
	form               payrollForm
	selectedPeriod     *domain.PayrollPeriod
	selectedResult     *domain.PayrollResult
	selectedResultName string
	pendingDeleteID    uuid.UUID
	errMsg             string
	statusMsg          string
	width              int
	height             int
}

func newPayrollSection(
	payrollSvc *application.PayrollService,
	empSvc *application.EmployeeService,
	orgID uuid.UUID,
) *payrollSection {
	delegate := list.NewDefaultDelegate()

	l := list.New(nil, delegate, 0, 0)
	l.Title = "Payroll Periods"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.Styles.NoItems = l.Styles.NoItems.PaddingLeft(2)

	rl := list.New(nil, delegate, 0, 0)
	rl.SetShowHelp(false)
	rl.SetShowStatusBar(false)
	rl.SetFilteringEnabled(false)
	rl.Styles.NoItems = rl.Styles.NoItems.PaddingLeft(2)

	return &payrollSection{
		payrollSvc: payrollSvc,
		empSvc:     empSvc,
		orgID:      orgID,
		list:       l,
		resultList: rl,
	}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *payrollSection) Init() tea.Cmd {
	return loadPeriodsCmd(s.payrollSvc, s.orgID)
}

func (s *payrollSection) IsOverlay() bool {
	return s.state == payrollStateCreating || s.state == payrollStateDeleting ||
		s.state == payrollStateResultDetail ||
		s.list.FilterState() == list.Filtering
}

func (s *payrollSection) ShortHelp() []key.Binding {
	switch s.state {
	case payrollStateCreating:
		return []key.Binding{formKeys.Submit, formKeys.Cancel}
	case payrollStateDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	case payrollStateResults:
		return []key.Binding{payrollKeys.ViewResults, sectionBackKey}
	case payrollStateResultDetail:
		return []key.Binding{sectionBackKey}
	default:
		return []key.Binding{
			mainKeys.New,
			payrollKeys.Generate,
			payrollKeys.ViewResults,
			payrollKeys.Finalize,
			payrollKeys.Unfinalize,
			mainKeys.Delete,
			mainKeys.Filter,
		}
	}
}

func (s *payrollSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		s.resultList.SetSize(s.width, s.listHeight())
		return s, nil

	case activeOrgLoadedMsg:
		s.orgID = msg.orgID
		s.state = payrollStateList
		s.selectedPeriod = nil
		s.errMsg = ""
		s.statusMsg = ""
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case periodsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load payroll data — try again"
			return s, nil
		}
		items := make([]list.Item, len(msg.periods))
		for i, p := range msg.periods {
			items[i] = periodItem{period: p, count: msg.counts[p.ID]}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case resultsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load payroll data — try again"
			s.state = payrollStateList
			return s, nil
		}
		items := make([]list.Item, len(msg.results))
		for i, r := range msg.results {
			items[i] = resultItem{result: r, empName: msg.employees[r.EmployeeID]}
		}
		cmd := s.resultList.SetItems(items)
		s.state = payrollStateResults
		s.errMsg = ""
		s.statusMsg = ""
		return s, cmd

	case createPeriodDoneMsg:
		s.state = payrollStateList
		if msg.err != nil {
			s.errMsg = userFriendlyPayrollError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		s.statusMsg = "Period created"
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case generateResultsDoneMsg:
		if msg.err != nil {
			s.statusMsg = ""
			s.errMsg = userFriendlyPayrollError(msg.err)
			return s, nil
		}
		if msg.count == 0 {
			s.statusMsg = ""
			s.errMsg = "No employees in this organization — nothing was generated"
			return s, nil
		}
		s.errMsg = ""
		s.statusMsg = fmt.Sprintf("Generated payroll for %d employee(s)", msg.count)
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case finalizePeriodDoneMsg:
		if msg.err != nil {
			s.errMsg = userFriendlyPayrollError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		s.statusMsg = "Period finalized"
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case unfinalizePeriodDoneMsg:
		if msg.err != nil {
			s.errMsg = userFriendlyPayrollError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		s.statusMsg = "Period unfinalized"
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case deletePeriodDoneMsg:
		s.state = payrollStateList
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = userFriendlyPayrollError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		s.statusMsg = "Period deleted"
		return s, loadPeriodsCmd(s.payrollSvc, s.orgID)

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	// Forward non-key, non-async messages to the active list.
	switch s.state {
	case payrollStateList:
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	case payrollStateResults:
		var cmd tea.Cmd
		s.resultList, cmd = s.resultList.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *payrollSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case payrollStateList:
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}
		switch {
		case key.Matches(msg, mainKeys.New):
			if s.orgID == uuid.Nil {
				s.errMsg = "Select an active organization first"
				return s, nil
			}
			s.form = newPayrollForm()
			s.state = payrollStateCreating
			s.errMsg = ""
			s.statusMsg = ""
			return s, nil

		case key.Matches(msg, payrollKeys.ViewResults):
			selected, ok := s.list.SelectedItem().(periodItem)
			if !ok {
				return s, nil
			}
			s.selectedPeriod = selected.period
			s.resultList.Title = fmt.Sprintf("%s %d — Results",
				time.Month(selected.period.Month).String(), selected.period.Year)
			return s, loadResultsCmd(s.payrollSvc, s.empSvc, selected.period.ID, s.orgID)

		case key.Matches(msg, payrollKeys.Generate):
			selected, ok := s.list.SelectedItem().(periodItem)
			if !ok {
				return s, nil
			}
			if selected.period.Status == domain.PayrollPeriodStatusFinalized {
				s.errMsg = "Cannot regenerate for a finalized period"
				return s, nil
			}
			s.statusMsg = "Generating..."
			s.errMsg = ""
			return s, generateResultsCmd(s.payrollSvc, selected.period.ID)

		case key.Matches(msg, payrollKeys.Finalize):
			selected, ok := s.list.SelectedItem().(periodItem)
			if !ok {
				return s, nil
			}
			s.errMsg = ""
			s.statusMsg = ""
			return s, finalizePeriodCmd(s.payrollSvc, selected.period.ID)

		case key.Matches(msg, payrollKeys.Unfinalize):
			selected, ok := s.list.SelectedItem().(periodItem)
			if !ok {
				return s, nil
			}
			s.errMsg = ""
			s.statusMsg = ""
			return s, unfinalizePeriodCmd(s.payrollSvc, selected.period.ID)

		case key.Matches(msg, mainKeys.Delete):
			selected, ok := s.list.SelectedItem().(periodItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.period.ID
			s.state = payrollStateDeleting
			s.errMsg = ""
			s.statusMsg = ""
			return s, nil
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case payrollStateCreating:
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			period, err := s.form.toDomain(s.orgID)
			if err != nil {
				s.errMsg = err.Error()
				return s, nil
			}
			s.errMsg = ""
			s.state = payrollStateList
			return s, createPeriodCmd(s.payrollSvc, period)
		case formCancel:
			s.state = payrollStateList
			s.errMsg = ""
			return s, nil
		default:
			return s, cmd
		}

	case payrollStateDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.state = payrollStateList
			s.pendingDeleteID = uuid.Nil
			return s, deletePeriodCmd(s.payrollSvc, id)
		case key.Matches(msg, confirmKeys.No):
			s.state = payrollStateList
			s.pendingDeleteID = uuid.Nil
			return s, nil
		}

	case payrollStateResults:
		switch {
		case key.Matches(msg, sectionBackKey):
			s.state = payrollStateList
			s.selectedPeriod = nil
			s.errMsg = ""
			s.statusMsg = ""
			return s, nil
		case key.Matches(msg, payrollKeys.ViewResults):
			if selected, ok := s.resultList.SelectedItem().(resultItem); ok {
				s.selectedResult = selected.result
				s.selectedResultName = selected.empName
				s.state = payrollStateResultDetail
			}
			return s, nil
		}
		var cmd tea.Cmd
		s.resultList, cmd = s.resultList.Update(msg)
		return s, cmd

	case payrollStateResultDetail:
		if key.Matches(msg, sectionBackKey) {
			s.state = payrollStateResults
			s.selectedResult = nil
			s.selectedResultName = ""
			return s, nil
		}
		return s, nil
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *payrollSection) View(width, height int) string {
	var mainView string
	switch s.state {
	case payrollStateResults, payrollStateResultDetail:
		mainView = s.resultList.View()
	default:
		mainView = s.list.View()
	}

	statusRow := ""
	if s.errMsg != "" {
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Width(width).
			Render("  " + s.errMsg)
	} else if s.statusMsg != "" {
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("114")).
			Width(width).
			Render("  " + s.statusMsg)
	}
	if statusRow == "" {
		switch s.state {
		case payrollStateResults:
			if len(s.resultList.Items()) == 0 {
				statusRow = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Width(width).
					Render("  Press g to generate payroll for this period.")
			}
		default:
			if len(s.list.Items()) == 0 {
				hint := "Press n to create a payroll period."
				if s.orgID == uuid.Nil {
					hint = "Select an active organization first."
				}
				statusRow = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Width(width).
					Render("  " + hint)
			}
		}
	}
	if statusRow != "" {
		mainView = lipgloss.JoinVertical(lipgloss.Left, mainView, statusRow)
	}

	switch s.state {
	case payrollStateDeleting:
		return s.renderDeleteConfirm(mainView, width)
	case payrollStateCreating:
		return s.renderFormOverlay(width, height)
	case payrollStateResultDetail:
		return s.renderResultDetail(width, height)
	}
	return mainView
}

func (s *payrollSection) renderFormOverlay(width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).MarginBottom(1).
		Foreground(lipgloss.Color("205"))

	errorLine := ""
	if s.errMsg != "" {
		errorLine = "\n" + lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Render("  "+s.errMsg)
	}

	inner := titleStyle.Render("New Payroll Period") + "\n" + s.form.view() + errorLine

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(40).
		Render(inner)

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}

func (s *payrollSection) renderDeleteConfirm(mainView string, width int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(periodItem); ok {
		name = selected.Title()
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Delete %q? [y] yes  [n/bksp] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, mainView, prompt)
}

func (s *payrollSection) renderResultDetail(width, height int) string {
	return renderPayrollResultDetail(s.selectedResult, s.selectedResultName, s.selectedPeriod, width, height)
}

func (s *payrollSection) listHeight() int {
	if s.height <= 1 {
		return s.height
	}
	return s.height - 1
}

func userFriendlyPayrollError(err error) string {
	switch {
	case errors.Is(err, application.ErrPayrollPeriodExists):
		return "A period for that month already exists"
	case errors.Is(err, application.ErrPayrollPeriodAlreadyFinalized):
		return "Period is already finalized"
	case errors.Is(err, application.ErrPayrollPeriodNotFinalized):
		return "Period is not finalized"
	case errors.Is(err, application.ErrPayrollPeriodEmpty):
		return "Cannot finalize — no results generated yet"
	case errors.Is(err, application.ErrPayrollPeriodNotFound):
		return "Period not found"
	case errors.Is(err, application.ErrPayrollCalculationFailed):
		raw := err.Error()
		switch {
		case strings.Contains(raw, "SMIG") || strings.Contains(raw, "minimum wage"):
			return "An employee's salary is below the legal minimum wage (SMIG)"
		case strings.Contains(raw, "rate table") || strings.Contains(raw, "payroll year"):
			return "No tax rates configured for this payroll year"
		default:
			return "Payroll calculation failed — check employee records"
		}
	default:
		return "Something went wrong — please try again"
	}
}
