package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/iamoeg/bootdev-capstone/db/migration"
	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestPayrollResultRepository_Concurrency(t *testing.T) {
	t.Parallel()

	t.Run("concurrent creates do not conflict", func(t *testing.T) {
		// Use a file-based database for concurrency test
		dbPath := filepath.Join(t.TempDir(), "test.db")
		db, err := sql.Open("sqlite3", dbPath)
		require.NoError(t, err)
		defer db.Close()

		// Enable foreign keys and WAL mode for better concurrency
		_, err = db.Exec("PRAGMA foreign_keys = ON")
		require.NoError(t, err)
		_, err = db.Exec("PRAGMA journal_mode=WAL")
		require.NoError(t, err)

		// Run migrations
		err = goose.SetDialect("sqlite3")
		require.NoError(t, err)
		goose.SetBaseFS(migration.FS)
		err = goose.Up(db, ".")
		require.NoError(t, err)

		// Setup test data
		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		// Create employees for concurrent payroll results
		employeeIDs := make([]string, numGoroutines)
		for i := range numGoroutines {
			emp := createTestEmployee(orgID, compID, i+10) // Start from 10 to avoid conflicts
			err := empRepo.Create(ctx, emp)
			require.NoError(t, err)
			employeeIDs[i] = emp.ID.String()
		}

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(empIDStr string) {
				defer wg.Done()
				result := createTestPayrollResult(periodID, mustParseUUID(empIDStr), compID)
				errChan <- repo.Create(ctx, result)
			}(employeeIDs[i])
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err)
		}

		results, err := repo.FindByPeriod(ctx, periodID)
		require.NoError(t, err)
		require.Len(t, results, numGoroutines)
	})
}

// ============================================================================
// Transaction Rollback Tests
// ============================================================================

