package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ===============================================================================
// REPOSITORY INTERFACES
// ===============================================================================

// payrollPeriodRepository defines the minimal interface that PayrollService
// needs for managing payroll periods.
type payrollPeriodRepository interface {
	// Create persists a new payroll period to the database.
	// Returns ErrDuplicateRecord if org/year/month combination already exists.
	Create(ctx context.Context, period *domain.PayrollPeriod) error

	// Delete soft-deletes a payroll period by setting deleted_at timestamp.
	// Returns ErrRecordNotFound if period doesn't exist or is already deleted.
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore un-deletes a soft-deleted payroll period.
	// Returns ErrRecordNotFound if period doesn't exist or is not deleted.
	Restore(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes a payroll period from the database.
	// Returns ErrRecordNotFound if period doesn't exist.
	HardDelete(ctx context.Context, id uuid.UUID) error

	// Finalize transitions a payroll period from DRAFT to FINALIZED status.
	// Sets finalized_at timestamp and updates updated_at.
	// Returns ErrRecordNotFound if period doesn't exist or is already finalized.
	Finalize(ctx context.Context, id uuid.UUID) error

	// Unfinalize transitions a payroll period from FINALIZED to DRAFT status.
	// Clears finalized_at timestamp and updates updated_at.
	// Returns ErrRecordNotFound if period doesn't exist or is not finalized.
	Unfinalize(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a payroll period by its ID.
	// Only returns active (non-deleted) periods.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error)

	// FindByIDIncludingDeleted retrieves a payroll period by its ID,
	// including soft-deleted periods.
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error)

	// FindByOrgYearMonth retrieves a payroll period by organization, year, and month.
	// Only returns active (non-deleted) periods.
	// Returns ErrRecordNotFound if not found.
	FindByOrgYearMonth(ctx context.Context, orgID uuid.UUID, year, month int) (*domain.PayrollPeriod, error)

	// FindByOrganization retrieves all payroll periods for an organization.
	// Only returns active (non-deleted) periods.
	FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error)

	// FindByOrganizationIncludingDeleted retrieves all payroll periods for an organization,
	// including soft-deleted ones.
	FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error)

	// FindAllDraft retrieves all payroll periods in DRAFT status.
	// Only returns active (non-deleted) periods.
	FindAllDraft(ctx context.Context) ([]*domain.PayrollPeriod, error)

	// FindAll retrieves all payroll periods.
	// Only returns active (non-deleted) periods.
	FindAll(ctx context.Context) ([]*domain.PayrollPeriod, error)

	// FindAllIncludingDeleted retrieves all payroll periods,
	// including soft-deleted ones.
	FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollPeriod, error)
}

// payrollResultRepository defines the minimal interface that PayrollService
// needs for managing payroll results.
type payrollResultRepository interface {
	// Create persists a new payroll result to the database.
	// Returns ErrDuplicateRecord if employee/period combination already exists.
	Create(ctx context.Context, result *domain.PayrollResult) error

	// Delete soft-deletes a payroll result by setting deleted_at timestamp.
	// Returns ErrRecordNotFound if result doesn't exist or is already deleted.
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore un-deletes a soft-deleted payroll result.
	// Returns ErrRecordNotFound if result doesn't exist or is not deleted.
	Restore(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes a payroll result from the database.
	// Returns ErrRecordNotFound if result doesn't exist.
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a payroll result by its ID.
	// Only returns active (non-deleted) results.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error)

	// FindByIDIncludingDeleted retrieves a payroll result by its ID,
	// including soft-deleted results.
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error)

	// FindByPeriod retrieves all payroll results for a specific period.
	// Only returns active (non-deleted) results.
	FindByPeriod(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error)

	// FindByPeriodIncludingDeleted retrieves all payroll results for a period,
	// including soft-deleted results.
	FindByPeriodIncludingDeleted(ctx context.Context, periodID uuid.UUID) ([]*domain.PayrollResult, error)

	// FindByEmployee retrieves all payroll results for an employee.
	// Only returns active (non-deleted) results.
	FindByEmployee(ctx context.Context, employeeID uuid.UUID) ([]*domain.PayrollResult, error)

	// FindByEmployeeIncludingDeleted retrieves all payroll results for an employee,
	// including soft-deleted results.
	FindByEmployeeIncludingDeleted(ctx context.Context, employeeID uuid.UUID) ([]*domain.PayrollResult, error)

	// FindAll retrieves all payroll results.
	// Only returns active (non-deleted) results.
	FindAll(ctx context.Context) ([]*domain.PayrollResult, error)

	// FindAllIncludingDeleted retrieves all payroll results,
	// including soft-deleted ones.
	FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollResult, error)

	// ReplaceAllForPeriod atomically replaces all payroll results for a period.
	// Soft-deletes any existing active results and creates all new ones in a
	// single transaction, so the period never has a partial set of results.
	ReplaceAllForPeriod(ctx context.Context, periodID uuid.UUID, results []*domain.PayrollResult) error
}

