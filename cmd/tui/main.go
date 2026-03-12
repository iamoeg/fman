package main

import (
	"database/sql"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	_ "github.com/mattn/go-sqlite3"

	"github.com/iamoeg/bootdev-capstone/db/migration"
	calculator "github.com/iamoeg/bootdev-capstone/internal/adapter/payroll"
	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
	"github.com/iamoeg/bootdev-capstone/ui/tui"
)

func main() {
	// Debug logging — set DEBUG=1 to enable.
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatal("fatal: could not open debug log:", err)
		}
		defer f.Close()
	}

	// 1. Load (or create) config.
	cfg, err := config.LoadOrCreate("")
	if err != nil {
		log.Fatal("fatal: could not load config:", err)
	}

	// 2. Resolve database path and open SQLite.
	dbPath, err := cfg.ResolveDatabasePath()
	if err != nil {
		log.Fatal("fatal: could not resolve database path:", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal("fatal: could not open database:", err)
	}
	defer db.Close()

	// CRITICAL: enable foreign key enforcement on every connection.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		log.Fatal("fatal: could not enable foreign keys:", err)
	}

	// 3. Run migrations.
	if err := migration.RunMigrations(db); err != nil {
		log.Fatal("fatal: could not run migrations:", err)
	}

	// 4. Build repositories.
	orgRepo := sqlite.NewOrganizationRepository(db)
	empRepo := sqlite.NewEmployeeRepository(db)
	compRepo := sqlite.NewCompensationPackageRepository(db)
	periodRepo := sqlite.NewPayrollPeriodRepository(db)
	resultRepo := sqlite.NewPayrollResultRepository(db)
	auditRepo := sqlite.NewAuditLogRepository(db)

	// 5. Build services.
	orgSvc := application.NewOrganizationService(orgRepo)
	empSvc := application.NewEmployeeService(empRepo)
	compSvc := application.NewCompensationPackageService(compRepo)
	calc := calculator.New()
	payrollSvc := application.NewPayrollService(periodRepo, resultRepo, empRepo, compRepo, calc)
	auditSvc := application.NewAuditLogService(auditRepo)

	// 6. Wire the App container.
	app := &tui.App{
		Config:              cfg,
		OrganizationService: orgSvc,
		EmployeeService:     empSvc,
		CompensationService: compSvc,
		PayrollService:      payrollSvc,
		AuditLogService:     auditSvc,
	}

	// 7. Start the TUI.
	p := tea.NewProgram(tui.NewModel(app), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatal("fatal: TUI error:", err)
	}
}
