package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	_ "github.com/mattn/go-sqlite3"

	"github.com/iamoeg/bootdev-capstone/db/migration"
	calculator "github.com/iamoeg/bootdev-capstone/internal/adapter/payroll"
	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/application"
	"github.com/iamoeg/bootdev-capstone/pkg/config"
	"github.com/iamoeg/bootdev-capstone/ui/tui"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "finmgmt",
	Short: "Moroccan payroll manager",
	RunE:  runTUI,
}

func init() {
	rootCmd.PersistentFlags().StringVar(
		&cfgPath, "config", "",
		"path to config file (default: ~/.config/finmgmt/config.yaml)",
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Debug logging — set DEBUG=1 to enable.
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			return fmt.Errorf("could not open debug log: %w", err)
		}
		defer f.Close()
	}

	// 1. Load (or create) config.
	cfg, err := config.LoadOrCreate(cfgPath)
	if err != nil {
		return fmt.Errorf("could not load config: %w", err)
	}

	// 2. Resolve database path and open SQLite.
	dbPath, err := cfg.ResolveDatabasePath()
	if err != nil {
		return fmt.Errorf("could not resolve database path: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return fmt.Errorf("could not open database: %w", err)
	}
	defer db.Close()

	// CRITICAL: enable foreign key enforcement on every connection.
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("could not enable foreign keys: %w", err)
	}

	// 3. Run migrations.
	if err := migration.RunMigrations(db); err != nil {
		return fmt.Errorf("could not run migrations: %w", err)
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
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}
