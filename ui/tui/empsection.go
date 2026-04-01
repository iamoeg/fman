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

	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/internal/domain"
)

// empState is the internal state machine for the employees section.
type empState int

const (
	empStateList          empState = iota
	empStateCreating               // create form open
	empStateEditing                // edit form open
	empStateDeleting               // delete confirmation open
	empStateDetail                 // read-only detail overlay
	empStateHistory                // full-screen payroll history list
	empStateHistoryDetail          // result detail overlay (shared renderer)
	empStateDeleted                // browsing soft-deleted employees
	empStateHardDeleting           // hard-delete confirmation overlay
)

var empDetailKey = key.NewBinding(
	key.WithKeys("enter"),
	key.WithHelp("enter", "view details"),
)

var empHistoryKey = key.NewBinding(
	key.WithKeys("h"),
	key.WithHelp("h", "payroll history"),
)

// ---------------------------------------------------------------------------
// Form field indices
// ---------------------------------------------------------------------------

const (
	empFieldFullName      = iota // 0 — required, text
	empFieldCINNum               // 1 — required, text
	empFieldCNSSNum              // 2 — optional, text
	empFieldBirthDate            // 3 — required, text (YYYY-MM-DD)
	empFieldGender               // 4 — required, cycle (MALE/FEMALE)
	empFieldHireDate             // 5 — required, text (YYYY-MM-DD)
	empFieldPosition             // 6 — required, text
	empFieldCompPkg              // 7 — required, picker (cycles through packages)
	empFieldDisplayName          // 8 — optional, text
	empFieldMaritalStatus        // 9 — required, cycle
	empFieldNumDependents        // 10 — optional, int
	empFieldNumChildren          // 11 — optional, int
	empFieldPhoneNumber          // 12 — optional, text
	empFieldEmailAddress         // 13 — optional, text
	empFieldAddress              // 14 — optional, text
	empFieldBankRIB              // 15 — optional, text
	empFieldCount                // 16
)

const visibleFields = 10

var (
	genderValues = []domain.GenderEnum{
		domain.GenderMale,
		domain.GenderFemale,
	}
	maritalValues = []domain.MaritalStatusEnum{
		domain.MaritalStatusSingle,
		domain.MaritalStatusMarried,
		domain.MaritalStatusSeparated,
		domain.MaritalStatusDivorced,
		domain.MaritalStatusWidowed,
	}
	empFieldMeta = [empFieldCount]struct {
		label    string
		required bool
	}{
		empFieldFullName:      {"Full Name", true},
		empFieldCINNum:        {"CIN Number", true},
		empFieldCNSSNum:       {"CNSS Number", false},
		empFieldBirthDate:     {"Birth Date", true},
		empFieldGender:        {"Gender", true},
		empFieldHireDate:      {"Hire Date", true},
		empFieldPosition:      {"Position", true},
		empFieldCompPkg:       {"Comp Package", true},
		empFieldDisplayName:   {"Display Name", false},
		empFieldMaritalStatus: {"Marital Status", true},
		empFieldNumDependents: {"Dependents", false},
		empFieldNumChildren:   {"Children", false},
		empFieldPhoneNumber:   {"Phone", false},
		empFieldEmailAddress:  {"Email", false},
		empFieldAddress:       {"Address", false},
		empFieldBankRIB:       {"Bank RIB", false},
	}
)

func isCycleOrPickerField(idx int) bool {
	return idx == empFieldGender || idx == empFieldMaritalStatus || idx == empFieldCompPkg
}

// ---------------------------------------------------------------------------
// empForm
// ---------------------------------------------------------------------------

type empForm struct {
	inputs     [empFieldCount]textinput.Model
	genderIdx  int
	maritalIdx int
	pkgIdx     int
	pkgs       []*domain.EmployeeCompensationPackage
	focused    int
	viewOffset int
}

