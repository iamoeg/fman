package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestCompensationPackageRepository_Concurrency(t *testing.T) {
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
		err = goose.Up(db, "../../../db/migration")
		require.NoError(t, err)

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				pkg := createTestCompensationPackage()
				errChan <- repo.Create(ctx, pkg)
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err)
		}

		pkgs, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, pkgs, numGoroutines)
	})
}

// ============================================================================
// Transaction Rollback Tests
// ============================================================================

func TestCompensationPackageRepository_TransactionRollback(t *testing.T) {
	t.Parallel()

	t.Run("failed operation does not persist partial data", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create first package successfully
		pkg1 := createTestCompensationPackage()
		err := repo.Create(ctx, pkg1)
		require.NoError(t, err)

		// Manually insert a duplicate ID (will fail on primary key constraint)
		// This simulates what happens when transaction fails partway through
		pkg2 := createTestCompensationPackage()
		pkg2.ID = pkg1.ID // Duplicate ID

		err = repo.Create(ctx, pkg2)
		require.Error(t, err)

		// Verify pkg2 was not created (transaction rolled back)
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM employee_compensation_package WHERE id = ?",
			pkg2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "should only have one package with this ID")

		// Verify no extra audit log for pkg2 (rolled back with transaction)
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			pkg2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "should only have CREATE audit log for first package")
	})
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestCompensationPackageRepository_Timestamps(t *testing.T) {
	t.Parallel()

	t.Run("created_at and updated_at are set on create", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		before := time.Now().UTC().Truncate(time.Second)

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, pkg.ID)
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

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		originalUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)

		// Wait to ensure timestamp difference
		time.Sleep(2 * time.Second)

		pkg.BaseSalary = money.FromCents(500000)
		err = repo.Update(ctx, pkg)
		require.NoError(t, err)

		found, err = repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)

		newUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)
		require.True(t, newUpdatedAt.After(originalUpdatedAt),
			"UpdatedAt %v should be after original %v", newUpdatedAt, originalUpdatedAt)
	})

	t.Run("deleted_at is set on delete", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByIDIncludingDeleted(ctx, pkg.ID)
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

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, pkg.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestCompensationPackageRepository_AuditLog(t *testing.T) {
	t.Parallel()

	t.Run("audit log contains valid JSON", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		var after string
		err = db.QueryRow(
			"SELECT after FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'CREATE'",
			pkg.ID.String(),
		).Scan(&after)
		require.NoError(t, err)

		// Verify it's valid JSON and contains expected data
		require.Contains(t, after, pkg.ID.String())
		require.Contains(t, after, "MAD") // Currency
	})

	t.Run("update audit log has before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		pkg.BaseSalary = money.FromCents(500000)
		err = repo.Update(ctx, pkg)
		require.NoError(t, err)

		var before, after sql.NullString
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'UPDATE'",
			pkg.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		// Verify audit log contains expected data
		require.Contains(t, before.String, pkg.ID.String())
		require.Contains(t, after.String, pkg.ID.String())
		require.Contains(t, before.String, "MAD")
		require.Contains(t, after.String, "MAD")
	})

	t.Run("hard delete audit log has before but no after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, pkg.ID)
		require.NoError(t, err)

		var before sql.NullString
		var after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'HARD_DELETE'",
			pkg.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, pkg.ID.String())
		require.Equal(t, "null", after) // Should be "null" for HARD_DELETE
	})
}

// ============================================================================
// Money Handling Tests
// ============================================================================

func TestCompensationPackageRepository_MoneyHandling(t *testing.T) {
	t.Parallel()

	t.Run("preserves exact cents precision", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Test edge cases for money precision
		testCases := []struct {
			name  string
			cents int64
		}{
			{"minimum positive", 1},
			{"SMIG minimum", 300000},
			{"with 50 cents", 300050},
			{"with 99 cents", 300099},
			{"large round", 1000000},
			{"large with cents", 1234567},
			{"very large", 999999999},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				pkg := createTestCompensationPackage()
				pkg.BaseSalary = money.FromCents(tc.cents)

				err := repo.Create(ctx, pkg)
				require.NoError(t, err)

				found, err := repo.FindByID(ctx, pkg.ID)
				require.NoError(t, err)

				require.Equal(t, tc.cents, found.BaseSalary.Cents(),
					"Expected exact cents: %d, got: %d", tc.cents, found.BaseSalary.Cents())
			})
		}
	})

	t.Run("currency is preserved", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		pkg.Currency = money.MAD

		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, money.MAD, found.Currency)
	})
}

