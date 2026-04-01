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
)

// ============================================================================
// Test Helpers for Employee
// ============================================================================

var empCounter int64

// createTestEmployee creates a valid test employee with unique identifiers.
// Requires a valid organization and compensation package to be created first.
func createTestEmployee(orgID, compPackID uuid.UUID, serialNum int) *domain.Employee {
	now := time.Now().UTC()
	counter := atomic.AddInt64(&empCounter, 1)

	// Birth date: 25 years ago (legal working age)
	birthDate := now.AddDate(-25, 0, 0)
	// Hire date: 6 months ago
	hireDate := now.AddDate(0, -6, 0)

	return &domain.Employee{
		ID:                    uuid.New(),
		OrgID:                 orgID,
		SerialNum:             serialNum,
		FullName:              fmt.Sprintf("Ahmed Ben Ali %d", counter),
		DisplayName:           fmt.Sprintf("Ahmed %d", counter),
		Address:               "12 Rue Hassan II, Rabat",
		EmailAddress:          fmt.Sprintf("ahmed.ali%d@example.ma", counter),
		PhoneNumber:           fmt.Sprintf("+212-6%08d", counter),
		BirthDate:             birthDate,
		Gender:                domain.GenderMale,
		MaritalStatus:         domain.MaritalStatusSingle,
		NumDependents:         0,
		NumChildren:           0,
		CINNum:                fmt.Sprintf("AA%06d", counter), // Unique CIN
		CNSSNum:               fmt.Sprintf("%09d", counter),   // Unique CNSS
		HireDate:              hireDate,
		Position:              "Software Developer",
		CompensationPackageID: compPackID,
		BankRIB:               fmt.Sprintf("%024d", counter),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
}

// setupEmployeeTestDeps creates organization and compensation package dependencies.
func setupEmployeeTestDeps(t *testing.T, ctx context.Context, db *sql.DB) (orgID, compPkgID uuid.UUID) {
	t.Helper()

	// Create organization
	orgRepo := sqlite.NewOrganizationRepository(db)
	org := createTestOrganization()
	err := orgRepo.Create(ctx, org)
	require.NoError(t, err)

	// Create compensation package
	compPackRepo := sqlite.NewCompensationPackageRepository(db)
	compPack := createTestCompensationPackage(org.ID)
	err = compPackRepo.Create(ctx, compPack)
	require.NoError(t, err)

	return org.ID, compPack.ID
}

// ============================================================================
// FindByID Tests
// ============================================================================

func TestEmployeeRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("returns employee when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create employee
		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Find it
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, emp.ID, found.ID)
		require.Equal(t, emp.FullName, found.FullName)
		require.Equal(t, emp.CINNum, found.CINNum)
		require.Equal(t, emp.SerialNum, found.SerialNum)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByID(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create and delete employee
		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp.ID)
		require.NoError(t, err)

		// Should not find it
		found, err := repo.FindByID(ctx, emp.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByIDIncludingDeleted Tests
// ============================================================================

func TestEmployeeRepository_FindByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns active employee", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, emp.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, emp.ID, found.ID)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("returns soft-deleted employee", func(t *testing.T) {
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

		found, err := repo.FindByIDIncludingDeleted(ctx, emp.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, emp.ID, found.ID)
		require.NotNil(t, found.DeletedAt) // Should have deleted_at set
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByIDIncludingDeleted(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByOrgAndSerialNum Tests
// ============================================================================

func TestEmployeeRepository_FindByOrgAndSerialNum(t *testing.T) {
	t.Parallel()

	t.Run("finds employee by org and serial number", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create employee with serial number 1
		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Find by org and serial number
		found, err := repo.FindByOrgAndSerialNum(ctx, orgID, 1)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, emp.ID, found.ID)
		require.Equal(t, 1, found.SerialNum)
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, _ := setupEmployeeTestDeps(t, ctx, db)

		found, err := repo.FindByOrgAndSerialNum(ctx, orgID, 999)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted employee", func(t *testing.T) {
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

		found, err := repo.FindByOrgAndSerialNum(ctx, orgID, 1)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByOrganization Tests
// ============================================================================

func TestEmployeeRepository_FindByOrganization(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, _ := setupEmployeeTestDeps(t, ctx, db)

		emps, err := repo.FindByOrganization(ctx, orgID)
		require.NoError(t, err)
		require.NotNil(t, emps)
		require.Len(t, emps, 0)
	})

	t.Run("returns all active employees for organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 3 employees
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp3 := createTestEmployee(orgID, compPackID, 3)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)
		err = repo.Create(ctx, emp3)
		require.NoError(t, err)

		emps, err := repo.FindByOrganization(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, emps, 3)
	})

	t.Run("does not return soft-deleted employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 2 employees, delete 1
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp1.ID)
		require.NoError(t, err)

		emps, err := repo.FindByOrganization(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, emps, 1)
		require.Equal(t, emp2.ID, emps[0].ID)
	})

	t.Run("only returns employees for specified organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create two organizations
		org1ID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		orgRepo := sqlite.NewOrganizationRepository(db)
		org2 := createTestOrganization()
		err := orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Create employees for both orgs
		emp1 := createTestEmployee(org1ID, compPackID, 1)
		emp2 := createTestEmployee(org2.ID, compPackID, 1) // Serial num 1 for org2

		err = repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)

		// Find employees for org1 only
		emps, err := repo.FindByOrganization(ctx, org1ID)
		require.NoError(t, err)
		require.Len(t, emps, 1)
		require.Equal(t, emp1.ID, emps[0].ID)
	})
}

