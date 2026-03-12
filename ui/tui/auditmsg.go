package tui

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/iamoeg/bootdev-capstone/internal/application"
)

type auditLogsLoadedMsg struct {
	logs []*application.AuditLog
	err  error
}

func loadAuditLogsCmd(svc *application.AuditLogService) tea.Cmd {
	return func() tea.Msg {
		logs, err := svc.FindRecent(context.Background(), 100)
		return auditLogsLoadedMsg{logs: logs, err: err}
	}
}
