package sqlite_adapter_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/fman/internal/adapter/sqlite"
	"github.com/iamoeg/fman/internal/domain"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

var periodCounter int64

// createTestPayrollPeriod creates a valid test payroll period.
// Requires an organization ID to be provided.
func createTestPayrollPeriod(orgID uuid.UUID) *domain.PayrollPeriod {
	now := time.Now().UTC()
	counter := atomic.AddInt64(&periodCounter, 1)

	// Generate year/month that won't conflict
	// Use counter to offset months
	baseYear := 2024
	baseMonth := 1
	totalMonths := int(counter)
	year := baseYear + (totalMonths / 12)
	month := baseMonth + (totalMonths % 12)
	if month > 12 {
		year++
		month -= 12
	}

	return &domain.PayrollPeriod{
		ID:          uuid.New(),
		OrgID:       orgID,
		Year:        year,
		Month:       month,
		Status:      domain.PayrollPeriodStatusDraft,
		FinalizedAt: nil,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ============================================================================
// FindByID Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("returns period when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create organization first
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create period
		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		// Find it
		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, period.ID, found.ID)
		require.Equal(t, period.OrgID, found.OrgID)
		require.Equal(t, period.Year, found.Year)
		require.Equal(t, period.Month, found.Month)
		require.Equal(t, domain.PayrollPeriodStatusDraft, found.Status)
		require.Nil(t, found.FinalizedAt)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByID(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create and delete period
		period := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		// Should not find it
		found, err := repo.FindByID(ctx, period.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByIDIncludingDeleted Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns active period", func(t *testing.T) {
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

		found, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, period.ID, found.ID)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("returns soft-deleted period", func(t *testing.T) {
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

		found, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, period.ID, found.ID)
		require.NotNil(t, found.DeletedAt)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByIDIncludingDeleted(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByOrgYearMonth Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByOrgYearMonth(t *testing.T) {
	t.Parallel()

	t.Run("finds period by organization, year, and month", func(t *testing.T) {
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
		period.Year = 2024
		period.Month = 6
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByOrgYearMonth(ctx, org.ID, 2024, 6)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, period.ID, found.ID)
		require.Equal(t, 2024, found.Year)
		require.Equal(t, 6, found.Month)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		found, err := repo.FindByOrgYearMonth(ctx, org.ID, 2025, 12)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted periods", func(t *testing.T) {
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
		period.Year = 2024
		period.Month = 8
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		found, err := repo.FindByOrgYearMonth(ctx, org.ID, 2024, 8)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByOrgYearMonthIncludingDeleted Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByOrgYearMonthIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("finds active period", func(t *testing.T) {
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
		period.Year = 2024
		period.Month = 3
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		found, err := repo.FindByOrgYearMonthIncludingDeleted(ctx, org.ID, 2024, 3)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("finds soft-deleted period", func(t *testing.T) {
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
		period.Year = 2024
		period.Month = 9
		err = repo.Create(ctx, period)
		require.NoError(t, err)

		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		found, err := repo.FindByOrgYearMonthIncludingDeleted(ctx, org.ID, 2024, 9)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.NotNil(t, found.DeletedAt)
	})
}

// ============================================================================
// FindByOrganization Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByOrganization(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		periods, err := repo.FindByOrganization(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, periods)
		require.Len(t, periods, 0)
	})

	t.Run("returns all active periods for organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 3 periods
		period1 := createTestPayrollPeriod(org.ID)
		period2 := createTestPayrollPeriod(org.ID)
		period3 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)
		err = repo.Create(ctx, period3)
		require.NoError(t, err)

		periods, err := repo.FindByOrganization(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, periods, 3)
	})

	t.Run("does not return soft-deleted periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 2 periods, delete 1
		period1 := createTestPayrollPeriod(org.ID)
		period2 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)

		err = repo.Delete(ctx, period1.ID)
		require.NoError(t, err)

		periods, err := repo.FindByOrganization(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, periods, 1)
		require.Equal(t, period2.ID, periods[0].ID)
	})

	t.Run("only returns periods for specified organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create two organizations
		org1 := createTestOrganization()
		org2 := createTestOrganization()
		err := orgRepo.Create(ctx, org1)
		require.NoError(t, err)
		err = orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Create periods for each org
		period1 := createTestPayrollPeriod(org1.ID)
		period2 := createTestPayrollPeriod(org2.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)

		// Query org1's periods
		periods, err := repo.FindByOrganization(ctx, org1.ID)
		require.NoError(t, err)
		require.Len(t, periods, 1)
		require.Equal(t, period1.ID, periods[0].ID)
	})
}