func newEmpForm(pkgs []*domain.EmployeeCompensationPackage) empForm {
	placeholders := [empFieldCount]string{
		empFieldFullName:      "e.g. Ahmed Ali",
		empFieldCINNum:        "e.g. AB123456",
		empFieldCNSSNum:       "(optional)",
		empFieldBirthDate:     "YYYY-MM-DD",
		empFieldGender:        "", // cycle field — no placeholder
		empFieldHireDate:      "YYYY-MM-DD",
		empFieldPosition:      "e.g. Software Engineer",
		empFieldCompPkg:       "", // picker field — no placeholder
		empFieldDisplayName:   "(optional)",
		empFieldMaritalStatus: "", // cycle field — no placeholder
		empFieldNumDependents: "0",
		empFieldNumChildren:   "0",
		empFieldPhoneNumber:   "(optional)",
		empFieldEmailAddress:  "(optional)",
		empFieldAddress:       "(optional)",
		empFieldBankRIB:       "(optional)",
	}
	charLimits := [empFieldCount]int{
		empFieldFullName:      128,
		empFieldCINNum:        16,
		empFieldCNSSNum:       16,
		empFieldBirthDate:     10,
		empFieldGender:        0, // unused
		empFieldHireDate:      10,
		empFieldPosition:      128,
		empFieldCompPkg:       0, // unused
		empFieldDisplayName:   64,
		empFieldMaritalStatus: 0, // unused
		empFieldNumDependents: 4,
		empFieldNumChildren:   4,
		empFieldPhoneNumber:   32,
		empFieldEmailAddress:  128,
		empFieldAddress:       256,
		empFieldBankRIB:       32,
	}

	var inputs [empFieldCount]textinput.Model
	for i := range inputs {
		t := textinput.New()
		t.Placeholder = placeholders[i]
		t.Width = 26
		if charLimits[i] > 0 {
			t.CharLimit = charLimits[i]
		}
		inputs[i] = t
	}
	inputs[empFieldFullName].Focus()

	return empForm{
		inputs: inputs,
		pkgs:   pkgs,
	}
}

func newEmpFormFromEmployee(pkgs []*domain.EmployeeCompensationPackage, emp *domain.Employee) empForm {
	f := newEmpForm(pkgs)
	f.inputs[empFieldFullName].SetValue(emp.FullName)
	f.inputs[empFieldCINNum].SetValue(emp.CINNum)
	f.inputs[empFieldCNSSNum].SetValue(emp.CNSSNum)
	f.inputs[empFieldBirthDate].SetValue(emp.BirthDate.Format("2006-01-02"))
	f.inputs[empFieldHireDate].SetValue(emp.HireDate.Format("2006-01-02"))
	f.inputs[empFieldPosition].SetValue(emp.Position)
	f.inputs[empFieldDisplayName].SetValue(emp.DisplayName)
	f.inputs[empFieldNumDependents].SetValue(strconv.Itoa(emp.NumDependents))
	f.inputs[empFieldNumChildren].SetValue(strconv.Itoa(emp.NumChildren))
	f.inputs[empFieldPhoneNumber].SetValue(emp.PhoneNumber)
	f.inputs[empFieldEmailAddress].SetValue(emp.EmailAddress)
	f.inputs[empFieldAddress].SetValue(emp.Address)
	f.inputs[empFieldBankRIB].SetValue(emp.BankRIB)

	for i, g := range genderValues {
		if g == emp.Gender {
			f.genderIdx = i
			break
		}
	}
	for i, ms := range maritalValues {
		if ms == emp.MaritalStatus {
			f.maritalIdx = i
			break
		}
	}
	for i, pkg := range pkgs {
		if pkg.ID == emp.CompensationPackageID {
			f.pkgIdx = i
			break
		}
	}
	return f
}

func (f empForm) cycleCurrent(delta int) empForm {
	switch f.focused {
	case empFieldGender:
		f.genderIdx = (f.genderIdx + delta + len(genderValues)) % len(genderValues)
	case empFieldMaritalStatus:
		f.maritalIdx = (f.maritalIdx + delta + len(maritalValues)) % len(maritalValues)
	case empFieldCompPkg:
		if len(f.pkgs) > 0 {
			f.pkgIdx = (f.pkgIdx + delta + len(f.pkgs)) % len(f.pkgs)
		}
	}
	return f
}

