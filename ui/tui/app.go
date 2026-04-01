// Package tui implements the terminal user interface for the application.
package tui

import (
	"github.com/iamoeg/fman/internal/application"
	"github.com/iamoeg/fman/pkg/config"
)

// App is the composition root for the TUI.
// It holds all application services and is constructed once in cmd/tui/main.go.
// It is not a tea.Model — it is purely a wiring container.
type App struct {
	Config              *config.Config
	OrganizationService *application.OrganizationService
	EmployeeService     *application.EmployeeService
	CompensationService *application.CompensationPackageService
	PayrollService      *application.PayrollService
	AuditLogService     *application.AuditLogService
}
