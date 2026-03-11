package sqlite_adapter_test

import (
	"context"
	"database/sql"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Test Setup Helpers
// ============================================================================

var payrollResultCounter int64

// createTestPayrollResult creates a valid test payroll result with realistic values.
func createTestPayrollResult(periodID, employeeID, compensationPkgID uuid.UUID) *domain.PayrollResult {
	now := time.Now().UTC()
	atomic.AddInt64(&payrollResultCounter, 1)

	// Base salary and calculations
	baseSalary, _ := money.FromMAD(8000.00)
	seniorityBonus, _ := money.FromMAD(400.00)
	grossSalary, _ := baseSalary.Add(seniorityBonus)
	totalOtherBonus, _ := money.FromMAD(0.00)
	grossSalaryGrandTotal, _ := grossSalary.Add(totalOtherBonus)

	// CNSS contributions (employee)
	socialAllowanceEmp, _ := money.FromMAD(168.00)    // ~2% of gross
	jobLossCompensationEmp, _ := money.FromMAD(42.00) // ~0.5% of gross
	totalCNSSEmp, _ := socialAllowanceEmp.Add(jobLossCompensationEmp)

	// CNSS contributions (employer)
	socialAllowanceEmpr, _ := money.FromMAD(336.00)    // ~4% of gross
	jobLossCompensationEmpr, _ := money.FromMAD(84.00) // ~1% of gross
	trainingTax, _ := money.FromMAD(134.40)            // ~1.6% of gross
	familyBenefits, _ := money.FromMAD(420.00)         // ~5% of gross
	totalCNSSEmpr, _ := socialAllowanceEmpr.Add(jobLossCompensationEmpr)
	totalCNSSEmpr, _ = totalCNSSEmpr.Add(trainingTax)
	totalCNSSEmpr, _ = totalCNSSEmpr.Add(familyBenefits)

	// AMO contributions
	amoEmp, _ := money.FromMAD(168.00)  // ~2% of gross
	amoEmpr, _ := money.FromMAD(252.00) // ~3% of gross

	// Tax calculations
	totalExemptions, _ := money.FromMAD(1680.00) // 20% professional expenses
	taxableGrossSalary, _ := grossSalaryGrandTotal.Subtract(totalExemptions)
	taxableNetSalary, _ := taxableGrossSalary.Subtract(totalCNSSEmp)
	taxableNetSalary, _ = taxableNetSalary.Subtract(amoEmp)
	incomeTax, _ := money.FromMAD(520.00) // Progressive tax calculation

	// Final amount
	netToPay, _ := grossSalaryGrandTotal.Subtract(totalCNSSEmp)
	netToPay, _ = netToPay.Subtract(amoEmp)
	netToPay, _ = netToPay.Subtract(incomeTax)
	roundingAmount, _ := money.FromMAD(0.00)

	return &domain.PayrollResult{
		ID:                                 uuid.New(),
		PayrollPeriodID:                    periodID,
		EmployeeID:                         employeeID,
		CompensationPackageID:              compensationPkgID,
		Currency:                           money.MAD,
		BaseSalary:                         baseSalary,
		SeniorityBonus:                     seniorityBonus,
		GrossSalary:                        grossSalary,
		TotalOtherBonus:                    totalOtherBonus,
		GrossSalaryGrandTotal:              grossSalaryGrandTotal,
		TotalExemptions:                    totalExemptions,
		TaxableGrossSalary:                 taxableGrossSalary,
		SocialAllowanceEmployeeContrib:     socialAllowanceEmp,
		SocialAllowanceEmployerContrib:     socialAllowanceEmpr,
		JobLossCompensationEmployeeContrib: jobLossCompensationEmp,
		JobLossCompensationEmployerContrib: jobLossCompensationEmpr,
		TrainingTaxEmployerContrib:         trainingTax,
		FamilyBenefitsEmployerContrib:      familyBenefits,
		TotalCNSSEmployeeContrib:           totalCNSSEmp,
		TotalCNSSEmployerContrib:           totalCNSSEmpr,
		AMOEmployeeContrib:                 amoEmp,
		AMOEmployerContrib:                 amoEmpr,
		TaxableNetSalary:                   taxableNetSalary,
		IncomeTax:                          incomeTax,
		RoundingAmount:                     roundingAmount,
		NetToPay:                           netToPay,
		CreatedAt:                          now,
		UpdatedAt:                          now,
	}
}

// setupPayrollTestData creates a complete test environment with org, employee, period, etc.
// Returns: orgID, employeeID, compensationPkgID, periodID
func setupPayrollTestData(t *testing.T, db *sql.DB) (orgID uuid.UUID, empID uuid.UUID, compPkgID uuid.UUID, periodID uuid.UUID) {
	t.Helper()

	ctx := context.Background()

	// Create organization
	orgRepo := sqlite.NewOrganizationRepository(db)
	org := createTestOrganization()
	err := orgRepo.Create(ctx, org)
	require.NoError(t, err)

	// Create compensation package
	compRepo := sqlite.NewCompensationPackageRepository(db)
	comp := createTestCompensationPackage(org.ID)
	err = compRepo.Create(ctx, comp)
	require.NoError(t, err)

	// Create employee with serial number 1
	empRepo := sqlite.NewEmployeeRepository(db)
	emp := createTestEmployee(org.ID, comp.ID, 1)
	err = empRepo.Create(ctx, emp)
	require.NoError(t, err)

	// Create payroll period
	periodRepo := sqlite.NewPayrollPeriodRepository(db)
	period := createTestPayrollPeriod(org.ID)
	err = periodRepo.Create(ctx, period)
	require.NoError(t, err)

	return org.ID, emp.ID, comp.ID, period.ID
}

// ============================================================================
// FindByID Tests
// ============================================================================

func TestPayrollResultRepository_FindByID(t *testing.T) {
	t.Parallel()

	t.Run("returns payroll result when found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		// Create payroll result
		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Find it
		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, result.ID, found.ID)
		require.Equal(t, result.PayrollPeriodID, found.PayrollPeriodID)
		require.Equal(t, result.EmployeeID, found.EmployeeID)
		require.True(t, result.NetToPay.Equals(found.NetToPay))
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByID(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("does not return soft-deleted payroll results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		// Create and delete payroll result
		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		// Should not find it
		found, err := repo.FindByID(ctx, result.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByIDIncludingDeleted Tests
// ============================================================================

func TestPayrollResultRepository_FindByIDIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns active payroll result", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, result.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, result.ID, found.ID)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("returns soft-deleted payroll result", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		found, err := repo.FindByIDIncludingDeleted(ctx, result.ID)
		require.NoError(t, err)
		require.NotNil(t, found)
		require.Equal(t, result.ID, found.ID)
		require.NotNil(t, found.DeletedAt) // Should have deleted_at set
	})

	t.Run("returns ErrRecordNotFound when not found", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		found, err := repo.FindByIDIncludingDeleted(ctx, randomID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// FindByPeriod Tests
// ============================================================================

func TestPayrollResultRepository_FindByPeriod(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no results for period", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, _, _, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		results, err := repo.FindByPeriod(ctx, periodID)
		require.NoError(t, err)
		require.NotNil(t, results)
		require.Len(t, results, 0)
	})

	t.Run("returns all active results for period", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create 3 employees with unique serial numbers
		emp1 := createTestEmployee(orgID, compID, 2)
		emp2 := createTestEmployee(orgID, compID, 3)
		emp3 := createTestEmployee(orgID, compID, 4)
		require.NoError(t, empRepo.Create(ctx, emp1))
		require.NoError(t, empRepo.Create(ctx, emp2))
		require.NoError(t, empRepo.Create(ctx, emp3))

		// Create payroll results for all 3 (unique period+employee combinations)
		result1 := createTestPayrollResult(periodID, emp1.ID, compID)
		result2 := createTestPayrollResult(periodID, emp2.ID, compID)
		result3 := createTestPayrollResult(periodID, emp3.ID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		results, err := repo.FindByPeriod(ctx, periodID)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})

	t.Run("does not return soft-deleted results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create 2 employees with unique serial numbers
		emp1 := createTestEmployee(orgID, compID, 2)
		emp2 := createTestEmployee(orgID, compID, 3)
		require.NoError(t, empRepo.Create(ctx, emp1))
		require.NoError(t, empRepo.Create(ctx, emp2))

		// Create 2 results (unique period+employee), delete 1
		result1 := createTestPayrollResult(periodID, emp1.ID, compID)
		result2 := createTestPayrollResult(periodID, emp2.ID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindByPeriod(ctx, periodID)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, result2.ID, results[0].ID)
	})
}

// ============================================================================
// FindByPeriodIncludingDeleted Tests
// ============================================================================

func TestPayrollResultRepository_FindByPeriodIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all results including soft-deleted for period", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, _, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		empRepo := sqlite.NewEmployeeRepository(db)
		ctx := context.Background()

		// Create 3 employees with unique serial numbers
		emp1 := createTestEmployee(orgID, compID, 2)
		emp2 := createTestEmployee(orgID, compID, 3)
		emp3 := createTestEmployee(orgID, compID, 4)
		require.NoError(t, empRepo.Create(ctx, emp1))
		require.NoError(t, empRepo.Create(ctx, emp2))
		require.NoError(t, empRepo.Create(ctx, emp3))

		// Create 3 results (unique period+employee), delete 1
		result1 := createTestPayrollResult(periodID, emp1.ID, compID)
		result2 := createTestPayrollResult(periodID, emp2.ID, compID)
		result3 := createTestPayrollResult(periodID, emp3.ID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindByPeriodIncludingDeleted(ctx, periodID)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})
}