func (f empForm) advanceFocus(delta int) empForm {
	if !isCycleOrPickerField(f.focused) {
		f.inputs[f.focused].Blur()
	}
	f.focused = (f.focused + delta + empFieldCount) % empFieldCount
	if !isCycleOrPickerField(f.focused) {
		f.inputs[f.focused].Focus()
	}
	// Keep focused field within the visible window.
	if f.focused < f.viewOffset {
		f.viewOffset = f.focused
	} else if f.focused >= f.viewOffset+visibleFields {
		f.viewOffset = f.focused - visibleFields + 1
	}
	return f
}

func (f empForm) update(msg tea.KeyMsg) (empForm, formResult, tea.Cmd) {
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

	if isCycleOrPickerField(f.focused) {
		switch msg.String() {
		case "left", "h":
			return f.cycleCurrent(-1), formContinue, nil
		case "right", "l", " ":
			return f.cycleCurrent(1), formContinue, nil
		}
		return f, formContinue, nil
	}

	var cmd tea.Cmd
	f.inputs[f.focused], cmd = f.inputs[f.focused].Update(msg)
	return f, formContinue, cmd
}

func (f empForm) toDomain(orgID uuid.UUID) (*domain.Employee, error) {
	fullName := strings.TrimSpace(f.inputs[empFieldFullName].Value())
	if fullName == "" {
		return nil, errors.New("Full name is required")
	}

	cinNum := strings.TrimSpace(f.inputs[empFieldCINNum].Value())
	if cinNum == "" {
		return nil, errors.New("CIN number is required")
	}

	birthDateStr := strings.TrimSpace(f.inputs[empFieldBirthDate].Value())
	if birthDateStr == "" {
		return nil, errors.New("Birth date is required (YYYY-MM-DD)")
	}
	birthDate, err := time.Parse("2006-01-02", birthDateStr)
	if err != nil {
		return nil, errors.New("Birth date must be in YYYY-MM-DD format")
	}

	hireDateStr := strings.TrimSpace(f.inputs[empFieldHireDate].Value())
	if hireDateStr == "" {
		return nil, errors.New("Hire date is required (YYYY-MM-DD)")
	}
	hireDate, parseErr := time.Parse("2006-01-02", hireDateStr)
	if parseErr != nil {
		return nil, errors.New("Hire date must be in YYYY-MM-DD format")
	}

	position := strings.TrimSpace(f.inputs[empFieldPosition].Value())
	if position == "" {
		return nil, errors.New("Position is required")
	}

	if len(f.pkgs) == 0 {
		return nil, errors.New("No compensation packages available — create one first")
	}

	numDependents := 0
	if depStr := strings.TrimSpace(f.inputs[empFieldNumDependents].Value()); depStr != "" {
		numDependents, err = strconv.Atoi(depStr)
		if err != nil || numDependents < 0 {
			return nil, errors.New("Dependents must be a non-negative integer")
		}
	}

	numChildren := 0
	if childrenStr := strings.TrimSpace(f.inputs[empFieldNumChildren].Value()); childrenStr != "" {
		numChildren, err = strconv.Atoi(childrenStr)
		if err != nil || numChildren < 0 {
			return nil, errors.New("Children must be a non-negative integer")
		}
	}

	return &domain.Employee{
		OrgID:                 orgID,
		FullName:              fullName,
		CINNum:                cinNum,
		CNSSNum:               strings.TrimSpace(f.inputs[empFieldCNSSNum].Value()),
		BirthDate:             birthDate.UTC(),
		Gender:                genderValues[f.genderIdx],
		HireDate:              hireDate.UTC(),
		Position:              position,
		CompensationPackageID: f.pkgs[f.pkgIdx].ID,
		DisplayName:           strings.TrimSpace(f.inputs[empFieldDisplayName].Value()),
		MaritalStatus:         maritalValues[f.maritalIdx],
		NumDependents:         numDependents,
		NumChildren:           numChildren,
		PhoneNumber:           strings.TrimSpace(f.inputs[empFieldPhoneNumber].Value()),
		EmailAddress:          strings.TrimSpace(f.inputs[empFieldEmailAddress].Value()),
		Address:               strings.TrimSpace(f.inputs[empFieldAddress].Value()),
		BankRIB:               strings.TrimSpace(f.inputs[empFieldBankRIB].Value()),
	}, nil
}

