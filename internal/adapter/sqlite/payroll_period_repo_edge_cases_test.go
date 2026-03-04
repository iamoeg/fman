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

func TestPayrollPeriodRepository_Concurrency(t *testing.T) {
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

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err = orgRepo.Create(ctx, org)
		require.NoError(t, err)

		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for range numGoroutines {
			go func() {
				defer wg.Done()
				period := createTestPayrollPeriod(org.ID)
				errChan <- repo.Create(ctx, period)
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err)
		}

		periods, err := repo.FindByOrganization(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, periods, numGoroutines)
	})
}

// ============================================================================
// Transaction Rollback Tests
// ============================================================================

func TestPayrollPeriodRepository_TransactionRollback(t *testing.T) {
	t.Parallel()

	t.Run("failed operation does not persist partial data", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create first period successfully
		period1 := createTestPayrollPeriod(org.ID)
		period1.Year = 2024
		period1.Month = 6
		err = repo.Create(ctx, period1)
		require.NoError(t, err)

		// Try to create duplicate (should fail on UNIQUE constraint)
		period2 := createTestPayrollPeriod(org.ID)
		period2.Year = 2024
		period2.Month = 6 // Same year/month/org = UNIQUE violation
		err = repo.Create(ctx, period2)
		require.Error(t, err)

		// Verify period2 was not created (transaction rolled back)
		found, err := repo.FindByID(ctx, period2.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
		require.Nil(t, found)

		// Verify no audit log for period2 (rolled back with transaction)
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			period2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "audit log should not exist for failed create")

		// Verify period1 still exists and only period1 has audit log
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			period1.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "period1 should have audit log")
	})
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestPayrollPeriodRepository_Timestamps(t *testing.T) {
	t.Parallel()

	t.Run("created_at and updated_at are set on create", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, period.ID)
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

	t.Run("updated_at changes on finalize", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		originalUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)

		// Wait to ensure timestamp difference
		time.Sleep(2 * time.Second)

		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		found, err = repo.FindByID(ctx, period.ID)
		require.NoError(t, err)

		newUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)
		require.True(t, newUpdatedAt.After(originalUpdatedAt),
			"UpdatedAt %v should be after original %v", newUpdatedAt, originalUpdatedAt)
	})

	t.Run("deleted_at is set on delete", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
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

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, period.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("finalized_at is set on finalize", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.NotNil(t, found.FinalizedAt)

		finalizedAt := found.FinalizedAt.UTC().Truncate(time.Second)
		require.True(t, finalizedAt.Equal(before) || finalizedAt.After(before),
			"FinalizedAt %v should be >= %v", finalizedAt, before)
		require.True(t, finalizedAt.Equal(after) || finalizedAt.Before(after),
			"FinalizedAt %v should be <= %v", finalizedAt, after)
	})

	t.Run("finalized_at is cleared on unfinalize", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		err = repo.Unfinalize(ctx, period.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Nil(t, found.FinalizedAt)
	})
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestPayrollPeriodRepository_AuditLog(t *testing.T) {
	t.Parallel()

	t.Run("audit log contains valid JSON", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		var after string
		err = db.QueryRow(
			"SELECT after FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'CREATE'",
			period.ID.String(),
		).Scan(&after)
		require.NoError(t, err)

		// Verify it's valid JSON
		require.Contains(t, after, period.ID.String())
		require.Contains(t, after, "DRAFT")
	})

	t.Run("finalize audit log has before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		var before, after sql.NullString
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'UPDATE' ORDER BY timestamp DESC LIMIT 1",
			period.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, "DRAFT")
		require.Contains(t, after.String, "FINALIZED")
	})

	t.Run("hard delete audit log has before but no after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, period.ID)
		require.NoError(t, err)

		var before sql.NullString
		var after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'HARD_DELETE'",
			period.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, period.ID.String())
		require.Equal(t, "null", after) // Should be "null" for HARD_DELETE
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestPayrollPeriodRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create a database and close it
		db := setupTestDB(t)
		db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		period := createTestPayrollPeriod(uuid.New())
		err := repo.Create(ctx, period)
		require.Error(t, err)
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestPayrollPeriodRepository_DataIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("preserves all field values on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		period.Year = 2025
		period.Month = 11
		period.Status = domain.PayrollPeriodStatusDraft

		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)

		require.Equal(t, period.ID, found.ID)
		require.Equal(t, period.OrgID, found.OrgID)
		require.Equal(t, period.Year, found.Year)
		require.Equal(t, period.Month, found.Month)
		require.Equal(t, period.Status, found.Status)
		require.Nil(t, found.FinalizedAt)
	})

	t.Run("finalized_at is null for draft periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, domain.PayrollPeriodStatusDraft, found.Status)
		require.Nil(t, found.FinalizedAt)
	})

	t.Run("finalized_at is not null for finalized periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, domain.PayrollPeriodStatusFinalized, found.Status)
		require.NotNil(t, found.FinalizedAt)
	})
}

// ============================================================================
// Foreign Key Cascade Tests
// ============================================================================

func TestPayrollPeriodRepository_ForeignKeyCascade(t *testing.T) {
	t.Parallel()

	t.Run("deleting organization cascades to payroll periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		// Hard delete organization (CASCADE)
		err = orgRepo.HardDelete(ctx, org.ID)
		require.NoError(t, err)

		// Period should also be deleted
		found, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// Year/Month Boundary Tests
// ============================================================================

func TestPayrollPeriodRepository_YearMonthBoundaries(t *testing.T) {
	t.Parallel()

	t.Run("handles minimum valid year and month", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		period.Year = 2020 // Minimum from CHECK constraint
		period.Month = 1   // Minimum month

		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, 2020, found.Year)
		require.Equal(t, 1, found.Month)
	})

	t.Run("handles maximum valid year and month", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		period := createTestPayrollPeriod(org.ID)
		period.Year = 2050 // Maximum from CHECK constraint
		period.Month = 12  // Maximum month

		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, 2050, found.Year)
		require.Equal(t, 12, found.Month)
	})
}