// ============================================================================
// FindByOrganizationIncludingDeleted Tests
// ============================================================================

func TestEmployeeRepository_FindByOrganizationIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all employees including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 3 employees, delete 1
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp3 := createTestEmployee(orgID, compPackID, 3)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)
		err = repo.Create(ctx, emp3)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp1.ID)
		require.NoError(t, err)

		emps, err := repo.FindByOrganizationIncludingDeleted(ctx, orgID)
		require.NoError(t, err)
		require.Len(t, emps, 3) // Should include deleted employee
	})
}

// ============================================================================
// FindAll Tests
// ============================================================================

func TestEmployeeRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		emps, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.NotNil(t, emps)
		require.Len(t, emps, 0)
	})

	t.Run("returns all active employees across organizations", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 3 employees
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp3 := createTestEmployee(orgID, compPackID, 3)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)
		err = repo.Create(ctx, emp3)
		require.NoError(t, err)

		emps, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, emps, 3)
	})

	t.Run("does not return soft-deleted employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 2 employees, delete 1
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp1.ID)
		require.NoError(t, err)

		emps, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, emps, 1)
		require.Equal(t, emp2.ID, emps[0].ID)
	})
}

// ============================================================================
// FindAllIncludingDeleted Tests
// ============================================================================

func TestEmployeeRepository_FindAllIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all employees including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 3 employees, delete 1
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp3 := createTestEmployee(orgID, compPackID, 3)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)
		err = repo.Create(ctx, emp3)
		require.NoError(t, err)

		err = repo.Delete(ctx, emp1.ID)
		require.NoError(t, err)

		emps, err := repo.FindAllIncludingDeleted(ctx)
		require.NoError(t, err)
		require.Len(t, emps, 3) // Should include deleted employee
	})
}

// ============================================================================
// GetNextSerialNumber Tests
// ============================================================================

func TestEmployeeRepository_GetNextSerialNumber(t *testing.T) {
	t.Parallel()

	t.Run("returns 1 for organization with no employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, _ := setupEmployeeTestDeps(t, ctx, db)

		sn, err := repo.GetNextSerialNumber(ctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 1, sn)
	})

	t.Run("returns next serial number after existing employees", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create 3 employees
		emp1 := createTestEmployee(orgID, compPackID, 1)
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp3 := createTestEmployee(orgID, compPackID, 3)

		err := repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)
		err = repo.Create(ctx, emp3)
		require.NoError(t, err)

		// Next should be 4
		sn, err := repo.GetNextSerialNumber(ctx, orgID)
		require.NoError(t, err)
		require.Equal(t, 4, sn)
	})

	t.Run("serial numbers are per-organization", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		org1ID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create second organization
		org2 := createTestOrganization()
		err := orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Create employees for org1
		emp1 := createTestEmployee(org1ID, compPackID, 1)
		emp2 := createTestEmployee(org1ID, compPackID, 2)

		err = repo.Create(ctx, emp1)
		require.NoError(t, err)
		err = repo.Create(ctx, emp2)
		require.NoError(t, err)

		// Org1 next serial should be 3
		sn1, err := repo.GetNextSerialNumber(ctx, org1ID)
		require.NoError(t, err)
		require.Equal(t, 3, sn1)

		// Org2 next serial should be 1 (independent)
		sn2, err := repo.GetNextSerialNumber(ctx, org2.ID)
		require.NoError(t, err)
		require.Equal(t, 1, sn2)
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestEmployeeRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates employee successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Verify it was created
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, emp.FullName, found.FullName)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'CREATE'",
			emp.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("handles empty optional fields", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		emp.DisplayName = ""  // Empty string should become NULL
		emp.Address = ""      // Empty string should become NULL
		emp.EmailAddress = "" // Empty string should become NULL
		emp.PhoneNumber = ""  // Empty string should become NULL
		emp.CNSSNum = ""      // Empty string should become NULL
		emp.BankRIB = ""      // Empty string should become NULL

		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Verify empty strings were stored as NULL (retrieved as empty strings)
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, "", found.DisplayName)
		require.Equal(t, "", found.Address)
		require.Equal(t, "", found.EmailAddress)
	})

	t.Run("enforces unique constraint on CIN number", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp1 := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp1)
		require.NoError(t, err)

		// Try to create another employee with same CIN
		emp2 := createTestEmployee(orgID, compPackID, 2)
		emp2.CINNum = emp1.CINNum // Duplicate CIN
		err = repo.Create(ctx, emp2)
		require.Error(t, err) // Should fail on UNIQUE constraint
	})

	t.Run("enforces unique constraint on org_id and serial_num", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp1 := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp1)
		require.NoError(t, err)

		// Try to create another employee with same serial number in same org
		emp2 := createTestEmployee(orgID, compPackID, 1) // Same serial num
		emp2.CINNum = "DIFFERENT123"                     // Different CIN to avoid that constraint
		err = repo.Create(ctx, emp2)
		require.Error(t, err) // Should fail on UNIQUE(org_id, serial_num)
	})
}

