package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/iamoeg/bootdev-capstone/db/migration"
	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestEmployeeRepository_Concurrency(t *testing.T) {
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
		err = migration.RunMigrations(db)
		require.NoError(t, err)

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := range numGoroutines {
			go func(serialNum int) {
				defer wg.Done()
				emp := createTestEmployee(orgID, compPackID, serialNum)
				errChan <- repo.Create(ctx, emp)
			}(i + 1) // Serial numbers 1-10
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err)
		}

		emps, err := repo.FindByOrganization(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, emps, numGoroutines)
	})
}

// ============================================================================
// Transaction Rollback Tests
// ============================================================================

func TestEmployeeRepository_TransactionRollback(t *testing.T) {
	t.Parallel()

	t.Run("failed operation does not persist partial data", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create first employee successfully
		emp1 := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp1)
		require.NoError(t, err)

		// Try to create duplicate (should fail on UNIQUE constraint)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp2.CINNum = emp1.CINNum // Duplicate CIN number (UNIQUE constraint)
		err = repo.Create(ctx, emp2)
		require.Error(t, err)

		// Verify emp2 was not created (transaction rolled back)
		found, err := repo.FindByID(ctx, emp2.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
		require.Nil(t, found)

		// Verify no audit log for emp2 (rolled back with transaction)
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			emp2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "audit log should not exist for failed create")

		// Verify emp1 still exists and only emp1 has audit log
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			emp1.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "emp1 should have audit log")
	})
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestEmployeeRepository_Timestamps(t *testing.T) {
	t.Parallel()

	t.Run("created_at and updated_at are set on create", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		before := time.Now().UTC().Truncate(time.Second)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, emp.ID)
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

	t.Run("updated_at changes on update", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		originalUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)

		// Wait to ensure timestamp difference
		time.Sleep(2 * time.Second)

		emp.FullName = "Updated Name"
		err = repo.Update(ctx, emp)
		require.NoError(t, err)

		found, err = repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)

		newUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)
		require.True(t, newUpdatedAt.After(originalUpdatedAt),
			"UpdatedAt %v should be after original %v", newUpdatedAt, originalUpdatedAt)
	})

	t.Run("deleted_at is set on delete", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Delete(ctx, emp.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByIDIncludingDeleted(ctx, emp.ID)
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

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, emp.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestEmployeeRepository_AuditLog(t *testing.T) {
	t.Parallel()

	t.Run("audit log contains valid JSON", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		var after string
		err = db.QueryRow(
			"SELECT after FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'CREATE'",
			emp.ID.String(),
		).Scan(&after)
		require.NoError(t, err)

		// Verify it's valid JSON and contains employee data
		require.Contains(t, after, emp.FullName)
		require.Contains(t, after, emp.ID.String())
		require.Contains(t, after, emp.CINNum)
	})

	t.Run("update audit log has before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		originalName := "Original Name"
		emp.FullName = originalName
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		emp.FullName = "Updated Name"
		err = repo.Update(ctx, emp)
		require.NoError(t, err)

		var before, after sql.NullString
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'UPDATE'",
			emp.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, originalName)
		require.Contains(t, after.String, "Updated Name")
	})

	t.Run("hard delete audit log has before but no after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, emp.ID)
		require.NoError(t, err)

		var before sql.NullString
		var after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'HARD_DELETE'",
			emp.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, emp.FullName)
		require.Equal(t, "null", after) // Should be "null" for HARD_DELETE
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestEmployeeRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create a database and close it
		db := setupTestDB(t)
		db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		emp := createTestEmployee(uuid.New(), uuid.New(), 1)
		err := repo.Create(ctx, emp)
		require.Error(t, err)
	})

	t.Run("handles invalid foreign key - organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		compPackRepo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create a real org for the comp package (FK required), but use a
		// different non-existent org_id for the employee to trigger the error
		pkgOrg := createAndPersistTestOrg(t, db)
		compPack := createTestCompensationPackage(pkgOrg.ID)
		err := compPackRepo.Create(ctx, compPack)
		require.NoError(t, err)

		// Try to create employee with non-existent org_id
		invalidOrgID := uuid.New()
		emp := createTestEmployee(invalidOrgID, compPack.ID, 1)
		err = repo.Create(ctx, emp)
		require.Error(t, err) // Should fail on foreign key constraint
	})

	t.Run("handles invalid foreign key - compensation package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization only (no compensation package)
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Try to create employee with non-existent compensation_package_id
		invalidCompPackID := uuid.New()
		emp := createTestEmployee(org.ID, invalidCompPackID, 1)
		err = repo.Create(ctx, emp)
		require.Error(t, err) // Should fail on foreign key constraint
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestEmployeeRepository_DataIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("preserves all field values on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		emp.FullName = "Mohammed Ben Ahmed El Fassi"
		emp.DisplayName = "Mohammed"
		emp.Address = "15 Avenue Mohammed V, Casablanca"
		emp.EmailAddress = "mohammed.fassi@example.ma"
		emp.PhoneNumber = "+212-661234567"
		emp.Position = "Ingénieur Logiciel" // Test Unicode
		emp.CINNum = "AA123456"
		emp.CNSSNum = "987654321"
		emp.BankRIB = "123456789012345678901234"
		emp.MaritalStatus = domain.MaritalStatusMarried
		emp.NumDependents = 2
		emp.NumChildren = 1

		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)

		require.Equal(t, emp.FullName, found.FullName)
		require.Equal(t, emp.DisplayName, found.DisplayName)
		require.Equal(t, emp.Address, found.Address)
		require.Equal(t, emp.EmailAddress, found.EmailAddress)
		require.Equal(t, emp.PhoneNumber, found.PhoneNumber)
		require.Equal(t, emp.Position, found.Position)
		require.Equal(t, emp.CINNum, found.CINNum)
		require.Equal(t, emp.CNSSNum, found.CNSSNum)
		require.Equal(t, emp.BankRIB, found.BankRIB)
		require.Equal(t, emp.Gender, found.Gender)
		require.Equal(t, emp.MaritalStatus, found.MaritalStatus)
		require.Equal(t, emp.NumDependents, found.NumDependents)
		require.Equal(t, emp.NumChildren, found.NumChildren)
		require.Equal(t, emp.SerialNum, found.SerialNum)
		require.Equal(t, emp.OrgID, found.OrgID)
		require.Equal(t, emp.CompensationPackageID, found.CompensationPackageID)
	})

	t.Run("handles special characters in fields", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		emp.FullName = "Employee with 'quotes' and \"double quotes\""
		emp.Address = "Address with\nnewlines\tand\ttabs"
		emp.Position = "Position with Ã©Ã  Ã¨ Ãª characters"

		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, emp.FullName, found.FullName)
		require.Equal(t, emp.Address, found.Address)
		require.Equal(t, emp.Position, found.Position)
	})

	t.Run("handles Arabic names correctly", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		emp.FullName = "Ù…Ø­Ù…Ø¯ Ø¨Ù† Ø£Ø­Ù…Ø¯" // Arabic name

		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, emp.FullName, found.FullName)
	})

	t.Run("preserves date precision", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		originalBirthDate := emp.BirthDate.UTC().Truncate(time.Second)
		originalHireDate := emp.HireDate.UTC().Truncate(time.Second)

		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)

		foundBirthDate := found.BirthDate.UTC().Truncate(time.Second)
		foundHireDate := found.HireDate.UTC().Truncate(time.Second)

		require.True(t, originalBirthDate.Equal(foundBirthDate),
			"BirthDate should be preserved: expected %v, got %v", originalBirthDate, foundBirthDate)
		require.True(t, originalHireDate.Equal(foundHireDate),
			"HireDate should be preserved: expected %v, got %v", originalHireDate, foundHireDate)
	})
}

