package sqlite_adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
)

// ============================================================================
// FindForRecord Tests
// ============================================================================

func TestAuditLogRepository_FindForRecord(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no audit logs exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		logs, err := repo.FindForRecord(ctx, "organization", randomID.String())
		require.NoError(t, err)
		require.NotNil(t, logs)
		require.Len(t, logs, 0)
	})

	t.Run("returns all audit logs for a record", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create organization (generates CREATE audit log)
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		// Update organization (generates UPDATE audit log)
		org.Name = "Updated Name"
		err = orgRepo.Update(ctx, org)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		// Delete organization (generates DELETE audit log)
		err = orgRepo.Delete(ctx, org.ID)
		require.NoError(t, err)

		// Query audit logs for this organization
		logs, err := auditRepo.FindForRecord(ctx, "organization", org.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// All queries use DESC order - newest first
		require.Equal(t, "DELETE", logs[0].Action)
		require.Equal(t, "UPDATE", logs[1].Action)
		require.Equal(t, "CREATE", logs[2].Action)

		// Verify all logs reference the same record
		for _, log := range logs {
			require.Equal(t, "organization", log.TableName)
			require.Equal(t, org.ID.String(), log.RecordID)
		}
	})

	t.Run("does not return logs for different records", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create two organizations
		org1 := createTestOrganization()
		org2 := createTestOrganization()

		err := orgRepo.Create(ctx, org1)
		require.NoError(t, err)
		err = orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Query audit logs for org1 only
		logs, err := auditRepo.FindForRecord(ctx, "organization", org1.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 1)
		require.Equal(t, org1.ID.String(), logs[0].RecordID)
	})

	t.Run("returns logs for specific table only", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Query with wrong table name
		logs, err := auditRepo.FindForRecord(ctx, "employee", org.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 0)

		// Query with correct table name
		logs, err = auditRepo.FindForRecord(ctx, "organization", org.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 1)
	})

	t.Run("parses all fields correctly", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		logs, err := auditRepo.FindForRecord(ctx, "organization", org.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 1)

		log := logs[0]
		require.NotEmpty(t, log.ID)
		require.Equal(t, "organization", log.TableName)
		require.Equal(t, org.ID.String(), log.RecordID)
		require.Equal(t, "CREATE", log.Action)
		require.Empty(t, log.Before) // CREATE has no "before"
		require.NotEmpty(t, log.After)
		require.Contains(t, log.After, org.Name)
		require.False(t, log.Timestamp.IsZero())
	})
}

// ============================================================================
// FindRecent Tests
// ============================================================================

func TestAuditLogRepository_FindRecent(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no audit logs exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		logs, err := repo.FindRecent(ctx, 10)
		require.NoError(t, err)
		require.NotNil(t, logs)
		require.Len(t, logs, 0)
	})

	t.Run("returns most recent audit logs", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 5 organizations (5 CREATE audit logs)
		for i := 0; i < 5; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Get recent 3
		logs, err := auditRepo.FindRecent(ctx, 3)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// Verify they're all CREATE actions
		for _, log := range logs {
			require.Equal(t, "CREATE", log.Action)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 10 organizations
		for i := 0; i < 10; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Test different limits
		testCases := []struct {
			limit    int
			expected int
		}{
			{limit: 1, expected: 1},
			{limit: 5, expected: 5},
			{limit: 10, expected: 10},
			{limit: 20, expected: 10}, // Can't exceed actual count
		}

		for _, tc := range testCases {
			logs, err := auditRepo.FindRecent(ctx, tc.limit)
			require.NoError(t, err)
			require.Len(t, logs, tc.expected, "limit: %d", tc.limit)
		}
	})

	t.Run("returns logs in reverse chronological order", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		org.Name = "Updated"
		err = orgRepo.Update(ctx, org)
		require.NoError(t, err)

		logs, err := auditRepo.FindRecent(ctx, 10)
		require.NoError(t, err)
		require.Len(t, logs, 2)

		// DESC order - most recent first (UPDATE, then CREATE)
		require.Equal(t, "UPDATE", logs[0].Action)
		require.Equal(t, "CREATE", logs[1].Action)

		// Timestamps should be in descending order
		require.True(t, logs[0].Timestamp.After(logs[1].Timestamp),
			"First log timestamp should be after second log timestamp")
	})
}