// ============================================================================
// FindByOrganizationIncludingDeleted Tests
// ============================================================================

func TestPayrollPeriodRepository_FindByOrganizationIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all periods including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 3 periods, delete 1
		period1 := createTestPayrollPeriod(org.ID)
		period2 := createTestPayrollPeriod(org.ID)
		period3 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)
		err = repo.Create(ctx, period3)
		require.NoError(t, err)

		err = repo.Delete(ctx, period1.ID)
		require.NoError(t, err)

		periods, err := repo.FindByOrganizationIncludingDeleted(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, periods, 3)
	})
}

// ============================================================================
// FindAllDraft Tests
// ============================================================================

func TestPayrollPeriodRepository_FindAllDraft(t *testing.T) {
	t.Parallel()

	t.Run("returns only draft periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create draft and finalized periods
		draft1 := createTestPayrollPeriod(org.ID)
		draft2 := createTestPayrollPeriod(org.ID)
		finalized := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, draft1)
		require.NoError(t, err)
		err = repo.Create(ctx, draft2)
		require.NoError(t, err)
		err = repo.Create(ctx, finalized)
		require.NoError(t, err)

		// Finalize one period
		err = repo.Finalize(ctx, finalized.ID)
		require.NoError(t, err)

		// Query draft periods
		drafts, err := repo.FindAllDraft(ctx)
		require.NoError(t, err)
		require.Len(t, drafts, 2)

		// Verify all returned periods are draft
		for _, p := range drafts {
			require.Equal(t, domain.PayrollPeriodStatusDraft, p.Status)
		}
	})

	t.Run("returns empty slice when no draft periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		drafts, err := repo.FindAllDraft(ctx)
		require.NoError(t, err)
		require.NotNil(t, drafts)
		require.Len(t, drafts, 0)
	})

	t.Run("does not return soft-deleted draft periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		draft := createTestPayrollPeriod(org.ID)
		err = repo.Create(ctx, draft)
		require.NoError(t, err)

		err = repo.Delete(ctx, draft.ID)
		require.NoError(t, err)

		drafts, err := repo.FindAllDraft(ctx)
		require.NoError(t, err)
		require.Len(t, drafts, 0)
	})
}

// ============================================================================
// FindAllDraftIncludingDeleted Tests
// ============================================================================

func TestPayrollPeriodRepository_FindAllDraftIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns draft periods including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 2 draft periods, delete 1
		draft1 := createTestPayrollPeriod(org.ID)
		draft2 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, draft1)
		require.NoError(t, err)
		err = repo.Create(ctx, draft2)
		require.NoError(t, err)

		err = repo.Delete(ctx, draft1.ID)
		require.NoError(t, err)

		drafts, err := repo.FindAllDraftIncludingDeleted(ctx)
		require.NoError(t, err)
		require.Len(t, drafts, 2)
	})
}

// ============================================================================
// FindAll Tests
// ============================================================================

func TestPayrollPeriodRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		periods, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.NotNil(t, periods)
		require.Len(t, periods, 0)
	})

	t.Run("returns all active periods across all organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create two organizations
		org1 := createTestOrganization()
		org2 := createTestOrganization()
		err := orgRepo.Create(ctx, org1)
		require.NoError(t, err)
		err = orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Create periods for each
		period1 := createTestPayrollPeriod(org1.ID)
		period2 := createTestPayrollPeriod(org2.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)

		periods, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, periods, 2)
	})

	t.Run("does not return soft-deleted periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 2 periods, delete 1
		period1 := createTestPayrollPeriod(org.ID)
		period2 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)

		err = repo.Delete(ctx, period1.ID)
		require.NoError(t, err)

		periods, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, periods, 1)
		require.Equal(t, period2.ID, periods[0].ID)
	})
}

// ============================================================================
// FindAllIncludingDeleted Tests
// ============================================================================