func (f empForm) view() string {
	labelReqStyle := lipgloss.NewStyle().Width(16).Foreground(lipgloss.Color("205")).Bold(true)
	labelOptStyle := lipgloss.NewStyle().Width(16).Foreground(lipgloss.Color("245"))
	cycleActiveStyle := lipgloss.NewStyle().Width(26).Foreground(lipgloss.Color("205"))
	cycleIdleStyle := lipgloss.NewStyle().Width(26).Foreground(lipgloss.Color("240"))

	end := f.viewOffset + visibleFields
	if end > empFieldCount {
		end = empFieldCount
	}

	var rows []string
	for i := f.viewOffset; i < end; i++ {
		meta := empFieldMeta[i]
		labelStr := meta.label
		if meta.required {
			labelStr += " *"
		}

		var labelRendered string
		if meta.required {
			labelRendered = labelReqStyle.Render(labelStr)
		} else {
			labelRendered = labelOptStyle.Render(labelStr)
		}

		var valueRendered string
		switch i {
		case empFieldGender:
			val := string(genderValues[f.genderIdx])
			if i == f.focused {
				valueRendered = cycleActiveStyle.Render("‹ " + val + " ›")
			} else {
				valueRendered = cycleIdleStyle.Render("  " + val + "  ")
			}
		case empFieldMaritalStatus:
			val := string(maritalValues[f.maritalIdx])
			if i == f.focused {
				valueRendered = cycleActiveStyle.Render("‹ " + val + " ›")
			} else {
				valueRendered = cycleIdleStyle.Render("  " + val + "  ")
			}
		case empFieldCompPkg:
			var val string
			if len(f.pkgs) == 0 {
				val = "(none)"
			} else {
				val = f.pkgs[f.pkgIdx].Name
			}
			if i == f.focused {
				valueRendered = cycleActiveStyle.Render("‹ " + val + " ›")
			} else {
				valueRendered = cycleIdleStyle.Render("  " + val + "  ")
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, labelRendered, valueRendered))
			// Static salary info row below the picker.
			salaryInfo := ""
			if len(f.pkgs) > 0 {
				pkg := f.pkgs[f.pkgIdx]
				salaryInfo = fmt.Sprintf("%.2f %s", pkg.BaseSalary.ToMAD(), pkg.Currency)
			}
			rows = append(rows, lipgloss.NewStyle().
				Width(16).Foreground(lipgloss.Color("240")).Render("Base Salary")+
				lipgloss.NewStyle().Width(26).Foreground(lipgloss.Color("243")).Render("  "+salaryInfo))
			continue
		default:
			valueRendered = f.inputs[i].View()
		}

		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, labelRendered, valueRendered))
	}

	// Scroll indicator when not all fields are visible.
	hasAbove := f.viewOffset > 0
	hasBelow := end < empFieldCount
	indicator := ""
	switch {
	case hasAbove && hasBelow:
		indicator = fmt.Sprintf("  ↑ %d above · ↓ %d below", f.viewOffset, empFieldCount-end)
	case hasAbove:
		indicator = fmt.Sprintf("  ↑ %d more above", f.viewOffset)
	case hasBelow:
		indicator = fmt.Sprintf("  ↓ %d more below", empFieldCount-end)
	}
	if indicator != "" {
		rows = append(rows, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(indicator))
	}

	return strings.Join(rows, "\n")
}