func TestPayrollResultRepository_TransactionRollback(t *testing.T) {
	t.Parallel()

	t.Run("failed operation does not persist partial data", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		// Create first result successfully
		result1 := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result1)
		require.NoError(t, err)

		// Try to create duplicate (should fail on UNIQUE constraint)
		result2 := createTestPayrollResult(periodID, empID, compID)
		err = repo.Create(ctx, result2)
		require.Error(t, err, "duplicate (period, employee) should fail")

		// Verify result2 was not created (transaction rolled back)
		found, err := repo.FindByID(ctx, result2.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
		require.Nil(t, found)

		// Verify no audit log for result2 (rolled back with transaction)
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			result2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "audit log should not exist for failed create")

		// Verify result1 still exists and only result1 has audit log
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			result1.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "result1 should have audit log")
	})
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestPayrollResultRepository_Timestamps(t *testing.T) {
	t.Parallel()

	t.Run("created_at and updated_at are set on create", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		before := time.Now().UTC().Truncate(time.Second)

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)

		// Convert to UTC and truncate to second precision
		createdAt := found.CreatedAt.UTC().Truncate(time.Second)
		updatedAt := found.UpdatedAt.UTC().Truncate(time.Second)

		// Timestamps should be within the time window
		require.True(t, createdAt.Equal(before) || createdAt.After(before),
			"CreatedAt %v should be >= %v", createdAt, before)
		require.True(t, createdAt.Equal(after) || createdAt.Before(after),
			"CreatedAt %v should be <= %v", createdAt, after)
		require.True(t, updatedAt.Equal(before) || updatedAt.After(before),
			"UpdatedAt %v should be >= %v", updatedAt, before)
		require.True(t, updatedAt.Equal(after) || updatedAt.Before(after),
			"UpdatedAt %v should be <= %v", updatedAt, after)
	})

	t.Run("deleted_at is set on delete", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByIDIncludingDeleted(ctx, result.ID)
		require.NoError(t, err)
		require.NotNil(t, found.DeletedAt)

		deletedAt := found.DeletedAt.UTC().Truncate(time.Second)
		require.True(t, deletedAt.Equal(before) || deletedAt.After(before),
			"DeletedAt %v should be >= %v", deletedAt, before)
		require.True(t, deletedAt.Equal(after) || deletedAt.Before(after),
			"DeletedAt %v should be <= %v", deletedAt, after)
	})

	t.Run("deleted_at is cleared on restore", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, result.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestPayrollResultRepository_AuditLog(t *testing.T) {
	t.Parallel()

	t.Run("audit log contains valid JSON", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		var after string
		err = db.QueryRow(
			"SELECT after FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'CREATE'",
			result.ID.String(),
		).Scan(&after)
		require.NoError(t, err)

		// Verify it's valid JSON and contains expected fields
		require.Contains(t, after, result.ID.String())
		require.Contains(t, after, result.EmployeeID.String())
		require.Contains(t, after, result.PayrollPeriodID.String())
	})

	t.Run("delete audit log has before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		var before, after sql.NullString
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'DELETE'",
			result.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, result.ID.String())
		require.True(t, after.Valid)
		require.Contains(t, after.String, result.ID.String())
	})

	t.Run("hard delete audit log has before but no after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, result.ID)
		require.NoError(t, err)

		var before sql.NullString
		var after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'HARD_DELETE'",
			result.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, result.ID.String())
		require.Equal(t, "null", after) // Should be "null" for HARD_DELETE
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestPayrollResultRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create a database and close it
		db := setupTestDB(t)
		_, empID, compID, periodID := setupPayrollTestData(t, db)
		db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.Error(t, err)
	})

	t.Run("returns ErrRecordNotFound for operations on non-existent records", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		nonExistentID := mustParseUUID("00000000-0000-0000-0000-000000000001")

		// Delete
		err := repo.Delete(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)

		// Restore
		err = repo.Restore(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)

		// HardDelete
		err = repo.HardDelete(ctx, nonExistentID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestPayrollResultRepository_DataIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("preserves all money field values on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)

		// Verify all monetary values are preserved
		require.True(t, result.BaseSalary.Equals(found.BaseSalary))
		require.True(t, result.SeniorityBonus.Equals(found.SeniorityBonus))
		require.True(t, result.GrossSalary.Equals(found.GrossSalary))
		require.True(t, result.TotalOtherBonus.Equals(found.TotalOtherBonus))
		require.True(t, result.GrossSalaryGrandTotal.Equals(found.GrossSalaryGrandTotal))
		require.True(t, result.TotalExemptions.Equals(found.TotalExemptions))
		require.True(t, result.TaxableGrossSalary.Equals(found.TaxableGrossSalary))
		require.True(t, result.SocialAllowanceEmployeeContrib.Equals(found.SocialAllowanceEmployeeContrib))
		require.True(t, result.SocialAllowanceEmployerContrib.Equals(found.SocialAllowanceEmployerContrib))
		require.True(t, result.JobLossCompensationEmployeeContrib.Equals(found.JobLossCompensationEmployeeContrib))
		require.True(t, result.JobLossCompensationEmployerContrib.Equals(found.JobLossCompensationEmployerContrib))
		require.True(t, result.TrainingTaxEmployerContrib.Equals(found.TrainingTaxEmployerContrib))
		require.True(t, result.FamilyBenefitsEmployerContrib.Equals(found.FamilyBenefitsEmployerContrib))
		require.True(t, result.TotalCNSSEmployeeContrib.Equals(found.TotalCNSSEmployeeContrib))
		require.True(t, result.TotalCNSSEmployerContrib.Equals(found.TotalCNSSEmployerContrib))
		require.True(t, result.AMOEmployeeContrib.Equals(found.AMOEmployeeContrib))
		require.True(t, result.AMOEmployerContrib.Equals(found.AMOEmployerContrib))
		require.True(t, result.TaxableNetSalary.Equals(found.TaxableNetSalary))
		require.True(t, result.IncomeTax.Equals(found.IncomeTax))
		require.True(t, result.RoundingAmount.Equals(found.RoundingAmount))
		require.True(t, result.NetToPay.Equals(found.NetToPay))
	})

	t.Run("stores money values as integer cents in database", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		baseSalary, _ := money.FromMAD(12345.67)
		result := createTestPayrollResult(periodID, empID, compID)
		result.BaseSalary = baseSalary
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Query database directly to verify cents storage
		var baseSalaryCents int64
		err = db.QueryRow(
			"SELECT base_salary_cents FROM payroll_result WHERE id = ?",
			result.ID.String(),
		).Scan(&baseSalaryCents)
		require.NoError(t, err)

		// 12345.67 MAD = 1234567 cents
		require.Equal(t, int64(1234567), baseSalaryCents)
		require.Equal(t, baseSalary.Cents(), baseSalaryCents)
	})

	t.Run("handles extreme money values correctly", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		// Set some extreme values (but within int64 range)
		result.BaseSalary = money.FromCents(999999999999) // ~10 billion MAD
		result.NetToPay = money.FromCents(1)              // 0.01 MAD
		result.RoundingAmount = money.FromCents(-100)     // -1.00 MAD

		err := repo.Create(ctx, result)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)

		require.True(t, result.BaseSalary.Equals(found.BaseSalary))
		require.True(t, result.NetToPay.Equals(found.NetToPay))
		require.True(t, result.RoundingAmount.Equals(found.RoundingAmount))
	})

	t.Run("preserves foreign key relationships", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)

		require.Equal(t, result.PayrollPeriodID, found.PayrollPeriodID)
		require.Equal(t, result.EmployeeID, found.EmployeeID)
		require.Equal(t, result.CompensationPackageID, found.CompensationPackageID)
	})

	t.Run("preserves currency type", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		result.Currency = money.MAD
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)

		require.Equal(t, money.MAD, found.Currency)
	})
}