// ===============================================================================
// SERVICE IMPLEMENTATION
// ===============================================================================

// payrollCalculator defines the interface for the Moroccan payroll calculation engine.
// Satisfied by internal/adapter/payroll.Calculator.
type payrollCalculator interface {
	Calculate(
		ctx context.Context,
		period *domain.PayrollPeriod,
		emp *domain.Employee,
		pkg *domain.EmployeeCompensationPackage,
	) (*domain.PayrollResult, error)
}

// PayrollService provides business logic for managing payroll operations.
// It orchestrates payroll period lifecycle, result generation, and workflow state.
//
// Responsibilities:
//   - Generate UUIDs and timestamps
//   - Validate domain business rules
//   - Coordinate payroll period and result repositories
//   - Manage workflow state transitions (DRAFT → FINALIZED)
//   - Orchestrate payroll calculation (Phase 1E integration point)
//   - Translate repository errors to service-level errors
//
// This is the most complex service in the application layer because it:
//   - Coordinates multiple repositories
//   - Manages batch operations (generating results for all employees)
//   - Enforces workflow rules (cannot finalize empty periods, etc.)
//   - Will integrate with calculation engine in Phase 1E
type PayrollService struct {
	periods      payrollPeriodRepository
	results      payrollResultRepository
	employees    employeeRepository
	compensation compensationPackageRepository
	calculator   payrollCalculator
}

// NewPayrollService creates a new PayrollService with the given repositories.
//
// All repository parameters should be implementations of their respective interfaces,
// typically from the sqlite adapter package in production or mocks in tests.
func NewPayrollService(
	periods payrollPeriodRepository,
	results payrollResultRepository,
	employees employeeRepository,
	compensation compensationPackageRepository,
	calculator payrollCalculator,
) *PayrollService {
	return &PayrollService{
		periods:      periods,
		results:      results,
		employees:    employees,
		compensation: compensation,
		calculator:   calculator,
	}
}

// ===============================================================================
// PAYROLL PERIOD - CREATE OPERATIONS
// ===============================================================================

// CreatePayrollPeriod creates a new payroll period in the system.
//
// The service will:
//  1. Generate a UUID if period.ID is uuid.Nil
//  2. Set CreatedAt, UpdatedAt timestamps to current UTC time
//  3. Set Status to DRAFT (if not already set)
//  4. Ensure FinalizedAt and DeletedAt are nil
//  5. Validate all domain business rules
//  6. Persist to database
//
// The period parameter is modified in-place, allowing the caller to access
// the generated ID and timestamps after creation.
//
// Note: This does NOT automatically generate payroll results. Call
// GeneratePayrollResults() separately to populate the period with calculations.
//
// Returns:
//   - ErrPayrollPeriodExists if org/year/month combination already exists
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	period := &domain.PayrollPeriod{
//	    OrgID: orgID,
//	    Year: 2025,
//	    Month: 1,
//	}
//	err := service.CreatePayrollPeriod(ctx, period)
//	fmt.Println(period.ID) // UUID was generated
func (s *PayrollService) CreatePayrollPeriod(
	ctx context.Context,
	period *domain.PayrollPeriod,
) error {
	// 1. Generate UUID if not provided
	if period.ID == uuid.Nil {
		period.ID = uuid.New()
	}

	// 2. Set timestamps
	now := time.Now().UTC()
	period.CreatedAt = now
	period.UpdatedAt = now
	period.DeletedAt = nil

	// 3. Set initial status to DRAFT
	if period.Status == "" {
		period.Status = domain.PayrollPeriodStatusDraft
	}

	// 4. Ensure FinalizedAt is nil for DRAFT status
	period.FinalizedAt = nil

	// 5. Validate domain rules
	if err := period.Validate(); err != nil {
		return fmt.Errorf("invalid payroll period: %w", err)
	}

	// 6. Persist
	if err := s.periods.Create(ctx, period); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrDuplicateRecord) {
			return ErrPayrollPeriodExists
		}
		return fmt.Errorf("failed to create payroll period: %w", err)
	}

	return nil
}