// ============================================================================
// FindByEmployee Tests
// ============================================================================

func TestPayrollResultRepository_FindByEmployee(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no payroll history", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, _, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		results, err := repo.FindByEmployee(ctx, empID)
		require.NoError(t, err)
		require.NotNil(t, results)
		require.Len(t, results, 0)
	})

	t.Run("returns payroll history for employee across periods", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 3 periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		period3 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))
		require.NoError(t, periodRepo.Create(ctx, period3))

		// Create payroll results for same employee in all 3 periods
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		result3 := createTestPayrollResult(period3.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		results, err := repo.FindByEmployee(ctx, empID)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})

	t.Run("does not return soft-deleted results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 2 periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))

		// Create 2 results, delete 1
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindByEmployee(ctx, empID)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, result2.ID, results[0].ID)
	})
}

// ============================================================================
// FindByEmployeeIncludingDeleted Tests
// ============================================================================

func TestPayrollResultRepository_FindByEmployeeIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all payroll history including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 3 periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		period3 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))
		require.NoError(t, periodRepo.Create(ctx, period3))

		// Create 3 results, delete 1
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		result3 := createTestPayrollResult(period3.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindByEmployeeIncludingDeleted(ctx, empID)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})
}

// ============================================================================
// FindAll Tests
// ============================================================================