// ---------------------------------------------------------------------------
// empSection
// ---------------------------------------------------------------------------

type empSection struct {
	empSvc             *application.EmployeeService
	compSvc            *application.CompensationPackageService
	payrollSvc         *application.PayrollService
	orgID              uuid.UUID
	list               list.Model
	historyList        list.Model
	pkgs               []*domain.EmployeeCompensationPackage
	state              empState
	form               empForm
	pendingDeleteID    uuid.UUID
	editTarget         *domain.Employee // non-nil when editing
	detailTarget       *domain.Employee // non-nil when viewing detail
	detailPkgName      string
	selectedHistResult *domain.PayrollResult
	selectedHistPeriod *domain.PayrollPeriod
	errMsg             string
	width, height      int
}

func newEmpSection(
	empSvc *application.EmployeeService,
	compSvc *application.CompensationPackageService,
	payrollSvc *application.PayrollService,
	orgID uuid.UUID,
) *empSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Employees"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.NoItems = l.Styles.NoItems.PaddingLeft(2)

	hl := list.New(nil, delegate, 0, 0)
	hl.SetShowHelp(false)
	hl.SetShowStatusBar(false)
	hl.SetFilteringEnabled(false)
	hl.Styles.NoItems = hl.Styles.NoItems.PaddingLeft(2)

	return &empSection{
		empSvc:      empSvc,
		compSvc:     compSvc,
		payrollSvc:  payrollSvc,
		orgID:       orgID,
		list:        l,
		historyList: hl,
	}
}

// ---------------------------------------------------------------------------
// sectionModel interface
// ---------------------------------------------------------------------------

func (s *empSection) Init() tea.Cmd {
	return loadEmpsCmd(s.empSvc, s.compSvc, s.orgID)
}

func (s *empSection) IsOverlay() bool {
	if s.state == empStateList || s.state == empStateDeleted || s.state == empStateHistory {
		return s.list.FilterState() == list.Filtering
	}
	return true
}

func (s *empSection) ShortHelp() []key.Binding {
	switch s.state {
	case empStateCreating, empStateEditing:
		return []key.Binding{formKeys.Submit, formKeys.Cancel}
	case empStateDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	case empStateDetail:
		return []key.Binding{empHistoryKey, sectionBackKey}
	case empStateHistory:
		return []key.Binding{payrollKeys.ViewResults, sectionBackKey}
	case empStateHistoryDetail:
		return []key.Binding{sectionBackKey}
	case empStateDeleted:
		return []key.Binding{
			mainKeys.ToggleDeleted,
			mainKeys.Restore,
			mainKeys.HardDelete,
		}
	case empStateHardDeleting:
		return []key.Binding{confirmKeys.Yes, confirmKeys.No}
	default:
		return []key.Binding{empDetailKey, mainKeys.New, mainKeys.Edit, mainKeys.Delete, mainKeys.Filter, mainKeys.ToggleDeleted}
	}
}