// ===============================================================================
// PAYROLL PERIOD - DELETE OPERATIONS
// ===============================================================================

// DeletePayrollPeriod soft-deletes a payroll period by setting its deleted_at timestamp.
//
// Soft delete allows the period to be restored later if needed and maintains
// referential integrity with related payroll results.
//
// Note: Due to CASCADE foreign keys, deleting a period will also soft-delete
// all payroll results in that period.
//
// Important: Cannot delete a FINALIZED period. Must call UnfinalizePayrollPeriod()
// first to transition it back to DRAFT status.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is already deleted
//   - ErrPayrollPeriodAlreadyFinalized if trying to delete a finalized period
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.DeletePayrollPeriod(ctx, periodID)
func (s *PayrollService) DeletePayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) error {
	// 1. Get period to check status
	period, err := s.periods.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	// 2. Verify period is not finalized
	if period.Status == domain.PayrollPeriodStatusFinalized {
		return ErrPayrollPeriodAlreadyFinalized
	}

	// 3. Delete
	if err := s.periods.Delete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to delete payroll period: %w", err)
	}

	return nil
}

// RestorePayrollPeriod restores a soft-deleted payroll period by clearing its deleted_at timestamp.
//
// This operation reverses a soft delete, making the period active again.
//
// Note: This does NOT automatically restore related payroll results.
// Those must be restored separately if needed.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is not deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.RestorePayrollPeriod(ctx, periodID)
func (s *PayrollService) RestorePayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.periods.Restore(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to restore payroll period: %w", err)
	}
	return nil
}

// HardDeletePayrollPeriod permanently deletes a payroll period from the database.
//
// WARNING: This is irreversible. The period, all its results, and all audit logs
// are permanently removed.
//
// Use cases:
//   - GDPR compliance (right to be forgotten)
//   - Test data cleanup
//   - Data purging of very old soft-deleted records
//
// Note: Due to CASCADE foreign keys, hard deleting a period will also permanently delete:
//   - All payroll results in the period
//   - All audit logs for the period and its results
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Only use for legitimate data purging
//	err := service.HardDeletePayrollPeriod(ctx, periodID)
func (s *PayrollService) HardDeletePayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.periods.HardDelete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to hard delete payroll period: %w", err)
	}
	return nil
}

// ===============================================================================
// PAYROLL PERIOD - WORKFLOW OPERATIONS
// ===============================================================================

// FinalizePayrollPeriod transitions a payroll period from DRAFT to FINALIZED status.
//
// Finalization indicates that:
//   - All payroll calculations have been reviewed and approved
//   - Results are now immutable (cannot be modified)
//   - Period is ready for payment processing and reporting
//
// The service will:
//  1. Verify period exists and is in DRAFT status
//  2. Verify period has at least one payroll result
//  3. Set status to FINALIZED and set finalized_at timestamp
//  4. Persist changes
//
// Once finalized, a period cannot be deleted or have its results modified
// unless it is first unfinalized using UnfinalizePayrollPeriod().
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - ErrPayrollPeriodAlreadyFinalized if period is already finalized
//   - ErrPayrollPeriodEmpty if period has no results
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// After generating and reviewing payroll results
//	err := service.FinalizePayrollPeriod(ctx, periodID)
func (s *PayrollService) FinalizePayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) error {
	// 1. Get period to check status
	period, err := s.periods.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	// 2. Verify period is in DRAFT status
	if period.Status == domain.PayrollPeriodStatusFinalized {
		return ErrPayrollPeriodAlreadyFinalized
	}

	// 3. Verify period has at least one result
	results, err := s.results.FindByPeriod(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get payroll results: %w", err)
	}
	if len(results) == 0 {
		return ErrPayrollPeriodEmpty
	}

	// 4. Finalize (repository handles timestamp and status update)
	if err := s.periods.Finalize(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to finalize payroll period: %w", err)
	}

	return nil
}