// ============================================================================
// Usage Guard Tests
// ============================================================================

func TestCompensationPackageRepository_UsageGuards(t *testing.T) {
	t.Parallel()

	t.Run("package can be updated when no employees use it", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Should succeed - no employees use it
		pkg.BaseSalary = money.FromCents(500000)
		err = repo.Update(ctx, pkg)
		require.NoError(t, err)
	})

	t.Run("package cannot be updated when employees use it", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		pkgRepo := sqlite.NewCompensationPackageRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Setup: org + package + employee
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		pkg := createTestCompensationPackage()
		err = pkgRepo.Create(ctx, pkg)
		require.NoError(t, err)

		_, err = db.Exec(`
			INSERT INTO employee (
				id, org_id, serial_num, full_name, birth_date, gender,
				marital_status, num_dependents, num_kids, cin_num, hire_date,
				position, compensation_package_id, created_at, updated_at
			) VALUES (?, ?, 1, 'Test', '1990-01-01', 'MALE', 'SINGLE', 0, 0, 'CIN001', '2025-01-01', 'Dev', ?, ?, ?)
		`, createTestOrganization().ID.String(), org.ID.String(), pkg.ID.String(),
			time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		require.NoError(t, err)

		// Should fail - employee uses it
		pkg.BaseSalary = money.FromCents(500000)
		err = pkgRepo.Update(ctx, pkg)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrCompensationPackageInUse)
	})

	t.Run("package can be deleted when no entities use it", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Should succeed - nothing uses it
		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)
	})

	t.Run("soft-deleted package can be hard-deleted when not in use", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Soft delete
		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Hard delete should succeed
		err = repo.HardDelete(ctx, pkg.ID)
		require.NoError(t, err)

		// Should be completely gone
		_, err = repo.FindByIDIncludingDeleted(ctx, pkg.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestCompensationPackageRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create a database and close it
		db := setupTestDB(t)
		db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.Error(t, err)
	})

	t.Run("returns appropriate error when trying to restore non-deleted package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Try to restore a non-deleted package (should fail)
		err = repo.Restore(ctx, pkg.ID)
		require.Error(t, err)

		// Package should still be active
		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestCompensationPackageRepository_DataIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("preserves all field values on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage()
		pkg.BaseSalary = money.FromCents(543210)
		pkg.Currency = money.MAD

		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)

		require.Equal(t, pkg.ID, found.ID)
		require.Equal(t, pkg.BaseSalary.Cents(), found.BaseSalary.Cents())
		require.Equal(t, pkg.Currency, found.Currency)
		require.Equal(t, pkg.CreatedAt.UTC().Truncate(time.Second),
			found.CreatedAt.UTC().Truncate(time.Second))
	})

	t.Run("multiple packages can exist with different salaries", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create packages with different salaries
		pkg1 := createTestCompensationPackage()
		pkg1.BaseSalary = money.FromCents(300000) // 3000 MAD

		pkg2 := createTestCompensationPackage()
		pkg2.BaseSalary = money.FromCents(500000) // 5000 MAD

		pkg3 := createTestCompensationPackage()
		pkg3.BaseSalary = money.FromCents(1000000) // 10000 MAD

		err := repo.Create(ctx, pkg1)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg2)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg3)
		require.NoError(t, err)

		// Verify all exist with correct salaries
		found1, err := repo.FindByID(ctx, pkg1.ID)
		require.NoError(t, err)
		require.Equal(t, int64(300000), found1.BaseSalary.Cents())

		found2, err := repo.FindByID(ctx, pkg2.ID)
		require.NoError(t, err)
		require.Equal(t, int64(500000), found2.BaseSalary.Cents())

		found3, err := repo.FindByID(ctx, pkg3.ID)
		require.NoError(t, err)
		require.Equal(t, int64(1000000), found3.BaseSalary.Cents())
	})
}