// ============================================================================
// FindByTable Tests
// ============================================================================

func TestAuditLogRepository_FindByTable(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no audit logs for table", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		logs, err := repo.FindByTable(ctx, "employee", 10)
		require.NoError(t, err)
		require.NotNil(t, logs)
		require.Len(t, logs, 0)
	})

	t.Run("returns only logs for specified table", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 3 organizations
		for i := 0; i < 3; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Query organization table
		logs, err := auditRepo.FindByTable(ctx, "organization", 10)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// All logs should be for organization table
		for _, log := range logs {
			require.Equal(t, "organization", log.TableName)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 5 organizations
		for i := 0; i < 5; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Query with limit
		logs, err := auditRepo.FindByTable(ctx, "organization", 2)
		require.NoError(t, err)
		require.Len(t, logs, 2)
	})

	t.Run("returns logs in reverse chronological order", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		org.Name = "Updated"
		err = orgRepo.Update(ctx, org)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		err = orgRepo.Delete(ctx, org.ID)
		require.NoError(t, err)

		logs, err := auditRepo.FindByTable(ctx, "organization", 10)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// DESC order - most recent first
		require.Equal(t, "DELETE", logs[0].Action)
		require.Equal(t, "UPDATE", logs[1].Action)
		require.Equal(t, "CREATE", logs[2].Action)
	})
}

// ============================================================================
// FindByAction Tests
// ============================================================================

func TestAuditLogRepository_FindByAction(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no logs for action", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		logs, err := repo.FindByAction(ctx, "DELETE", 10)
		require.NoError(t, err)
		require.NotNil(t, logs)
		require.Len(t, logs, 0)
	})

	t.Run("returns only logs for specified action", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create and update 2 organizations
		for i := 0; i < 2; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)

			org.Name = "Updated"
			err = orgRepo.Update(ctx, org)
			require.NoError(t, err)
		}
		// Total: 2 CREATE, 2 UPDATE

		// Query only CREATE actions
		createLogs, err := auditRepo.FindByAction(ctx, "CREATE", 10)
		require.NoError(t, err)
		require.Len(t, createLogs, 2)
		for _, log := range createLogs {
			require.Equal(t, "CREATE", log.Action)
		}

		// Query only UPDATE actions
		updateLogs, err := auditRepo.FindByAction(ctx, "UPDATE", 10)
		require.NoError(t, err)
		require.Len(t, updateLogs, 2)
		for _, log := range updateLogs {
			require.Equal(t, "UPDATE", log.Action)
		}
	})

	t.Run("works with all action types", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()

		// CREATE
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// UPDATE
		org.Name = "Updated"
		err = orgRepo.Update(ctx, org)
		require.NoError(t, err)

		// DELETE (soft)
		err = orgRepo.Delete(ctx, org.ID)
		require.NoError(t, err)

		// RESTORE
		err = orgRepo.Restore(ctx, org.ID)
		require.NoError(t, err)

		// HARD_DELETE
		err = orgRepo.HardDelete(ctx, org.ID)
		require.NoError(t, err)

		// Test each action type
		actions := []string{"CREATE", "UPDATE", "DELETE", "RESTORE", "HARD_DELETE"}
		for _, action := range actions {
			logs, err := auditRepo.FindByAction(ctx, action, 10)
			require.NoError(t, err)
			require.Len(t, logs, 1, "action: %s", action)
			require.Equal(t, action, logs[0].Action)
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 5 organizations (5 CREATE actions)
		for i := 0; i < 5; i++ {
			org := createTestOrganization()
			err := orgRepo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Query with limit
		logs, err := auditRepo.FindByAction(ctx, "CREATE", 3)
		require.NoError(t, err)
		require.Len(t, logs, 3)
	})

	t.Run("returns logs in reverse chronological order", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create 3 organizations at different times
		org1 := createTestOrganization()
		err := orgRepo.Create(ctx, org1)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		org2 := createTestOrganization()
		err = orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(time.Second)

		org3 := createTestOrganization()
		err = orgRepo.Create(ctx, org3)
		require.NoError(t, err)

		logs, err := auditRepo.FindByAction(ctx, "CREATE", 10)
		require.NoError(t, err)
		require.Len(t, logs, 3)

		// DESC order - verify timestamps are strictly descending
		require.True(t,
			logs[0].Timestamp.After(logs[1].Timestamp),
			"First log timestamp should be after second log timestamp")
		require.True(t,
			logs[1].Timestamp.After(logs[2].Timestamp),
			"Second log timestamp should be after third log timestamp")

		// Verify newest is first (org3, org2, org1)
		require.Equal(t, org3.ID.String(), logs[0].RecordID)
		require.Equal(t, org2.ID.String(), logs[1].RecordID)
		require.Equal(t, org1.ID.String(), logs[2].RecordID)
	})
}