func TestPayrollResultRepository_FindAll(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice when no results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		results, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.NotNil(t, results)
		require.Len(t, results, 0)
	})

	t.Run("returns all active payroll results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 3 different periods for the same employee
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		period3 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))
		require.NoError(t, periodRepo.Create(ctx, period3))

		// Create payroll results in different periods (unique period+employee)
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		result3 := createTestPayrollResult(period3.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		results, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, results, 3)
	})

	t.Run("does not return soft-deleted results", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 2 different periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))

		// Create 2 results in different periods, delete 1
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindAll(ctx)
		require.NoError(t, err)
		require.Len(t, results, 1)
		require.Equal(t, result2.ID, results[0].ID)
	})
}

// ============================================================================
// FindAllIncludingDeleted Tests
// ============================================================================

func TestPayrollResultRepository_FindAllIncludingDeleted(t *testing.T) {
	t.Parallel()

	t.Run("returns all results including soft-deleted", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		orgID, empID, compID, _ := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		periodRepo := sqlite.NewPayrollPeriodRepository(db)
		ctx := context.Background()

		// Create 3 different periods
		period1 := createTestPayrollPeriod(orgID)
		period2 := createTestPayrollPeriod(orgID)
		period3 := createTestPayrollPeriod(orgID)
		require.NoError(t, periodRepo.Create(ctx, period1))
		require.NoError(t, periodRepo.Create(ctx, period2))
		require.NoError(t, periodRepo.Create(ctx, period3))

		// Create 3 results in different periods, delete 1
		result1 := createTestPayrollResult(period1.ID, empID, compID)
		result2 := createTestPayrollResult(period2.ID, empID, compID)
		result3 := createTestPayrollResult(period3.ID, empID, compID)
		require.NoError(t, repo.Create(ctx, result1))
		require.NoError(t, repo.Create(ctx, result2))
		require.NoError(t, repo.Create(ctx, result3))

		err := repo.Delete(ctx, result1.ID)
		require.NoError(t, err)

		results, err := repo.FindAllIncludingDeleted(ctx)
		require.NoError(t, err)
		require.Len(t, results, 3)

		// Check that one has deleted_at set
		deletedCount := 0
		for _, r := range results {
			if r.DeletedAt != nil {
				deletedCount++
			}
		}
		require.Equal(t, 1, deletedCount)
	})
}

