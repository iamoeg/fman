package tui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/iamoeg/bootdev-capstone/internal/application"
)

var (
	auditDetailKey = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "detail"),
	)
	auditRefreshKey = key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "refresh"),
	)
)

type auditState int

const (
	auditStateList   auditState = iota
	auditStateDetail            // detail overlay
)

type auditSection struct {
	svc           *application.AuditLogService
	list          list.Model
	state         auditState
	errMsg        string
	width, height int
}

func newAuditSection(svc *application.AuditLogService) *auditSection {
	delegate := list.NewDefaultDelegate()
	l := list.New(nil, delegate, 0, 0)
	l.Title = "Audit Log"
	l.SetShowHelp(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.Styles.NoItems = l.Styles.NoItems.PaddingLeft(2)
	return &auditSection{svc: svc, list: l}
}

// sectionModel interface

func (s *auditSection) Init() tea.Cmd {
	return loadAuditLogsCmd(s.svc)
}

func (s *auditSection) IsOverlay() bool {
	return s.state == auditStateDetail || s.list.FilterState() == list.Filtering
}

func (s *auditSection) ShortHelp() []key.Binding {
	if s.state == auditStateDetail {
		return []key.Binding{mainKeys.Back}
	}
	return []key.Binding{auditDetailKey, auditRefreshKey, mainKeys.Filter}
}

func (s *auditSection) Update(msg tea.Msg) (sectionModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		s.width = msg.Width - sidebarWidth - 2
		s.height = msg.Height - headerHeight - footerHeight - 2
		s.list.SetSize(s.width, s.listHeight())
		return s, nil

	case auditLogsLoadedMsg:
		if msg.err != nil {
			s.errMsg = "load error: " + msg.err.Error()
			return s, nil
		}
		items := make([]list.Item, len(msg.logs))
		for i, l := range msg.logs {
			items[i] = auditItem{log: l}
		}
		cmd := s.list.SetItems(items)
		s.errMsg = ""
		return s, cmd

	case tea.KeyMsg:
		return s.updateKey(msg)
	}

	if s.state == auditStateList {
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd
	}
	return s, nil
}

func (s *auditSection) updateKey(msg tea.KeyMsg) (sectionModel, tea.Cmd) {
	switch s.state {

	case auditStateList:
		if s.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			s.list, cmd = s.list.Update(msg)
			return s, cmd
		}
		switch {
		case key.Matches(msg, auditDetailKey):
			if _, ok := s.list.SelectedItem().(auditItem); ok {
				s.state = auditStateDetail
				s.errMsg = ""
			}
			return s, nil
		case key.Matches(msg, auditRefreshKey):
			return s, loadAuditLogsCmd(s.svc)
		}
		var cmd tea.Cmd
		s.list, cmd = s.list.Update(msg)
		return s, cmd

	case auditStateDetail:
		if key.Matches(msg, mainKeys.Back) {
			s.state = auditStateList
		}
		return s, nil
	}
	return s, nil
}

func (s *auditSection) View(width, height int) string {
	listView := s.list.View()

	if s.errMsg != "" {
		statusRow := lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Width(width).
			Render("  " + s.errMsg)
		listView = lipgloss.JoinVertical(lipgloss.Left, listView, statusRow)
	}

	if s.state == auditStateDetail {
		selected, ok := s.list.SelectedItem().(auditItem)
		if !ok {
			return listView
		}
		return renderAuditDetail(selected.log, width, height)
	}
	return listView
}

func formatJSON(raw string) string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(raw), "", "  "); err != nil {
		return raw
	}
	return buf.String()
}

func (s *auditSection) listHeight() int {
	if s.height <= 1 {
		return s.height
	}
	return s.height - 1
}

func renderAuditDetail(log *application.AuditLog, width, height int) string {
	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	labelStyle := lipgloss.NewStyle().Width(12).Foreground(lipgloss.Color("245"))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	row := func(label, value string) string {
		return lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label), value)
	}
	divider := func(label string) string {
		return sectionStyle.Render("── " + label + " " + strings.Repeat("─", 16))
	}

	lines := []string{
		titleStyle.Render("Audit Entry"),
		"",
		row("Action", log.Action),
		row("Table", log.TableName),
		row("Record ID", log.RecordID),
		row("Timestamp", log.Timestamp.UTC().Format("2006-01-02 15:04:05 UTC")),
		"",
	}

	hasBefore := log.Before != "" && log.Before != "null"
	hasAfter := log.After != "" && log.After != "null"

	boxWidth := width - 8
	if boxWidth > 160 {
		boxWidth = 160
	}

	if hasBefore && hasAfter {
		colWidth := (boxWidth - 2) / 2
		labelBefore := sectionStyle.Render("── Before " + strings.Repeat("─", max(0, colWidth-12)))
		labelAfter := sectionStyle.Render("── After " + strings.Repeat("─", max(0, colWidth-11)))
		colStyle := lipgloss.NewStyle().Width(colWidth).Foreground(lipgloss.Color("240"))
		leftPane := lipgloss.JoinVertical(lipgloss.Left, labelBefore, colStyle.Render(formatJSON(log.Before)))
		rightPane := lipgloss.JoinVertical(lipgloss.Left, labelAfter, colStyle.Render(formatJSON(log.After)))
		spacer := lipgloss.NewStyle().Width(2).Render("")
		lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, leftPane, spacer, rightPane), "")
	} else {
		if hasBefore {
			lines = append(lines, divider("Before"))
			lines = append(lines, dimStyle.Render(formatJSON(log.Before)))
			lines = append(lines, "")
		}
		if hasAfter {
			lines = append(lines, divider("After"))
			lines = append(lines, dimStyle.Render(formatJSON(log.After)))
			lines = append(lines, "")
		}
	}

	lines = append(lines, hintStyle.Render("            [esc] close"))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("205")).
		Padding(1, 2).
		Width(boxWidth).
		Render(strings.Join(lines, "\n"))

	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("235")),
	)
}