// ============================================================================
// Audit Log Content Tests
// ============================================================================

func TestAuditLogRepository_ContentValidation(t *testing.T) {
	t.Parallel()

	t.Run("CREATE action has empty before and populated after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		logs, err := auditRepo.FindByAction(ctx, "CREATE", 1)
		require.NoError(t, err)
		require.Len(t, logs, 1)

		log := logs[0]
		require.Empty(t, log.Before)
		require.NotEmpty(t, log.After)
		require.Contains(t, log.After, org.Name)
		require.Contains(t, log.After, org.ID.String())
	})

	t.Run("UPDATE action has both before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		originalName := org.Name
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		org.Name = "Updated Name"
		err = orgRepo.Update(ctx, org)
		require.NoError(t, err)

		logs, err := auditRepo.FindByAction(ctx, "UPDATE", 1)
		require.NoError(t, err)
		require.Len(t, logs, 1)

		log := logs[0]
		require.NotEmpty(t, log.Before)
		require.NotEmpty(t, log.After)
		require.Contains(t, log.Before, originalName)
		require.Contains(t, log.After, "Updated Name")
	})

	t.Run("DELETE action shows before and after with deleted_at", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		err = orgRepo.Delete(ctx, org.ID)
		require.NoError(t, err)

		logs, err := auditRepo.FindByAction(ctx, "DELETE", 1)
		require.NoError(t, err)
		require.Len(t, logs, 1)

		log := logs[0]
		require.NotEmpty(t, log.Before)
		require.NotEmpty(t, log.After)
		// Before should have DeletedAt set to null (JSON uses capitalized field names)
		require.Contains(t, log.Before, `"DeletedAt":null`)
		// After should have DeletedAt set (not null)
		require.NotContains(t, log.After, `"DeletedAt":null`)
	})

	t.Run("HARD_DELETE action has before but after is null", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		err = orgRepo.HardDelete(ctx, org.ID)
		require.NoError(t, err)

		logs, err := auditRepo.FindByAction(ctx, "HARD_DELETE", 1)
		require.NoError(t, err)
		require.Len(t, logs, 1)

		log := logs[0]
		require.NotEmpty(t, log.Before)
		require.Contains(t, log.Before, org.Name)
		require.Equal(t, "null", log.After) // Should be JSON "null"
	})
}

// ============================================================================
// WithTx Tests
// ============================================================================

func TestAuditLogRepository_WithTx(t *testing.T) {
	t.Parallel()

	t.Run("can query within transaction", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		auditRepo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Start transaction
		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		// Query audit logs within transaction
		txAuditRepo := auditRepo.WithTx(tx)
		logs, err := txAuditRepo.FindForRecord(ctx, "organization", org.ID.String())
		require.NoError(t, err)
		require.Len(t, logs, 1)

		err = tx.Commit()
		require.NoError(t, err)
	})

	t.Run("returns new repository instance", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		tx, err := db.BeginTx(ctx, nil)
		require.NoError(t, err)
		defer tx.Rollback()

		txRepo := repo.WithTx(tx)

		// Should be different instances
		require.NotEqual(t, repo, txRepo)
	})
}

// ============================================================================
// Error Handling Tests
// ============================================================================

func TestAuditLogRepository_ErrorHandling(t *testing.T) {
	t.Parallel()

	t.Run("handles database connection errors gracefully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		db.Close() // Close database

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		_, err := repo.FindRecent(ctx, 10)
		require.Error(t, err)
	})

	t.Run("handles invalid UUID in record ID", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewAuditLogRepository(db)
		ctx := context.Background()

		// Should not error - it's just a string filter (no results)
		logs, err := repo.FindForRecord(ctx, "organization", "not-a-uuid")
		require.NoError(t, err)
		require.Len(t, logs, 0)
	})
}