// UnfinalizePayrollPeriod transitions a payroll period from FINALIZED to DRAFT status.
//
// This operation is used for error correction when:
//   - A calculation error is discovered after finalization
//   - Payroll needs to be regenerated
//   - Corrections are needed before payment processing
//
// The service will:
//  1. Verify period exists and is in FINALIZED status
//  2. Set status to DRAFT and clear finalized_at timestamp
//  3. Persist changes
//
// After unfinalizing, you can:
//   - Delete and regenerate payroll results
//   - Delete the period entirely
//   - Re-finalize after corrections
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - ErrPayrollPeriodNotFinalized if period is not currently finalized
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Discovered calculation error after finalization
//	err := service.UnfinalizePayrollPeriod(ctx, periodID)
//	// Delete results, regenerate, re-finalize
func (s *PayrollService) UnfinalizePayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) error {
	// 1. Get period to check status
	period, err := s.periods.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	// 2. Verify period is in FINALIZED status
	if period.Status != domain.PayrollPeriodStatusFinalized {
		return ErrPayrollPeriodNotFinalized
	}

	// 3. Unfinalize (repository handles timestamp and status update)
	if err := s.periods.Unfinalize(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to unfinalize payroll period: %w", err)
	}

	return nil
}

// ===============================================================================
// PAYROLL PERIOD - QUERY OPERATIONS
// ===============================================================================

// GetPayrollPeriod retrieves a payroll period by its ID.
//
// Only returns active (non-deleted) periods.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	period, err := service.GetPayrollPeriod(ctx, periodID)
func (s *PayrollService) GetPayrollPeriod(
	ctx context.Context,
	id uuid.UUID,
) (*domain.PayrollPeriod, error) {
	period, err := s.periods.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrPayrollPeriodNotFound
		}
		return nil, fmt.Errorf("failed to get payroll period: %w", err)
	}
	return period, nil
}

// GetPayrollPeriodIncludingDeleted retrieves a payroll period by its ID,
// including soft-deleted periods.
//
// This is useful for archive views and restore workflows.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist at all
//   - Wrapped repository errors for other failures
//
// Example:
//
//	period, err := service.GetPayrollPeriodIncludingDeleted(ctx, periodID)
//	if period.DeletedAt != nil {
//	    fmt.Println("This period is archived")
//	}
func (s *PayrollService) GetPayrollPeriodIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (*domain.PayrollPeriod, error) {
	period, err := s.periods.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrPayrollPeriodNotFound
		}
		return nil, fmt.Errorf("failed to get payroll period (including deleted): %w", err)
	}
	return period, nil
}

// GetPayrollPeriodByOrgYearMonth retrieves a payroll period by organization, year, and month.
//
// This is useful for looking up periods by their business identifiers
// rather than UUID.
//
// Only returns active (non-deleted) periods.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	period, err := service.GetPayrollPeriodByOrgYearMonth(ctx, orgID, 2025, 1)
func (s *PayrollService) GetPayrollPeriodByOrgYearMonth(
	ctx context.Context,
	orgID uuid.UUID,
	year, month int,
) (*domain.PayrollPeriod, error) {
	period, err := s.periods.FindByOrgYearMonth(ctx, orgID, year, month)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrPayrollPeriodNotFound
		}
		return nil, fmt.Errorf("failed to get payroll period by org/year/month: %w", err)
	}
	return period, nil
}

// ListPayrollPeriodsByOrganization retrieves all payroll periods for an organization.
//
// Only returns active (non-deleted) periods.
// Periods are returned in the order determined by the repository.
//
// Returns an empty slice if no periods exist.
//
// Returns:
//   - Empty slice if no periods found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	periods, err := service.ListPayrollPeriodsByOrganization(ctx, orgID)
func (s *PayrollService) ListPayrollPeriodsByOrganization(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.PayrollPeriod, error) {
	periods, err := s.periods.FindByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll periods: %w", err)
	}
	return periods, nil
}