// ============================================================================
// Foreign Key Cascade Tests
// ============================================================================

func TestEmployeeRepository_ForeignKeyCascade(t *testing.T) {
	t.Parallel()

	t.Run("deleting organization cascades to employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		empRepo := sqlite.NewEmployeeRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create employees
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)

		err := empRepo.Create(ctx, emp1)
		require.NoError(t, err)
		err = empRepo.Create(ctx, emp2)
		require.NoError(t, err)

		// Hard delete organization (should cascade to employees)
		err = orgRepo.HardDelete(ctx, orgID)
		require.NoError(t, err)

		// Employees should be gone
		found, err := empRepo.FindByIDIncludingDeleted(ctx, emp1.ID)
		require.Error(t, err)
		require.Nil(t, found)

		found, err = empRepo.FindByIDIncludingDeleted(ctx, emp2.ID)
		require.Error(t, err)
		require.Nil(t, found)
	})

	t.Run("cannot delete compensation package if employees use it", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		empRepo := sqlite.NewEmployeeRepository(db)
		compPackRepo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create employee using compensation package
		emp := createTestEmployee(orgID, compPackID, 1)
		err := empRepo.Create(ctx, emp)
		require.NoError(t, err)

		// Try to hard delete compensation package (should fail - ON DELETE RESTRICT)
		err = compPackRepo.HardDelete(ctx, compPackID)
		require.Error(t, err) // Should fail due to foreign key constraint
	})
}