func (s *empSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		s.historyList.SetSize(s.width, s.listHeight())
		return s, nil

	case activeOrgLoadedMsg:
		s.orgID = msg.orgID
		if s.state == empStateDeleted {
			return s, loadDeletedEmpsCmd(s.empSvc, s.orgID)
		}
		return s, loadEmpsCmd(s.empSvc, s.compSvc, s.orgID)

	case compsLoadedMsg:
		// Keep the package picker in sync whenever comp packages are reloaded.
		if msg.err == nil {
			s.pkgs = msg.pkgs
			// Rebuild list items so package names stay current (e.g. after a rename).
			names := pkgNameMap(msg.pkgs)
			current := s.list.Items()
			for i, it := range current {
				if ei, ok := it.(empItem); ok {
					current[i] = empItem{emp: ei.emp, pkgName: names[ei.emp.CompensationPackageID]}
				}
			}
			return s, s.list.SetItems(current)
		}
		return s, nil

	case empsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load employees — try again"
			return s, nil
		}
		s.pkgs = msg.pkgs
		names := pkgNameMap(msg.pkgs)
		items := make([]list.Item, len(msg.emps))
		for i, e := range msg.emps {
			items[i] = empItem{emp: e, pkgName: names[e.CompensationPackageID]}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case saveEmpDoneMsg:
		s.state = empStateList
		s.editTarget = nil
		if msg.err != nil {
			s.errMsg = userFriendlyEmpError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadEmpsCmd(s.empSvc, s.compSvc, s.orgID)

	case deleteEmpDoneMsg:
		s.state = empStateList
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = userFriendlyEmpError(msg.err)
			return s, nil
		}
		s.errMsg = ""
		return s, loadEmpsCmd(s.empSvc, s.compSvc, s.orgID)

	case empHistoryLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load employees — try again"
			s.state = empStateDetail
			return s, nil
		}
		items := make([]list.Item, len(msg.entries))
		for i, e := range msg.entries {
			items[i] = empHistoryItem{entry: e}
		}
		cmd := s.historyList.SetItems(items)
		s.state = empStateHistory
		s.errMsg = ""
		return s, cmd

	case empsDeletedLoadedMsg:
		if msg.err != nil {
			s.errMsg = "Could not load deleted employees — try again"
			return s, nil
		}
		var items []list.Item
		for _, e := range msg.emps {
			if e.DeletedAt != nil {
				items = append(items, empItem{emp: e, pkgName: ""})
			}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case restoreEmpDoneMsg:
		if msg.err != nil {
			s.errMsg = "Restore failed — try again"
			return s, nil
		}
		s.errMsg = ""
		return s, loadDeletedEmpsCmd(s.empSvc, s.orgID)

	case hardDeleteEmpDoneMsg:
		s.pendingDeleteID = uuid.Nil
		if msg.err != nil {
			s.errMsg = "Hard delete failed — try again"
			return s, nil
		}
		s.errMsg = ""
		return s, loadDeletedEmpsCmd(s.empSvc, s.orgID)

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	switch s.state {
	case empStateList, empStateDeleted:
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	case empStateHistory:
		var cmd tea.Cmd
		s.historyList, cmd = s.historyList.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *empSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case empStateDeleted:
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}
		switch {
		case key.Matches(msg, mainKeys.ToggleDeleted):
			s.list.Title = "Employees"
			s.state = empStateList
			s.errMsg = ""
			return s, loadEmpsCmd(s.empSvc, s.compSvc, s.orgID)

		case key.Matches(msg, mainKeys.Restore):
			selected, ok := s.list.SelectedItem().(empItem)
			if !ok {
				return s, nil
			}
			return s, restoreEmpCmd(s.empSvc, selected.emp.ID)

		case key.Matches(msg, mainKeys.HardDelete):
			selected, ok := s.list.SelectedItem().(empItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.emp.ID
			s.state = empStateHardDeleting
			return s, nil
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case empStateHardDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.pendingDeleteID = uuid.Nil
			s.state = empStateDeleted
			return s, hardDeleteEmpCmd(s.empSvc, id)
		case key.Matches(msg, confirmKeys.No):
			s.pendingDeleteID = uuid.Nil
			s.state = empStateDeleted
			return s, nil
		}
		return s, nil

	case empStateList:
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
			if len(s.pkgs) == 0 {
				s.errMsg = "Create a compensation package first"
				return s, nil
			}
			s.form = newEmpForm(s.pkgs)
			s.state = empStateCreating
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Edit):
			selected, ok := s.list.SelectedItem().(empItem)
			if !ok {
				return s, nil
			}
			s.editTarget = selected.emp
			s.form = newEmpFormFromEmployee(s.pkgs, selected.emp)
			s.state = empStateEditing
			s.errMsg = ""
			return s, nil

		case key.Matches(msg, mainKeys.Delete):
			selected, ok := s.list.SelectedItem().(empItem)
			if !ok {
				return s, nil
			}
			s.pendingDeleteID = selected.emp.ID
			s.state = empStateDeleting
			return s, nil

		case key.Matches(msg, empDetailKey):
			if selected, ok := s.list.SelectedItem().(empItem); ok {
				s.detailTarget = selected.emp
				s.detailPkgName = selected.pkgName
				s.state = empStateDetail
			}
			return s, nil

		case key.Matches(msg, mainKeys.ToggleDeleted):
			s.list.Title = "Employees [DELETED]"
			s.state = empStateDeleted
			s.errMsg = ""
			return s, loadDeletedEmpsCmd(s.empSvc, s.orgID)
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case empStateCreating, empStateEditing:
		f, result, cmd := s.form.update(msg)
		s.form = f
		switch result {
		case formSubmit:
			emp, err := s.form.toDomain(s.orgID)
			if err != nil {
				s.errMsg = err.Error()
				return s, nil
			}
			s.errMsg = ""
			if s.state == empStateEditing && s.editTarget != nil {
				// Preserve immutable fields from the original record.
				emp.ID = s.editTarget.ID
				emp.OrgID = s.editTarget.OrgID
				emp.SerialNum = s.editTarget.SerialNum
				emp.CreatedAt = s.editTarget.CreatedAt
				s.state = empStateList
				return s, updateEmpCmd(s.empSvc, emp)
			}
			s.state = empStateList
			return s, createEmpCmd(s.empSvc, emp)
		case formCancel:
			s.state = empStateList
			s.editTarget = nil
			s.errMsg = ""
			return s, nil
		default:
			return s, cmd
		}

	case empStateDeleting:
		switch {
		case key.Matches(msg, confirmKeys.Yes):
			id := s.pendingDeleteID
			s.state = empStateList
			s.pendingDeleteID = uuid.Nil
			return s, deleteEmpCmd(s.empSvc, id)
		case key.Matches(msg, confirmKeys.No):
			s.state = empStateList
			s.pendingDeleteID = uuid.Nil
			return s, nil
		}

	case empStateDetail:
		switch {
		case key.Matches(msg, empHistoryKey):
			s.historyList.Title = s.detailTarget.FullName + " — Payroll History"
			s.state = empStateHistory
			return s, loadEmpHistoryCmd(s.payrollSvc, s.detailTarget.ID)
		case key.Matches(msg, sectionBackKey):
			s.state = empStateList
			s.detailTarget = nil
			s.detailPkgName = ""
			return s, nil
		}
		return s, nil

	case empStateHistory:
		switch {
		case key.Matches(msg, sectionBackKey):
			s.state = empStateDetail
			return s, nil
		case key.Matches(msg, payrollKeys.ViewResults):
			if selected, ok := s.historyList.SelectedItem().(empHistoryItem); ok {
				s.selectedHistResult = selected.entry.result
				s.selectedHistPeriod = selected.entry.period
				s.state = empStateHistoryDetail
			}
			return s, nil
		}
		var cmd tea.Cmd
		s.historyList, cmd = s.historyList.Update(msg)
		return s, cmd

	case empStateHistoryDetail:
		if key.Matches(msg, sectionBackKey) {
			s.state = empStateHistory
			s.selectedHistResult = nil
			s.selectedHistPeriod = nil
			return s, nil
		}
		return s, nil
	}
	return s, nil
}