// ListPayrollPeriodsByOrganizationIncludingDeleted retrieves all payroll periods
// for an organization, including soft-deleted ones.
//
// This is useful for archive views and administrative interfaces.
// Check period.DeletedAt to distinguish active vs deleted periods.
//
// Returns an empty slice if no periods exist.
//
// Returns:
//   - Empty slice if no periods found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	periods, err := service.ListPayrollPeriodsByOrganizationIncludingDeleted(ctx, orgID)
func (s *PayrollService) ListPayrollPeriodsByOrganizationIncludingDeleted(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.PayrollPeriod, error) {
	periods, err := s.periods.FindByOrganizationIncludingDeleted(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll periods (including deleted): %w", err)
	}
	return periods, nil
}

// ListDraftPayrollPeriods retrieves all payroll periods in DRAFT status.
//
// This is useful for:
//   - Finding periods that need review/finalization
//   - Administrative dashboards
//   - Automated period processing
//
// Only returns active (non-deleted) periods.
//
// Returns an empty slice if no draft periods exist.
//
// Returns:
//   - Empty slice if no draft periods found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	drafts, err := service.ListDraftPayrollPeriods(ctx)
//	fmt.Printf("Found %d periods awaiting finalization\n", len(drafts))
func (s *PayrollService) ListDraftPayrollPeriods(
	ctx context.Context,
) ([]*domain.PayrollPeriod, error) {
	periods, err := s.periods.FindAllDraft(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list draft payroll periods: %w", err)
	}
	return periods, nil
}

// ListPayrollPeriods retrieves all payroll periods in the system.
//
// Only returns active (non-deleted) periods.
//
// Returns an empty slice if no periods exist.
//
// Returns:
//   - Empty slice if no periods found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	periods, err := service.ListPayrollPeriods(ctx)
func (s *PayrollService) ListPayrollPeriods(
	ctx context.Context,
) ([]*domain.PayrollPeriod, error) {
	periods, err := s.periods.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll periods: %w", err)
	}
	return periods, nil
}

// ListPayrollPeriodsIncludingDeleted retrieves all payroll periods,
// including soft-deleted ones.
//
// Returns an empty slice if no periods exist.
//
// Returns:
//   - Empty slice if no periods found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	periods, err := service.ListPayrollPeriodsIncludingDeleted(ctx)
func (s *PayrollService) ListPayrollPeriodsIncludingDeleted(
	ctx context.Context,
) ([]*domain.PayrollPeriod, error) {
	periods, err := s.periods.FindAllIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll periods (including deleted): %w", err)
	}
	return periods, nil
}

// ===============================================================================
// PAYROLL RESULT - GENERATION OPERATIONS
// ===============================================================================

// GeneratePayrollResults generates payroll results for all employees in a period.
//
// This is the core payroll processing operation. It:
//  1. Validates the period exists and is in DRAFT status
//  2. Retrieves all active employees for the period's organization
//  3. Deletes any existing results for the period (if regenerating)
//  4. For each employee:
//     - Retrieves their current compensation package
//     - Calculates their payroll using the Moroccan calculation engine
//     - Creates a payroll result record
//
// If any step fails, the entire operation is rolled back and no results are created.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - ErrPayrollPeriodAlreadyFinalized if trying to generate for finalized period
//   - ErrPayrollCalculationFailed if calculation fails for any employee
//   - Wrapped repository errors for other failures
//
// Example:
//
//	period, _ := service.CreatePayrollPeriod(ctx, period)
//	err := service.GeneratePayrollResults(ctx, period.ID)
//	// Now period contains calculated results for all employees
func (s *PayrollService) GeneratePayrollResults(
	ctx context.Context,
	periodID uuid.UUID,
) error {
	// 1. Get and validate period
	period, err := s.periods.FindByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	// 2. Verify period is in DRAFT status
	if period.Status == domain.PayrollPeriodStatusFinalized {
		return ErrPayrollPeriodAlreadyFinalized
	}

	// 3. Get all active employees for the organization
	empList, err := s.employees.FindByOrganization(ctx, period.OrgID)
	if err != nil {
		return fmt.Errorf("failed to get employees: %w", err)
	}

	// 4. Calculate results for all employees (pure computation, no DB writes yet).
	// All calculations must succeed before any database changes are made.
	results := make([]*domain.PayrollResult, 0, len(empList))
	for _, emp := range empList {
		pkg, err := s.compensation.FindByID(ctx, emp.CompensationPackageID)
		if err != nil {
			return fmt.Errorf("failed to get compensation package for employee %s: %w", emp.ID, err)
		}

		result, err := s.calculator.Calculate(ctx, period, emp, pkg)
		if err != nil {
			return fmt.Errorf("%w for employee %s: %v", ErrPayrollCalculationFailed, emp.ID, err)
		}

		results = append(results, result)
	}

	// 5. Atomically replace existing results with the newly calculated ones.
	// Either all results are replaced or none are (the period is never left partial).
	if err := s.results.ReplaceAllForPeriod(ctx, periodID, results); err != nil {
		return fmt.Errorf("failed to replace payroll results: %w", err)
	}

	return nil
}