// ============================================================================
// Update Tests
// ============================================================================

func TestEmployeeRepository_Update(t *testing.T) {
	t.Parallel()

	t.Run("updates employee successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create
		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Update
		emp.FullName = "Updated Name"
		emp.Position = "Senior Developer"
		emp.EmailAddress = "updated@example.ma"
		err = repo.Update(ctx, emp)
		require.NoError(t, err)

		// Verify
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, "Updated Name", found.FullName)
		require.Equal(t, "Senior Developer", found.Position)
		require.Equal(t, "updated@example.ma", found.EmailAddress)
	})

	t.Run("creates audit log with before and after", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create
		emp := createTestEmployee(orgID, compPackID, 1)
		originalName := emp.FullName
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Update
		emp.FullName = "Changed Name"
		err = repo.Update(ctx, emp)
		require.NoError(t, err)

		// Check audit log
		var before, after string
		err = db.QueryRow(
			"SELECT before, after FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'UPDATE'",
			emp.ID.String(),
		).Scan(&before, &after)
		require.NoError(t, err)
		require.Contains(t, before, originalName)
		require.Contains(t, after, "Changed Name")
	})

	t.Run("returns ErrRecordNotFound when employee does not exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Update(ctx, emp)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("returns ErrRecordNotFound when employee is deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create and delete
		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)
		err = repo.Delete(ctx, emp.ID)
		require.NoError(t, err)

		// Try to update
		emp.FullName = "Should Fail"
		err = repo.Update(ctx, emp)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not update org_id or serial_num", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		orgRepo := sqlite.NewOrganizationRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		// Create second organization
		org2 := createTestOrganization()
		err := orgRepo.Create(ctx, org2)
		require.NoError(t, err)

		// Create employee
		emp := createTestEmployee(orgID, compPackID, 1)
		originalOrgID := emp.OrgID
		originalSerialNum := emp.SerialNum
		err = repo.Create(ctx, emp)
		require.NoError(t, err)

		// Try to update org_id and serial_num (should be ignored by UPDATE query)
		emp.OrgID = org2.ID
		emp.SerialNum = 999
		err = repo.Update(ctx, emp)
		require.NoError(t, err)

		// Verify org_id and serial_num did NOT change
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Equal(t, originalOrgID, found.OrgID)
		require.Equal(t, originalSerialNum, found.SerialNum)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestEmployeeRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft deletes employee", func(t *testing.T) {
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

		// Should not be found by regular FindByID
		found, err := repo.FindByID(ctx, emp.ID)
		require.Error(t, err)
		require.Nil(t, found)

		// Should be found with IncludingDeleted
		foundDeleted, err := repo.FindByIDIncludingDeleted(ctx, emp.ID)
		require.NoError(t, err)
		require.NotNil(t, foundDeleted.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'DELETE'",
			emp.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// Restore Tests
// ============================================================================

func TestEmployeeRepository_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores soft-deleted employee", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		orgID, compPackID := setupEmployeeTestDeps(t, ctx, db)

		emp := createTestEmployee(orgID, compPackID, 1)
		err := repo.Create(ctx, emp)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, emp.ID)
		require.NoError(t, err)

		// Restore
		err = repo.Restore(ctx, emp.ID)
		require.NoError(t, err)

		// Should be found now
		found, err := repo.FindByID(ctx, emp.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
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

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'RESTORE'",
			emp.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// HardDelete Tests
// ============================================================================

func TestEmployeeRepository_HardDelete(t *testing.T) {
	t.Parallel()

	t.Run("permanently deletes employee", func(t *testing.T) {
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

		// Should not be found even with IncludingDeleted
		found, err := repo.FindByIDIncludingDeleted(ctx, emp.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("creates audit log entry before deletion", func(t *testing.T) {
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

		// Audit log should still exist
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'employee' AND record_id = ? AND action = 'HARD_DELETE'",
			emp.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}