// ---------------------------------------------------------------------------
// View
// ---------------------------------------------------------------------------

func (s *empSection) View(width, height int) string {
	if s.state == empStateHistory {
		histView := s.historyList.View()
		if s.errMsg != "" {
			statusRow := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Width(width).
				Render("  " + s.errMsg)
			histView = lipgloss.JoinVertical(lipgloss.Left, histView, statusRow)
		}
		return histView
	}

	listView := s.list.View()

	statusRow := ""
	if s.errMsg != "" {
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Width(width).
			Render("  " + s.errMsg)
	}
	if statusRow == "" && len(s.list.Items()) == 0 {
		var hint string
		switch {
		case s.state == empStateDeleted:
			hint = "No deleted employees."
		case s.orgID == uuid.Nil:
			hint = "Select an active organization first."
		case len(s.pkgs) == 0:
			hint = "Create a compensation package before adding employees."
		default:
			hint = "Press n to add your first employee."
		}
		statusRow = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Width(width).
			Render("  " + hint)
	}
	if statusRow != "" {
		listView = lipgloss.JoinVertical(lipgloss.Left, listView, statusRow)
	}

	switch s.state {
	case empStateDeleting:
		return s.renderDeleteConfirm(listView, width)
	case empStateHardDeleting:
		return s.renderHardDeleteConfirm(listView, width)
	case empStateCreating:
		return s.renderFormOverlay("New Employee", width, height)
	case empStateEditing:
		return s.renderFormOverlay("Edit Employee", width, height)
	case empStateDetail:
		return s.renderEmpDetail(width, height)
	case empStateHistoryDetail:
		return renderPayrollResultDetail(s.selectedHistResult, s.detailTarget.FullName, s.selectedHistPeriod, width, height)
	}
	return listView
}