// ============================================================================
// Create Tests
// ============================================================================

func TestPayrollResultRepository_Create(t *testing.T) {
	t.Parallel()

	t.Run("creates payroll result successfully", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Verify it was created
		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)
		require.Equal(t, result.ID, found.ID)
		require.True(t, result.NetToPay.Equals(found.NetToPay))
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'CREATE'",
			result.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("stores all monetary values as cents", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Query database directly to verify cents storage
		var baseSalaryCents int64
		err = db.QueryRow(
			"SELECT base_salary_cents FROM payroll_result WHERE id = ?",
			result.ID.String(),
		).Scan(&baseSalaryCents)
		require.NoError(t, err)
		require.Equal(t, result.BaseSalary.Cents(), baseSalaryCents)
	})
}

// ============================================================================
// Delete Tests
// ============================================================================

func TestPayrollResultRepository_Delete(t *testing.T) {
	t.Parallel()

	t.Run("soft deletes payroll result", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		// Should not be found by regular FindByID
		found, err := repo.FindByID(ctx, result.ID)
		require.Error(t, err)
		require.Nil(t, found)

		// Should be found with IncludingDeleted
		foundDeleted, err := repo.FindByIDIncludingDeleted(ctx, result.ID)
		require.NoError(t, err)
		require.NotNil(t, foundDeleted.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'DELETE'",
			result.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})

	t.Run("returns ErrRecordNotFound when result does not exist", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		randomID := uuid.New()
		err := repo.Delete(ctx, randomID)
		require.Error(t, err)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})
}

// ============================================================================
// Restore Tests
// ============================================================================

func TestPayrollResultRepository_Restore(t *testing.T) {
	t.Parallel()

	t.Run("restores soft-deleted payroll result", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		// Delete
		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		// Restore
		err = repo.Restore(ctx, result.ID)
		require.NoError(t, err)

		// Should be found now
		found, err := repo.FindByID(ctx, result.ID)
		require.NoError(t, err)
		require.Nil(t, found.DeletedAt)
	})

	t.Run("creates audit log entry", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.Delete(ctx, result.ID)
		require.NoError(t, err)

		err = repo.Restore(ctx, result.ID)
		require.NoError(t, err)

		// Check audit log
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'RESTORE'",
			result.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}

// ============================================================================
// HardDelete Tests
// ============================================================================

func TestPayrollResultRepository_HardDelete(t *testing.T) {
	t.Parallel()

	t.Run("permanently deletes payroll result", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, result.ID)
		require.NoError(t, err)

		// Should not be found even with IncludingDeleted
		found, err := repo.FindByIDIncludingDeleted(ctx, result.ID)
		require.Error(t, err)
		require.Nil(t, found)
		require.ErrorIs(t, err, sqlite.ErrRecordNotFound)
	})

	t.Run("creates audit log entry before deletion", func(t *testing.T) {
		t.Parallel()

		db := setupTestDB(t)
		defer db.Close()

		_, empID, compID, periodID := setupPayrollTestData(t, db)
		repo := sqlite.NewPayrollResultRepository(db)
		ctx := context.Background()

		result := createTestPayrollResult(periodID, empID, compID)
		err := repo.Create(ctx, result)
		require.NoError(t, err)

		err = repo.HardDelete(ctx, result.ID)
		require.NoError(t, err)

		// Audit log should still exist
		var count int
		err = db.QueryRow(
			"SELECT COUNT(*) FROM audit_log WHERE table_name = 'payroll_result' AND record_id = ? AND action = 'HARD_DELETE'",
			result.ID.String(),
		).Scan(&count)
		require.NoError(t, err)
		require.Equal(t, 1, count)
	})
}
