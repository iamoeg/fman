package sqlite_adapter_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
)

// ============================================================================
// DBActionEnum Tests
// ============================================================================

func TestDBActionEnum_IsSupported(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		action    sqlite.DBActionEnum
		supported bool
	}{
		{
			name:      "CREATE is supported",
			action:    sqlite.DBActionCreate,
			supported: true,
		},
		{
			name:      "UPDATE is supported",
			action:    sqlite.DBActionUpdate,
			supported: true,
		},
		{
			name:      "DELETE is supported",
			action:    sqlite.DBActionDelete,
			supported: true,
		},
		{
			name:      "RESTORE is supported",
			action:    sqlite.DBActionRestore,
			supported: true,
		},
		{
			name:      "HARD_DELETE is supported",
			action:    sqlite.DBActionHardDelete,
			supported: true,
		},
		{
			name:      "invalid action is not supported",
			action:    sqlite.DBActionEnum("INVALID"),
			supported: false,
		},
		{
			name:      "empty action is not supported",
			action:    sqlite.DBActionEnum(""),
			supported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.action.IsSupported()
			require.Equal(t, tt.supported, result)
		})
	}
}

// ============================================================================
// Audit Log Query Tests
// ============================================================================

func TestAuditLog_Queries(t *testing.T) {
	t.Parallel()

	t.Run("can query audit logs by record", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := repo.Create(ctx, org)
		require.NoError(t, err)

		// Update to create more audit entries
		org.Name = "Updated"
		err = repo.Update(ctx, org)
		require.NoError(t, err)

		// Query audit logs
		rows, err := db.Query(
			"SELECT action FROM audit_log WHERE record_id = ? ORDER BY timestamp",
			org.ID.String(),
		)
		require.NoError(t, err)
		defer rows.Close()

		actions := []string{}
		for rows.Next() {
			var action string
			err = rows.Scan(&action)
			require.NoError(t, err)
			actions = append(actions, action)
		}

		require.Len(t, actions, 2)
		require.Equal(t, "UPDATE", actions[0])
		require.Equal(t, "CREATE", actions[1])
	})

	t.Run("can query recent audit logs", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create multiple organizations
		for i := 0; i < 5; i++ {
			org := createTestOrganization()
			err := repo.Create(ctx, org)
			require.NoError(t, err)
		}

		// Query recent logs
		rows, err := db.Query(
			"SELECT action FROM audit_log ORDER BY timestamp DESC LIMIT 5",
		)
		require.NoError(t, err)
		defer rows.Close()

		count := 0
		for rows.Next() {
			count++
		}
		require.Equal(t, 5, count)
	})
}

// ============================================================================
// Constants Tests
// ============================================================================

func TestConstants(t *testing.T) {
	t.Parallel()

	t.Run("DBTimeFormat is RFC3339", func(t *testing.T) {
		require.Equal(t, time.RFC3339, sqlite.DBTimeFormat)
	})

	t.Run("SupportedDBActionsStr contains all actions", func(t *testing.T) {
		str := sqlite.SupportedDBActionsStr
		require.Contains(t, str, "CREATE")
		require.Contains(t, str, "UPDATE")
		require.Contains(t, str, "DELETE")
		require.Contains(t, str, "RESTORE")
		require.Contains(t, str, "HARD_DELETE")
	})
}
