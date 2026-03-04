package tui

import "github.com/iamoeg/bootdev-capstone/internal/application"

// App is the composition root for the TUI.
// It holds all application services and is constructed once in cmd/tui/main.go.
// It is not a tea.Model — it is purely a wiring container.
type App struct {
	Organizations *application.OrganizationService
	Employees     *application.EmployeeService
	Compensation  *application.CompensationPackageService
	Payroll       *application.PayrollService
}