// ============================================================================
// UNIQUE Constraint Tests
// ============================================================================

func TestPayrollResultRepository_UniqueConstraints(t *testing.T) {
	t.Parallel()

	t.Run("prevents duplicate (payroll_period_id, employee_id)", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		// Create first result
		result1 := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result1)
		require.NoError(t, err)

		// Try to create another result for same period and employee
		result2 := createTestPayrollResult(periodID, empID, compID)
		err = repo.Create(ctx, result2)
		require.Error(t, err, "duplicate (period, employee) should fail")
	})

	t.Run("allows same employee in different periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create two different periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))

		// Create results for same employee in different periods (should succeed)
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)

		err := repo.Create(ctx, result1)
		require.NoError(t, err)

		err = repo.Create(ctx, result2)
		require.NoError(t, err)
	})

	t.Run("allows different employees in same period", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create two different employees
		emp1 := createTestEmployee(orgID, compID, 100)
		emp2 := createTestEmployee(orgID, compID, 101)
		require.NoError(t, empRepo.Create(ctx, emp1))
		require.NoError(t, empRepo.Create(ctx, emp2))

		// Create results for different employees in same period (should succeed)
		result1 := createTestPayrollResult(periodID, emp1.ID, compID)
		result2 := createTestPayrollResult(periodID, emp2.ID, compID)

		err := repo.Create(ctx, result1)
		require.NoError(t, err)

		err = repo.Create(ctx, result2)
		require.NoError(t, err)
	})
}

// ============================================================================
// Query Performance Tests
// ============================================================================

func TestPayrollResultRepository_QueryPerformance(t *testing.T) {
	t.Parallel()

	t.Run("FindByPeriod handles large result sets", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create 100 employees
		const numEmployees = 100
		for i := range numEmployees {
			emp := createTestEmployee(orgID, compID, i+1000)
			err := empRepo.Create(ctx, emp)
			require.NoError(t, err)

			// Create payroll result for each
			result := createTestPayrollResult(periodID, emp.ID, compID)
			err = repo.Create(ctx, result)
			require.NoError(t, err)
		}

		// Query should return all results efficiently
		results, err := repo.FindByPeriod(ctx, periodID)
		require.NoError(t, err)
		require.Len(t, results, numEmployees)
	})

	t.Run("FindByEmployee handles employee payroll history", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 12 periods (one year of payroll)
		const numPeriods = 12
		for range numPeriods {
			period := createTestPayrollPeriod(orgID)
			err := periodRepo.Create(ctx, period)
			require.NoError(t, err)

			// Create payroll result for this period
			result := createTestPayrollResult(period.ID, empID, compID)
			err = repo.Create(ctx, result)
			require.NoError(t, err)
		}

		// Query should return all results for this employee
		results, err := repo.FindByEmployee(ctx, empID)
		require.NoError(t, err)
		require.Len(t, results, numPeriods)
	})
}

// ============================================================================
// Helper Functions
// ============================================================================

func mustParseUUID(s string) uuid.UUID {
	id, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return id
}
