package tui

import (
	"fmt"

	"github.com/iamoeg/bootdev-capstone/internal/application"
)

type auditItem struct {
	log *application.AuditLog
}

func (i auditItem) Title() string {
	return fmt.Sprintf("[%s] %s", i.log.Action, i.log.TableName)
}

func (i auditItem) Description() string {
	id := i.log.RecordID
	if len(id) > 18 {
		id = id[:18] + "…"
	}
	return fmt.Sprintf("%s  ·  %s", id, i.log.Timestamp.Format("2006-01-02 15:04:05"))
}

func (i auditItem) FilterValue() string {
	return i.log.TableName + " " + i.log.Action
}