// DeletePayrollResults deletes all payroll results for a specific period.
//
// This is useful when:
//   - Regenerating payroll after corrections
//   - Clearing draft calculations
//
// Important: Can only delete results for DRAFT periods. If the period is
// FINALIZED, it must be unfinalized first.
//
// Returns:
//   - ErrPayrollPeriodNotFound if period doesn't exist or is soft-deleted
//   - ErrPayrollPeriodAlreadyFinalized if trying to delete results from finalized period
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Delete existing results before regenerating
//	err := service.DeletePayrollResults(ctx, periodID)
//	err = service.GeneratePayrollResults(ctx, periodID)
func (s *PayrollService) DeletePayrollResults(
	ctx context.Context,
	periodID uuid.UUID,
) error {
	// 1. Get and validate period
	period, err := s.periods.FindByID(ctx, periodID)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrPayrollPeriodNotFound
		}
		return fmt.Errorf("failed to get payroll period: %w", err)
	}

	// 2. Verify period is in DRAFT status
	if period.Status == domain.PayrollPeriodStatusFinalized {
		return ErrPayrollPeriodAlreadyFinalized
	}

	// 3. Get all results for period
	results, err := s.results.FindByPeriod(ctx, periodID)
	if err != nil {
		return fmt.Errorf("failed to get payroll results: %w", err)
	}

	// 4. Delete each result
	for _, result := range results {
		if err := s.results.Delete(ctx, result.ID); err != nil {
			if errors.Is(err, sqlite.ErrRecordNotFound) {
				continue // Already deleted
			}
			return fmt.Errorf("failed to delete result %s: %w", result.ID, err)
		}
	}

	return nil
}

// ===============================================================================
// PAYROLL RESULT - QUERY OPERATIONS
// ===============================================================================

// GetPayrollResult retrieves a payroll result by its ID.
//
// Only returns active (non-deleted) results.
//
// Returns:
//   - ErrPayrollResultNotFound if result doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	result, err := service.GetPayrollResult(ctx, resultID)
func (s *PayrollService) GetPayrollResult(
	ctx context.Context,
	id uuid.UUID,
) (*domain.PayrollResult, error) {
	result, err := s.results.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrPayrollResultNotFound
		}
		return nil, fmt.Errorf("failed to get payroll result: %w", err)
	}
	return result, nil
}

// GetPayrollResultIncludingDeleted retrieves a payroll result by its ID,
// including soft-deleted results.
//
// This is useful for archive views and audit trails.
//
// Returns:
//   - ErrPayrollResultNotFound if result doesn't exist at all
//   - Wrapped repository errors for other failures
//
// Example:
//
//	result, err := service.GetPayrollResultIncludingDeleted(ctx, resultID)
func (s *PayrollService) GetPayrollResultIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (*domain.PayrollResult, error) {
	result, err := s.results.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrPayrollResultNotFound
		}
		return nil, fmt.Errorf("failed to get payroll result (including deleted): %w", err)
	}
	return result, nil
}

// ListPayrollResultsByPeriod retrieves all payroll results for a specific period.
//
// Only returns active (non-deleted) results.
// Results are returned in the order determined by the repository.
//
// Returns an empty slice if no results exist for the period.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResultsByPeriod(ctx, periodID)
//	fmt.Printf("Generated payroll for %d employees\n", len(results))
func (s *PayrollService) ListPayrollResultsByPeriod(
	ctx context.Context,
	periodID uuid.UUID,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindByPeriod(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results by period: %w", err)
	}
	return results, nil
}

