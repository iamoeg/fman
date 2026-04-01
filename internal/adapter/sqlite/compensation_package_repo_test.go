package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/fman/internal/adapter/sqlite"
	"github.com/iamoeg/fman/internal/domain"
	"github.com/iamoeg/fman/pkg/money"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

var pkgCounter int64

// createTestCompensationPackage creates a valid test compensation package for the given org.
func createTestCompensationPackage(orgID uuid.UUID) *domain.EmployeeCompensationPackage {
	now := time.Now().UTC()
	counter := atomic.AddInt64(&pkgCounter, 1)

	// Base salary between SMIG (3000 MAD) and reasonable max (50000 MAD)
	baseSalaryCents := (3000 + counter) * 100

	return &domain.EmployeeCompensationPackage{
		ID:         uuid.New(),
		OrgID:      orgID,
		Name:       fmt.Sprintf("Package %d", counter),
		Currency:   money.MAD,
		BaseSalary: money.FromCents(baseSalaryCents),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

// createAndPersistTestOrg creates a test organization and persists it to the given DB.
func createAndPersistTestOrg(t *testing.T, db *sql.DB) *domain.Organization {
	t.Helper()
	orgRepo := sqlite.NewOrganizationRepository(db)
	org := createTestOrganization()
	err := orgRepo.Create(context.Background(), org)
	require.NoError(t, err)
	return org
}

// ============================================================================
// FindByID Tests
// ============================================================================

func TestCompensationPackageRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("returns package when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create package
		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Find it
		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, pkg.ID, found.ID)
		require.Equal(t, pkg.BaseSalary.Cents(), found.BaseSalary.Cents())
		require.Equal(t, pkg.Currency, found.Currency)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByID(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted packages", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create and delete package
		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Should not find it
		found, err := repo.FindByID(ctx, pkg.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByIDIncludingDeleted Tests
// ============================================================================

func TestCompensationPackageRepository_FindByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns active package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, pkg.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, pkg.ID, found.ID)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("returns soft-deleted package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, pkg.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, pkg.ID, found.ID)
		require.NotNil(t, found.DeletedAt) // Should have deleted_at set
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewCompensationPackageRepository(db)
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

func TestCompensationPackageRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no packages", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkgs, err := repo.FindAll(ctx, org.ID)
		require.NoError(t, err)
		require.NotNil(t, pkgs)
		require.Len(t, pkgs, 0)
	})

	t.Run("returns all active packages", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create 3 packages
		pkg1 := createTestCompensationPackage(org.ID)
		pkg2 := createTestCompensationPackage(org.ID)
		pkg3 := createTestCompensationPackage(org.ID)

		err := repo.Create(ctx, pkg1)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg2)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg3)
		require.NoError(t, err)

		pkgs, err := repo.FindAll(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, pkgs, 3)
	})

	t.Run("does not return soft-deleted packages", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create 2 packages, delete 1
		pkg1 := createTestCompensationPackage(org.ID)
		pkg2 := createTestCompensationPackage(org.ID)

		err := repo.Create(ctx, pkg1)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg2)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg1.ID)
		require.NoError(t, err)

		pkgs, err := repo.FindAll(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, pkgs, 1)
		require.Equal(t, pkg2.ID, pkgs[0].ID)
	})
}

// ============================================================================
// FindAllIncludingDeleted Tests
// ============================================================================

func TestCompensationPackageRepository_FindAllIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all packages including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create 3 packages, delete 1
		pkg1 := createTestCompensationPackage(org.ID)
		pkg2 := createTestCompensationPackage(org.ID)
		pkg3 := createTestCompensationPackage(org.ID)

		err := repo.Create(ctx, pkg1)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg2)
		require.NoError(t, err)
		err = repo.Create(ctx, pkg3)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg1.ID)
		require.NoError(t, err)

		pkgs, err := repo.FindAllIncludingDeleted(ctx, org.ID)
		require.NoError(t, err)
		require.Len(t, pkgs, 3) // Should include deleted
	})
}

// ============================================================================
// CountEmployeesUsing Tests
// ============================================================================

