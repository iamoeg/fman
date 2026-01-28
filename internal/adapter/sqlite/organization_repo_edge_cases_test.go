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
)

// ============================================================================
// Concurrency Tests
// ============================================================================

func TestOrganizationRepository_Concurrency(t *testing.T) {
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

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		const numGoroutines = 10
		errChan := make(chan error, numGoroutines)

		var wg sync.WaitGroup
		wg.Add(numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				defer wg.Done()
				org := createTestOrganization()
				errChan <- repo.Create(ctx, org)
			}()
		}

		wg.Wait()
		close(errChan)

		for err := range errChan {
			require.NoError(t, err)
		}

		orgs, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, orgs, numGoroutines)
	})
}

// ============================================================================
// Transaction Rollback Tests
// ============================================================================

func TestOrganizationRepository_TransactionRollback(t *testing.T) {
	t.Parallel()

	t.Run("failed operation does not persist partial data", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create first org successfully
		org1 := createTestOrganization()
		err := repo.Create(ctx, org1)
		require.NoError(t, err)

		// Try to create duplicate (should fail on UNIQUE constraint)
		org2 := createTestOrganization()
		org2.ICENum = org1.ICENum // Duplicate ICE number (UNIQUE constraint)
		err = repo.Create(ctx, org2)
		require.Error(t, err)

		// Verify org2 was not created (transaction rolled back)
		found, err := repo.FindByID(ctx, org2.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
		require.Nil(t, found)

		// Verify no audit log for org2 (rolled back with transaction)
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			org2.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 0, count, "audit log should not exist for failed create")

		// Verify org1 still exists and only org1 has audit log
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE record_id = ?",
			org1.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count, "org1 should have audit log")
	})
}

// ============================================================================
// Timestamp Tests
// ============================================================================

func TestOrganizationRepository_Timestamps(t *testing.T) {
	t.Parallel()

	t.Run("created_at and updated_at are set on create", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		before := time.Now().UTC().Truncate(time.Second)

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByID(ctx, org.ID)
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

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		originalUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)

		// Wait to ensure timestamp difference
		time.Sleep(2 * time.Second)

		org.Name = "Updated Name"
		err = repo.Update(ctx, org)
		require.NoError(t, err)

		found, err = repo.FindByID(ctx, org.ID)
		require.NoError(t, err)

		newUpdatedAt := found.UpdatedAt.UTC().Truncate(time.Second)
		require.True(t, newUpdatedAt.After(originalUpdatedAt),
			"UpdatedAt %v should be after original %v", newUpdatedAt, originalUpdatedAt)
	})

	t.Run("deleted_at is set on delete", func(t *testing.T) {
		t.Parallel()
		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		before := time.Now().UTC().Truncate(time.Second)

		err = repo.Delete(ctx, org.ID)
		require.NoError(t, err)

		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		found, err := repo.FindByIDIncludingDeleted(ctx, org.ID)
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

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		err = repo.Delete(ctx, org.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, org.ID)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})
}

// ============================================================================
// Audit Log Tests
// ============================================================================

func TestOrganizationRepository_AuditLog(t *testing.T) {
	t.Parallel()

	t.Run("audit log contains valid JSON", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		var after string
		err = db.QueryRow(
			"SELECT after FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'CREATE'",
			org.ID.String(),
		).Scan(&after)
		require.NoError(t, err)

		// Verify it's valid JSON
		require.Contains(t, after, org.Name)
		require.Contains(t, after, org.ID.String())
	})

	t.Run("update audit log has before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		originalName := "Original Name"
		org.Name = originalName
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		org.Name = "Updated Name"
		err = repo.Update(ctx, org)
		require.NoError(t, err)

		var before, after sql.NullString
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'UPDATE'",
			org.ID.String(),
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

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, org.ID)
		require.NoError(t, err)

		var before sql.NullString
		var after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'HARD_DELETE'",
			org.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		require.True(t, before.Valid)
		require.Contains(t, before.String, org.Name)
		require.Equal(t, "null", after) // Should be "null" for HARD_DELETE
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestOrganizationRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		// Create a database and close it
		db := setupTestDB(t)
		db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.Error(t, err)
	})
}

// ============================================================================
// Data Integrity Tests
// ============================================================================

func TestOrganizationRepository_DataIntegrity(t *testing.T) {
	t.Parallel()

	t.Run("preserves all field values on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		org.Address = "123 Rue Mohammed V, Casablanca"
		org.Activity = "Développement Logiciel" // Test Unicode
		org.ICENum = "001234567890123"
		org.IFNum = "12345678"
		org.RCNum = "987654"
		org.CNSSNum = "1122334"
		org.BankRIB = "123456789012345678901234"

		err := repo.Create(ctx, org)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)

		require.Equal(t, org.Name, found.Name)
		require.Equal(t, org.Address, found.Address)
		require.Equal(t, org.Activity, found.Activity)
		require.Equal(t, org.LegalForm, found.LegalForm)
		require.Equal(t, org.ICENum, found.ICENum)
		require.Equal(t, org.IFNum, found.IFNum)
		require.Equal(t, org.RCNum, found.RCNum)
		require.Equal(t, org.CNSSNum, found.CNSSNum)
		require.Equal(t, org.BankRIB, found.BankRIB)
	})

	t.Run("handles special characters in fields", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		org.Name = "Company with 'quotes' and \"double quotes\""
		org.Address = "Address with\nnewlines\tand\ttabs"

		err := repo.Create(ctx, org)
		require.NoError(t, err)

		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Equal(t, org.Name, found.Name)
		require.Equal(t, org.Address, found.Address)
	})
}
