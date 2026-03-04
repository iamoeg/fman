package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"

	"github.com/iamoeg/bootdev-capstone/db/migration"
	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

// setupTestDB creates an in-memory SQLite database and runs migrations.
func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)

	// Enable foreign keys
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	require.NoError(t, err)

	// Set goose dialect
	err = goose.SetDialect("sqlite3")
	require.NoError(t, err)

	// Run migrations
	goose.SetBaseFS(migration.FS)
	err = goose.Up(db, ".")
	require.NoError(t, err)

	return db
}

var orgCounter int64

// createTestOrganization creates a valid test organization with unique identifiers.
func createTestOrganization() *domain.Organization {
	now := time.Now().UTC()
	counter := atomic.AddInt64(&orgCounter, 1)

	return &domain.Organization{
		ID:        uuid.New(),
		Name:      fmt.Sprintf("Test Company SARL %d", counter),
		Address:   "123 Rue Mohammed V, Casablanca",
		Activity:  "Software Development",
		LegalForm: domain.LegalFormSARL,
		ICENum:    fmt.Sprintf("%015d", counter), // 15 digits
		IFNum:     fmt.Sprintf("%08d", counter),  // 8 digits
		RCNum:     fmt.Sprintf("%06d", counter),  // 6 digits
		CNSSNum:   fmt.Sprintf("%07d", counter),  // 7 digits
		BankRIB:   fmt.Sprintf("%024d", counter), // 24 digits
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// ============================================================================
// FindByID Tests
// ============================================================================

func TestOrganizationRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("returns organization when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Find it
		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, org.ID, found.ID)
		require.Equal(t, org.Name, found.Name)
		require.Equal(t, org.ICENum, found.ICENum)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByID(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create and delete organization
		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		err = repo.Delete(ctx, org.ID)
		require.NoError(t, err)

		// Should not find it
		found, err := repo.FindByID(ctx, org.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByIDIncludingDeleted Tests
// ============================================================================

func TestOrganizationRepository_FindByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns active organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, org.ID, found.ID)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("returns soft-deleted organization", func(t *testing.T) {
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

		found, err := repo.FindByIDIncludingDeleted(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, org.ID, found.ID)
		require.NotNil(t, found.DeletedAt) // Should have deleted_at set
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByIDIncludingDeleted(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindAll Tests
// ============================================================================

func TestOrganizationRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		orgs, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.NotNil(t, orgs)
		require.Len(t, orgs, 0)
	})

	t.Run("returns all active organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create 3 organizations
		org1 := createTestOrganization()
		org2 := createTestOrganization()
		org3 := createTestOrganization()

		err := repo.Create(ctx, org1)
		require.NoError(t, err)
		err = repo.Create(ctx, org2)
		require.NoError(t, err)
		err = repo.Create(ctx, org3)
		require.NoError(t, err)

		orgs, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, orgs, 3)
	})

	t.Run("does not return soft-deleted organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create 2 orgs, delete 1
		org1 := createTestOrganization()
		org2 := createTestOrganization()

		err := repo.Create(ctx, org1)
		require.NoError(t, err)
		err = repo.Create(ctx, org2)
		require.NoError(t, err)

		err = repo.Delete(ctx, org1.ID)
		require.NoError(t, err)

		orgs, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, orgs, 1)
		require.Equal(t, org2.ID, orgs[0].ID)
	})
}

// ============================================================================
// FindAllIncludingDeleted Tests
// ============================================================================

func TestOrganizationRepository_FindAllIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all organizations including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create 3 orgs, delete 1
		org1 := createTestOrganization()
		org2 := createTestOrganization()
		org3 := createTestOrganization()

		err := repo.Create(ctx, org1)
		require.NoError(t, err)
		err = repo.Create(ctx, org2)
		require.NoError(t, err)
		err = repo.Create(ctx, org3)
		require.NoError(t, err)

		err = repo.Delete(ctx, org1.ID)
		require.NoError(t, err)

		orgs, err := repo.FindAllIncludingDeleted(ctx)
		require.NoError(t, err)
		require.Len(t, orgs, 3) // Should include deleted one
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestOrganizationRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates organization successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()

		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Verify it was created
		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Equal(t, org.Name, found.Name)
		require.Equal(t, org.ICENum, found.ICENum)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()

		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Check audit log exists
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'CREATE'",
			org.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("handles nullable fields correctly", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		org.Address = ""   // Empty string should become NULL
		org.Activity = ""  // Empty string should become NULL
		org.ICENum = ""    // Empty string should become NULL
		org.IFNum = ""     // Empty string should become NULL
		org.RCNum = ""     // Empty string should become NULL
		org.CNSSNum = ""   // Empty string should become NULL
		org.BankRIB = ""   // Empty string should become NULL
		org.LegalForm = "" // Empty string should become NULL

		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Verify empty strings were stored as NULL (retrieved as empty strings)
		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Equal(t, "", found.Address)
		require.Equal(t, "", found.Activity)
		require.Equal(t, "", found.ICENum)
	})
}

// ============================================================================
// Update Tests
// ============================================================================

func TestOrganizationRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates organization successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create
		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Update
		org.Name = "Updated Company Name"
		org.Address = "New Address"
		err = repo.Update(ctx, org)
		require.NoError(t, err)

		// Verify
		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Equal(t, "Updated Company Name", found.Name)
		require.Equal(t, "New Address", found.Address)
	})

	t.Run("creates audit log with before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create
		org := createTestOrganization()
		originalName := org.Name
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Update
		org.Name = "Changed Name"
		err = repo.Update(ctx, org)
		require.NoError(t, err)

		// Check audit log
		var before, after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'UPDATE'",
			org.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)
		require.Contains(t, before, originalName)
		require.Contains(t, after, "Changed Name")
	})

	t.Run("returns ErrRecordNotFound when organization does not exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Update(ctx, org)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("returns ErrRecordNotFound when organization is deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create and delete
		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)
		err = repo.Delete(ctx, org.ID)
		require.NoError(t, err)

		// Try to update
		org.Name = "Should Fail"
		err = repo.Update(ctx, org)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestOrganizationRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft deletes organization", func(t *testing.T) {
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

		// Should not be found by regular FindByID
		found, err := repo.FindByID(ctx, org.ID)
		require.Error(t, err)
		require.Nil(t, found)

		// Should be found with IncludingDeleted
		foundDeleted, err := repo.FindByIDIncludingDeleted(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, foundDeleted.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'DELETE'",
			org.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// Restore Tests
// ============================================================================

func TestOrganizationRepository_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores soft-deleted organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, org.ID)
		require.NoError(t, err)

		// Restore
		err = repo.Restore(ctx, org.ID)
		require.NoError(t, err)

		// Should be found now
		found, err := repo.FindByID(ctx, org.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'RESTORE'",
			org.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// HardDelete Tests
// ============================================================================

func TestOrganizationRepository_HardDelete(t *testing.T) {
	t.Parallel()

	t.Run("permanently deletes organization", func(t *testing.T) {
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

		// Should not be found even with IncludingDeleted
		found, err := repo.FindByIDIncludingDeleted(ctx, org.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("creates audit log entry before deletion", func(t *testing.T) {
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

		// Audit log should still exist
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'organization' AND record_id = ? AND action = 'HARD_DELETE'",
			org.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}