func (s *empSection) renderFormOverlay(title string, width, height int) string {
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

func (s *empSection) renderHardDeleteConfirm(listView string, width int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(empItem); ok {
		name = selected.Title()
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Hard-delete employee %q? This is permanent and cannot be undone. [y] yes  [n/bksp] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
}

func (s *empSection) renderDeleteConfirm(listView string, width int) string {
	name := ""
	if selected, ok := s.list.SelectedItem().(empItem); ok {
		name = selected.Title()
	}
	prompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("196")).
		Bold(true).
		Width(width).
		Render(fmt.Sprintf("  Delete employee %q? [y] yes  [n/bksp] cancel", name))
	return lipgloss.JoinVertical(lipgloss.Left, listView, prompt)
}

func (s *empSection) renderEmpDetail(width, height int) string {
	e := s.detailTarget

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))

	opt := func(v string) string {
		if v == "" {
			return "—"
		}
		return v
	}
	row := func(label, value string) string {
		return fmt.Sprintf("  %-20s%-22s", label, value)
	}
	divider := func(label string) string {
		return sectionStyle.Render("── " + label + " " + strings.Repeat("─", 20))
	}

	lines := []string{
		titleStyle.Render(fmt.Sprintf("#%d · %s", e.SerialNum, e.FullName)),
		"",
		divider("Personal"),
		row("Full Name", e.FullName),
		row("Display Name", opt(e.DisplayName)),
		row("Birth Date", e.BirthDate.Format("2006-01-02")),
		row("Gender", string(e.Gender)),
		row("CIN", e.CINNum),
		row("CNSS", opt(e.CNSSNum)),
		row("Phone", opt(e.PhoneNumber)),
		row("Email", opt(e.EmailAddress)),
		row("Address", opt(e.Address)),
		"",
		divider("Employment"),
		row("Employee #", fmt.Sprintf("%d", e.SerialNum)),
		row("Position", e.Position),
		row("Hire Date", e.HireDate.Format("2006-01-02")),
		row("Package", s.detailPkgName),
		"",
		divider("Tax Info"),
		row("Marital Status", string(e.MaritalStatus)),
		row("Dependents", fmt.Sprintf("%d", e.NumDependents)),
		row("Children", fmt.Sprintf("%d", e.NumChildren)),
		"",
		divider("Banking"),
		row("Bank RIB", opt(e.BankRIB)),
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

func (s *empSection) listHeight() int {
	if s.height <= 1 {
		return s.height
	}
	return s.height - 1
}

func userFriendlyEmpError(err error) string {
	switch {
	case errors.Is(err, application.ErrEmployeeNotFound):
		return "Employee not found"
	case errors.Is(err, application.ErrEmployeeExists):
		return "CIN or CNSS number already in use"
	default:
		return "Something went wrong — please try again"
	}
}