func TestCompensationPackageRepository_CountEmployeesUsing(t *testing.T) {
	t.Parallel()

	t.Run("returns zero when no employees use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		count, err := repo.CountEmployeesUsing(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), count)
	})

	t.Run("returns count when employees use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		pkgRepo := sqlite.NewCompensationPackageRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create compensation package
		pkg := createTestCompensationPackage(org.ID)
		err = pkgRepo.Create(ctx, pkg)
		require.NoError(t, err)

		// Create employees using this package
		_, err = db.Exec(`
			INSERT INTO employee (
				id, org_id, serial_num, full_name, birth_date, gender,
				marital_status, num_dependents, num_children, cin_num, hire_date,
				position, compensation_package_id, created_at, updated_at
			) VALUES
			(?, ?, 1, 'Ahmed Ali', '1990-01-01', 'MALE', 'SINGLE', 0, 0, 'CIN001', '2025-01-01', 'Developer', ?, ?, ?),
			(?, ?, 2, 'Fatima Zahra', '1992-05-15', 'FEMALE', 'MARRIED', 2, 1, 'CIN002', '2025-01-01', 'Designer', ?, ?, ?)
		`,
			uuid.New().String(), org.ID.String(), pkg.ID.String(), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339),
			uuid.New().String(), org.ID.String(), pkg.ID.String(), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339),
		)
		require.NoError(t, err)

		count, err := pkgRepo.CountEmployeesUsing(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, int64(2), count)
	})
}

// ============================================================================
// CountPayrollResultsUsing Tests
// ============================================================================

func TestCompensationPackageRepository_CountPayrollResultsUsing(t *testing.T) {
	t.Parallel()

	t.Run("returns zero when no payroll results use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		count, err := repo.CountPayrollResultsUsing(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, int64(0), count)
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestCompensationPackageRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates package successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Verify it was created
		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, pkg.ID, found.ID)
		require.Equal(t, pkg.BaseSalary.Cents(), found.BaseSalary.Cents())
		require.Equal(t, pkg.Currency, found.Currency)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'CREATE'",
			pkg.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("preserves money precision on round-trip", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Test various money values
		testCases := []struct {
			name  string
			cents int64
		}{
			{"SMIG minimum", 300000},         // 3000.00 MAD
			{"with cents", 350075},           // 3500.75 MAD
			{"large amount", 10000000},       // 100,000.00 MAD
			{"one cent", 1},                  // 0.01 MAD
			{"max reasonable", 999999999999}, // Very large but valid
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				pkg := createTestCompensationPackage(org.ID)
				pkg.BaseSalary = money.FromCents(tc.cents)

				err := repo.Create(ctx, pkg)
				require.NoError(t, err)

				found, err := repo.FindByID(ctx, pkg.ID)
				require.NoError(t, err)
				require.Equal(t, tc.cents, found.BaseSalary.Cents(),
					"Money precision should be exact: expected %d, got %d", tc.cents, found.BaseSalary.Cents())
			})
		}
	})
}

// ============================================================================
// Update Tests
// ============================================================================

func TestCompensationPackageRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates package successfully when not in use", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create
		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Update
		pkg.BaseSalary = money.FromCents(500000) // 5000 MAD
		err = repo.Update(ctx, pkg)
		require.NoError(t, err)

		// Verify
		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Equal(t, int64(500000), found.BaseSalary.Cents())
	})

	t.Run("creates audit log with before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create
		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Update
		pkg.BaseSalary = money.FromCents(600000)
		err = repo.Update(ctx, pkg)
		require.NoError(t, err)

		// Check audit log
		var before, after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'UPDATE'",
			pkg.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)

		// Verify both snapshots contain the package ID
		require.Contains(t, before, pkg.ID.String())
		require.Contains(t, after, pkg.ID.String())

		// Verify both snapshots contain the currency
		require.Contains(t, before, "MAD")
		require.Contains(t, after, "MAD")
	})

	t.Run("returns ErrRecordNotFound when package does not exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Update(ctx, pkg)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("returns ErrRecordNotFound when package is deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		// Create and delete
		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)
		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Try to update
		pkg.BaseSalary = money.FromCents(700000)
		err = repo.Update(ctx, pkg)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("returns ErrCompensationPackageInUse when employees use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		pkgRepo := sqlite.NewCompensationPackageRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create compensation package
		pkg := createTestCompensationPackage(org.ID)
		err = pkgRepo.Create(ctx, pkg)
		require.NoError(t, err)

		// Create employee using this package
		_, err = db.Exec(`
			INSERT INTO employee (
				id, org_id, serial_num, full_name, birth_date, gender,
				marital_status, num_dependents, num_children, cin_num, hire_date,
				position, compensation_package_id, created_at, updated_at
			) VALUES (?, ?, 1, 'Test Employee', '1990-01-01', 'MALE', 'SINGLE', 0, 0, 'CIN001', '2025-01-01', 'Developer', ?, ?, ?)
		`, uuid.New().String(), org.ID.String(), pkg.ID.String(), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		require.NoError(t, err)

		// Try to update - should fail
		pkg.BaseSalary = money.FromCents(800000)
		err = pkgRepo.Update(ctx, pkg)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrCompensationPackageInUse)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestCompensationPackageRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft deletes package when not in use", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Should not be found by regular FindByID
		found, err := repo.FindByID(ctx, pkg.ID)
		require.Error(t, err)
		require.Nil(t, found)

		// Should be found with IncludingDeleted
		foundDeleted, err := repo.FindByIDIncludingDeleted(ctx, pkg.ID)
		require.NoError(t, err)
		require.NotNil(t, foundDeleted.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'DELETE'",
			pkg.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("returns ErrCompensationPackageInUse when employees use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		pkgRepo := sqlite.NewCompensationPackageRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create compensation package
		pkg := createTestCompensationPackage(org.ID)
		err = pkgRepo.Create(ctx, pkg)
		require.NoError(t, err)

		// Create employee using this package
		_, err = db.Exec(`
			INSERT INTO employee (
				id, org_id, serial_num, full_name, birth_date, gender,
				marital_status, num_dependents, num_children, cin_num, hire_date,
				position, compensation_package_id, created_at, updated_at
			) VALUES (?, ?, 1, 'Test Employee', '1990-01-01', 'MALE', 'SINGLE', 0, 0, 'CIN001', '2025-01-01', 'Developer', ?, ?, ?)
		`, uuid.New().String(), org.ID.String(), pkg.ID.String(), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		require.NoError(t, err)

		// Try to delete - should fail
		err = pkgRepo.Delete(ctx, pkg.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrCompensationPackageInUse)
	})
}

// ============================================================================
// Restore Tests
// ============================================================================

func TestCompensationPackageRepository_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores soft-deleted package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		// Restore
		err = repo.Restore(ctx, pkg.ID)
		require.NoError(t, err)

		// Should be found now
		found, err := repo.FindByID(ctx, pkg.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.Delete(ctx, pkg.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, pkg.ID)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'RESTORE'",
			pkg.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// HardDelete Tests
// ============================================================================

func TestCompensationPackageRepository_HardDelete(t *testing.T) {
	t.Parallel()

	t.Run("permanently deletes package when not in use", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, pkg.ID)
		require.NoError(t, err)

		// Should not be found even with IncludingDeleted
		found, err := repo.FindByIDIncludingDeleted(ctx, pkg.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("creates audit log entry before deletion", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		org := createAndPersistTestOrg(t, db)
		repo := sqlite.NewCompensationPackageRepository(db)
		ctx := context.Background()

		pkg := createTestCompensationPackage(org.ID)
		err := repo.Create(ctx, pkg)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, pkg.ID)
		require.NoError(t, err)

		// Audit log should still exist
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee_compensation_package' AND record_id = ? AND action = 'HARD_DELETE'",
			pkg.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("returns ErrCompensationPackageInUse when employees use package", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		pkgRepo := sqlite.NewCompensationPackageRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		// Create organization
		org := createTestOrganization()
		err := orgRepo.Create(ctx, org)
		require.NoError(t, err)

		// Create compensation package
		pkg := createTestCompensationPackage(org.ID)
		err = pkgRepo.Create(ctx, pkg)
		require.NoError(t, err)

		// Create employee using this package
		_, err = db.Exec(`
			INSERT INTO employee (
				id, org_id, serial_num, full_name, birth_date, gender,
				marital_status, num_dependents, num_children, cin_num, hire_date,
				position, compensation_package_id, created_at, updated_at
			) VALUES (?, ?, 1, 'Test Employee', '1990-01-01', 'MALE', 'SINGLE', 0, 0, 'CIN001', '2025-01-01', 'Developer', ?, ?, ?)
		`, uuid.New().String(), org.ID.String(), pkg.ID.String(), time.Now().Format(time.RFC3339), time.Now().Format(time.RFC3339))
		require.NoError(t, err)

		// Soft delete first
		err = pkgRepo.Delete(ctx, pkg.ID)
		require.Error(t, err) // Should fail because in use

		// Try to hard delete without soft delete - should still fail
		err = pkgRepo.HardDelete(ctx, pkg.ID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrCompensationPackageInUse)
	})
}