// ListPayrollResultsByPeriodIncludingDeleted retrieves all payroll results for a period,
// including soft-deleted results.
//
// This is useful for archive views and audit trails.
//
// Returns an empty slice if no results exist for the period.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResultsByPeriodIncludingDeleted(ctx, periodID)
func (s *PayrollService) ListPayrollResultsByPeriodIncludingDeleted(
	ctx context.Context,
	periodID uuid.UUID,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindByPeriodIncludingDeleted(ctx, periodID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results by period (including deleted): %w", err)
	}
	return results, nil
}

// ListPayrollResultsByEmployee retrieves all payroll results for an employee.
//
// This provides the employee's complete payroll history across all periods.
//
// Only returns active (non-deleted) results.
// Results are returned in the order determined by the repository.
//
// Returns an empty slice if employee has no payroll results.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResultsByEmployee(ctx, employeeID)
//	fmt.Printf("Employee has %d payroll records\n", len(results))
func (s *PayrollService) ListPayrollResultsByEmployee(
	ctx context.Context,
	employeeID uuid.UUID,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindByEmployee(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results by employee: %w", err)
	}
	return results, nil
}

// ListPayrollResultsByEmployeeIncludingDeleted retrieves all payroll results for an employee,
// including soft-deleted results.
//
// This is useful for complete payroll history views and audit trails.
//
// Returns an empty slice if employee has no payroll results.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResultsByEmployeeIncludingDeleted(ctx, employeeID)
func (s *PayrollService) ListPayrollResultsByEmployeeIncludingDeleted(
	ctx context.Context,
	employeeID uuid.UUID,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindByEmployeeIncludingDeleted(ctx, employeeID)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results by employee (including deleted): %w", err)
	}
	return results, nil
}

// ListPayrollResults retrieves all payroll results in the system.
//
// Only returns active (non-deleted) results.
//
// Returns an empty slice if no results exist.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResults(ctx)
func (s *PayrollService) ListPayrollResults(
	ctx context.Context,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results: %w", err)
	}
	return results, nil
}

// ListPayrollResultsIncludingDeleted retrieves all payroll results,
// including soft-deleted ones.
//
// Returns an empty slice if no results exist.
//
// Returns:
//   - Empty slice if no results found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	results, err := service.ListPayrollResultsIncludingDeleted(ctx)
func (s *PayrollService) ListPayrollResultsIncludingDeleted(
	ctx context.Context,
) ([]*domain.PayrollResult, error) {
	results, err := s.results.FindAllIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list payroll results (including deleted): %w", err)
	}
	return results, nil
}

// ===============================================================================
// PAYROLL PERIOD ERRORS
// ===============================================================================

var (
	// ErrPayrollPeriodNotFound is returned when a payroll period cannot be found
	// by its ID or by organization/year/month criteria.
	ErrPayrollPeriodNotFound = errors.New("payroll period not found")

	// ErrPayrollPeriodExists is returned when attempting to create a payroll period
	// for an organization/year/month combination that already exists.
	ErrPayrollPeriodExists = errors.New("payroll period already exists")

	// ErrPayrollPeriodAlreadyFinalized is returned when attempting to finalize
	// a payroll period that is already in FINALIZED status.
	ErrPayrollPeriodAlreadyFinalized = errors.New("payroll period is already finalized")

	// ErrPayrollPeriodNotFinalized is returned when attempting to unfinalize
	// a payroll period that is not currently in FINALIZED status.
	ErrPayrollPeriodNotFinalized = errors.New("payroll period is not finalized")

	// ErrPayrollPeriodEmpty is returned when attempting to finalize a payroll period
	// that has no payroll results. At least one result must exist before finalization.
	ErrPayrollPeriodEmpty = errors.New("payroll period has no results and cannot be finalized")

	// ErrPayrollCalculationFailed is returned when payroll calculation fails for
	// an employee during result generation. This is a general error that wraps
	// more specific calculation errors.
	ErrPayrollCalculationFailed = errors.New("payroll calculation failed")
)

// ===============================================================================
// PAYROLL RESULT ERRORS
// ===============================================================================

var (
	// ErrPayrollResultNotFound is returned when a payroll result cannot be found
	// by its ID or other identifying criteria.
	ErrPayrollResultNotFound = errors.New("payroll result not found")

	// ErrPayrollResultExists is returned when attempting to create a payroll result
	// for an employee/period combination that already exists.
	ErrPayrollResultExists = errors.New("payroll result already exists")
)
