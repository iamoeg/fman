package tui

import "github.com/iamoeg/bootdev-capstone/internal/application"

// App is the composition root for the TUI.
// It holds all application services and is constructed once in cmd/tui/main.go.
// It is not a tea.Model — it is purely a wiring container.
type App struct {
	OrganizationService *application.OrganizationService
	EmployeeService     *application.EmployeeService
	CompensationService *application.CompensationPackageService
	PayrollService      *application.PayrollService
}