func TestPayrollPeriodRepository_FindAllIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all periods including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create 3 periods, delete 1
		period1 := createTestPayrollPeriod(org.ID)
		period2 := createTestPayrollPeriod(org.ID)
		period3 := createTestPayrollPeriod(org.ID)

		err = repo.Create(ctx, period1)
		require.NoError(t, err)
		err = repo.Create(ctx, period2)
		require.NoError(t, err)
		err = repo.Create(ctx, period3)
		require.NoError(t, err)

		err = repo.Delete(ctx, period1.ID)
		require.NoError(t, err)

		periods, err := repo.FindAllIncludingDeleted(ctx)
		require.NoError(t, err)
		require.Len(t, periods, 3)
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestPayrollPeriodRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates period successfully", func(t *testing.T) {
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

		// Verify created
		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, period.ID, found.ID)
		require.Equal(t, period.Year, found.Year)
		require.Equal(t, period.Month, found.Month)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'CREATE'",
			period.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("enforces unique constraint on org_id, year, month", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgRepo := sqlite.NewOrganizationRepository(db)
		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create first period
		period1 := createTestPayrollPeriod(org.ID)
		period1.Year = 2024
		period1.Month = 12
		err = repo.Create(ctx, period1)
		require.NoError(t, err)

		// Try to create duplicate
		period2 := createTestPayrollPeriod(org.ID)
		period2.Year = 2024
		period2.Month = 12
		err = repo.Create(ctx, period2)
		require.Error(t, err)
	})
}

// ============================================================================
// Finalize Tests
// ============================================================================

func TestPayrollPeriodRepository_Finalize(t *testing.T) {
	t.Parallel()

	t.Run("finalizes draft period successfully", func(t *testing.T) {
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

		// Finalize
		before := time.Now().UTC().Truncate(time.Second)
		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)
		after := time.Now().UTC().Truncate(time.Second).Add(time.Second)

		// Verify status changed and finalized_at set
		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, domain.PayrollPeriodStatusFinalized, found.Status)
		require.NotNil(t, found.FinalizedAt)

		finalizedAt := found.FinalizedAt.UTC().Truncate(time.Second)
		require.True(t, finalizedAt.Equal(before) || finalizedAt.After(before))
		require.True(t, finalizedAt.Equal(after) || finalizedAt.Before(after))
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log contains before (DRAFT) and after (FINALIZED)
		var before, after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'UPDATE' ORDER BY timestamp DESC LIMIT 1",
			period.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)
		require.Contains(t, before, "DRAFT")
		require.Contains(t, after, "FINALIZED")
	})

	t.Run("returns error when period not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		err := repo.Finalize(ctx, randomID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("cannot finalize already finalized period", func(t *testing.T) {
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

		// Finalize once
		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		// Try to finalize again
		err = repo.Finalize(ctx, period.ID)
		require.Error(t, err)
	})
}

// ============================================================================
// Unfinalize Tests
// ============================================================================

func TestPayrollPeriodRepository_Unfinalize(t *testing.T) {
	t.Parallel()

	t.Run("unfinalizes finalized period successfully", func(t *testing.T) {
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

		// Finalize then unfinalize
		err = repo.Finalize(ctx, period.ID)
		require.NoError(t, err)

		err = repo.Unfinalize(ctx, period.ID)
		require.NoError(t, err)

		// Verify status changed back and finalized_at cleared
		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Equal(t, domain.PayrollPeriodStatusDraft, found.Status)
		require.Nil(t, found.FinalizedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'UPDATE'",
			period.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 2, count) // One for finalize, one for unfinalize
	})

	t.Run("cannot unfinalize draft period", func(t *testing.T) {
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

		// Try to unfinalize draft period
		err = repo.Unfinalize(ctx, period.ID)
		require.Error(t, err)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestPayrollPeriodRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft deletes period", func(t *testing.T) {
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

		// Should not be found by regular FindByID
		found, err := repo.FindByID(ctx, period.ID)
		require.Error(t, err)
		require.Nil(t, found)

		// Should be found with IncludingDeleted
		foundDeleted, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
		require.NoError(t, err)
		require.NotNil(t, foundDeleted.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'DELETE'",
			period.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// Restore Tests
// ============================================================================

func TestPayrollPeriodRepository_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores soft-deleted period", func(t *testing.T) {
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

		// Delete
		err = repo.Delete(ctx, period.ID)
		require.NoError(t, err)

		// Restore
		err = repo.Restore(ctx, period.ID)
		require.NoError(t, err)

		// Should be found now
		found, err := repo.FindByID(ctx, period.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'RESTORE'",
			period.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// HardDelete Tests
// ============================================================================

func TestPayrollPeriodRepository_HardDelete(t *testing.T) {
	t.Parallel()

	t.Run("permanently deletes period", func(t *testing.T) {
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

		// Should not be found even with IncludingDeleted
		found, err := repo.FindByIDIncludingDeleted(ctx, period.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("creates audit log entry before deletion", func(t *testing.T) {
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

		// Audit log should still exist
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_period' AND record_id = ? AND action = 'HARD_DELETE'",
			period.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}
