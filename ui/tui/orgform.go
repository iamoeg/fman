package tui

import (
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// formResult signals what happened after processing a key in the form.
type formResult int

const (
	formContinue formResult = iota
	formSubmit
	formCancel
)

// Field indices within orgForm.inputs.
const (
	fieldName     = 0
	fieldAddress  = 1
	fieldActivity = 2
	fieldICE      = 3
	fieldIF       = 4
	fieldRC       = 5
	fieldCNSS     = 6
	fieldRIB      = 7
	fieldCount    = 8
)

var fieldLabels = [fieldCount]string{
	"Name *", "Address", "Activity", "ICE", "IF", "RC", "CNSS", "Bank RIB",
}

var fieldPlaceholders = [fieldCount]string{
	"Company name (required)",
	"Street, city",
	"Business activity",
	"ICE number",
	"Tax ID (IF)",
	"Commerce registry (RC)",
	"CNSS number",
	"Bank RIB",
}

// orgForm holds state for the create/edit overlay form.
type orgForm struct {
	inputs     [fieldCount]textinput.Model
	focusIndex int
	editing    bool
	orgID      uuid.UUID // uuid.Nil when creating
}

func newOrgForm() orgForm {
	var f orgForm
	for i := range f.inputs {
		t := textinput.New()
		t.Placeholder = fieldPlaceholders[i]
		t.Width = 38
		f.inputs[i] = t
	}
	f.inputs[fieldName].CharLimit = 120
	f.inputs[fieldName].Focus()
	return f
}

func newOrgFormFromOrg(org *domain.Organization) orgForm {
	f := newOrgForm()
	f.editing = true
	f.orgID = org.ID
	f.inputs[fieldName].SetValue(org.Name)
	f.inputs[fieldAddress].SetValue(org.Address)
	f.inputs[fieldActivity].SetValue(org.Activity)
	f.inputs[fieldICE].SetValue(org.ICENum)
	f.inputs[fieldIF].SetValue(org.IFNum)
	f.inputs[fieldRC].SetValue(org.RCNum)
	f.inputs[fieldCNSS].SetValue(org.CNSSNum)
	f.inputs[fieldRIB].SetValue(org.BankRIB)
	return f
}

// update processes a key message, returning updated form, result signal, and any cmd.
func (f orgForm) update(msg tea.KeyMsg) (orgForm, formResult, tea.Cmd) {
	switch {
	case key.Matches(msg, formKeys.Cancel):
		return f, formCancel, nil

	case key.Matches(msg, formKeys.Submit):
		return f, formSubmit, nil

	case key.Matches(msg, formKeys.NextField):
		f.inputs[f.focusIndex].Blur()
		f.focusIndex = (f.focusIndex + 1) % fieldCount
		cmd := f.inputs[f.focusIndex].Focus()
		return f, formContinue, cmd

	case key.Matches(msg, formKeys.PrevField):
		f.inputs[f.focusIndex].Blur()
		f.focusIndex = (f.focusIndex + fieldCount - 1) % fieldCount
		cmd := f.inputs[f.focusIndex].Focus()
		return f, formContinue, cmd

	default:
		var cmd tea.Cmd
		f.inputs[f.focusIndex], cmd = f.inputs[f.focusIndex].Update(msg)
		return f, formContinue, cmd
	}
}

// toDomain converts form values to a domain.Organization.
// Returns an error if required fields are missing.
func (f orgForm) toDomain() (*domain.Organization, error) {
	name := strings.TrimSpace(f.inputs[fieldName].Value())
	if name == "" {
		return nil, errors.New("Name is required")
	}
	org := &domain.Organization{
		Name:      name,
		Address:   strings.TrimSpace(f.inputs[fieldAddress].Value()),
		Activity:  strings.TrimSpace(f.inputs[fieldActivity].Value()),
		LegalForm: domain.LegalFormSARL,
		ICENum:    strings.TrimSpace(f.inputs[fieldICE].Value()),
		IFNum:     strings.TrimSpace(f.inputs[fieldIF].Value()),
		RCNum:     strings.TrimSpace(f.inputs[fieldRC].Value()),
		CNSSNum:   strings.TrimSpace(f.inputs[fieldCNSS].Value()),
		BankRIB:   strings.TrimSpace(f.inputs[fieldRIB].Value()),
	}
	if f.editing {
		org.ID = f.orgID
	}
	return org, nil
}

// view renders the form fields as a string for use inside the overlay box.
func (f orgForm) view() string {
	labelW := 12
	labelStyle := lipgloss.NewStyle().
		Width(labelW).
		Foreground(lipgloss.Color("245"))
	activeLabelStyle := lipgloss.NewStyle().
		Width(labelW).
		Foreground(lipgloss.Color("205")).
		Bold(true)
	staticStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	var rows []string
	for i, inp := range &f.inputs {
		lbl := labelStyle.Render(fieldLabels[i])
		if i == f.focusIndex {
			lbl = activeLabelStyle.Render(fieldLabels[i])
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, lbl, inp.View()))

		// Insert static LegalForm row after Activity.
		if i == fieldActivity {
			legalLabel := labelStyle.Render("Legal Form")
			legalVal := staticStyle.Render("SARL  (only supported form)")
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Center, legalLabel, legalVal))
		}
	}
	return strings.Join(rows, "\n")
}
