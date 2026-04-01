package application

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/fman/internal/domain"

	sqlite "github.com/iamoeg/fman/internal/adapter/sqlite"
)

// ===============================================================================
// REPOSITORY INTERFACE
// ===============================================================================

// employeeRepository defines the minimal interface that EmployeeService
// needs from its persistence layer.
//
// The repository is responsible for:
//   - Persisting employee data
//   - Handling database transactions
//   - Creating audit logs
//   - Managing soft deletes
//   - Generating serial numbers per organization
type employeeRepository interface {
	// Create persists a new employee to the database.
	// Returns ErrDuplicateRecord if CIN or CNSS number already exists.
	// Returns error if foreign keys (org_id, compensation_package_id) are invalid.
	Create(ctx context.Context, emp *domain.Employee) error

	// Update modifies an existing employee in the database.
	// Note: org_id and serial_num are immutable and excluded from UPDATE.
	// Returns ErrRecordNotFound if employee doesn't exist or is soft-deleted.
	// Returns ErrDuplicateRecord if updated CIN/CNSS conflicts.
	Update(ctx context.Context, emp *domain.Employee) error

	// Delete soft-deletes an employee by setting deleted_at timestamp.
	// Returns ErrRecordNotFound if employee doesn't exist or is already deleted.
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore un-deletes a soft-deleted employee by clearing deleted_at.
	// Returns ErrRecordNotFound if employee doesn't exist or is not deleted.
	Restore(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes an employee from the database.
	// This is irreversible and should only be used for data purging (GDPR, test cleanup).
	// Returns ErrRecordNotFound if employee doesn't exist.
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves an employee by their ID.
	// Only returns active (non-deleted) employees.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error)

	// FindByIDIncludingDeleted retrieves an employee by their ID,
	// including soft-deleted employees.
	// Returns ErrRecordNotFound if not found.
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Employee, error)

	// FindAll retrieves all active (non-deleted) employees.
	// Returns empty slice if none found.
	FindAll(ctx context.Context) ([]*domain.Employee, error)

	// FindAllIncludingDeleted retrieves all employees,
	// including soft-deleted ones.
	// Returns empty slice if none found.
	FindAllIncludingDeleted(ctx context.Context) ([]*domain.Employee, error)

	// GetNextSerialNumber returns the next available serial number
	// for employees in the given organization.
	// Serial numbers start at 1 and increment for each new employee.
	GetNextSerialNumber(ctx context.Context, orgID uuid.UUID) (int, error)

	// FindByOrgAndSerialNum retrieves an employee by organization and serial number.
	// Only returns active (non-deleted) employees.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByOrgAndSerialNum(ctx context.Context, orgID uuid.UUID, serialNum int) (*domain.Employee, error)

	// FindByOrganization retrieves all active employees in an organization.
	// Returns empty slice if none found.
	FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error)

	// FindByOrganizationIncludingDeleted retrieves all employees in an organization,
	// including soft-deleted ones.
	// Returns empty slice if none found.
	FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error)
}

// ===============================================================================
// SERVICE IMPLEMENTATION
// ===============================================================================

// EmployeeService provides business logic for managing employees.
// It orchestrates domain validation, UUID generation, timestamp management,
// serial number generation, and persistence operations.
//
// Responsibilities:
//   - Generate UUIDs and timestamps
//   - Generate per-organization serial numbers
//   - Validate domain business rules
//   - Coordinate with repository layer
//   - Translate repository errors to service-level errors
//
// Key Features:
//   - Multi-tenant: All employees belong to an organization
//   - Serial numbers: Auto-generated per organization (Employee #1, #2, etc.)
//   - Immutable fields: org_id and serial_num cannot be changed after creation
type EmployeeService struct {
	repo employeeRepository
}

// NewEmployeeService creates a new EmployeeService with the given repository.
//
// The repository parameter should be an implementation of employeeRepository,
// typically *sqlite.EmployeeRepository in production or a mock in tests.
func NewEmployeeService(repo employeeRepository) *EmployeeService {
	return &EmployeeService{
		repo: repo,
	}
}

// ===============================================================================
// CREATE OPERATIONS
// ===============================================================================

// CreateEmployee creates a new employee in the system.
//
// The service will:
//  1. Generate a UUID if emp.ID is uuid.Nil
//  2. Generate the next serial number for the employee's organization
//  3. Set CreatedAt, UpdatedAt timestamps to current UTC time
//  4. Ensure DeletedAt is nil (not soft-deleted)
//  5. Validate all domain business rules
//  6. Persist to database
//
// The emp parameter is modified in-place, allowing the caller to access
// the generated ID, serial number, and timestamps after creation.
//
// Important: emp.OrgID and emp.CompensationPackageID must be set before calling.
// The service does not validate that these IDs exist - the database foreign key
// constraints will enforce this.
//
// Returns:
//   - ErrEmployeeExists if CIN or CNSS number already exists
//   - Domain validation errors if business rules are violated
//   - Foreign key constraint errors if org_id or compensation_package_id are invalid
//   - Wrapped repository errors for other failures
//
// Example:
//
//	emp := &domain.Employee{
//	    OrgID: orgID,
//	    FullName: "Ahmed Ali",
//	    CINNum: "AB123456",
//	    BirthDate: time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC),
//	    HireDate: time.Now().UTC(),
//	    Gender: domain.GenderMale,
//	    MaritalStatus: domain.MaritalStatusSingle,
//	    Position: "Software Engineer",
//	    CompensationPackageID: pkgID,
//	}
//	err := service.CreateEmployee(ctx, emp)
//	fmt.Println(emp.SerialNum) // Auto-generated serial number
func (s *EmployeeService) CreateEmployee(
	ctx context.Context,
	emp *domain.Employee,
) error {
	// 1. Generate UUID if not provided
	if emp.ID == uuid.Nil {
		emp.ID = uuid.New()
	}

	// 2. Generate serial number for this organization
	serialNum, err := s.repo.GetNextSerialNumber(ctx, emp.OrgID)
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}
	emp.SerialNum = serialNum

	// 3. Set timestamps
	now := time.Now().UTC()
	emp.CreatedAt = now
	emp.UpdatedAt = now
	emp.DeletedAt = nil

	// 4. Validate domain rules
	if err := emp.Validate(); err != nil {
		return fmt.Errorf("invalid employee: %w", err)
	}

	// 5. Persist
	if err := s.repo.Create(ctx, emp); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrDuplicateRecord) {
			return ErrEmployeeExists
		}
		return fmt.Errorf("failed to create employee: %w", err)
	}

	return nil
}

// ===============================================================================
// UPDATE OPERATIONS
// ===============================================================================

// UpdateEmployee updates an existing employee in the system.
//
// The service will:
//  1. Update the UpdatedAt timestamp to current UTC time
//  2. Validate all domain business rules
//  3. Persist changes to database
//
// The emp.ID must be set to the employee being updated.
// The emp.CreatedAt and emp.DeletedAt fields are not modified.
//
// Important: emp.OrgID and emp.SerialNum are immutable and CANNOT be changed.
// The repository's UPDATE query excludes these fields, so even if you modify
// them in the emp object, they will not be updated in the database.
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist or is soft-deleted
//   - ErrEmployeeExists if updated CIN/CNSS number conflicts with another employee
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	emp, _ := service.GetEmployee(ctx, empID)
//	emp.Position = "Senior Software Engineer"
//	emp.CompensationPackageID = newPkgID // Promotion!
//	err := service.UpdateEmployee(ctx, emp)
func (s *EmployeeService) UpdateEmployee(
	ctx context.Context,
	emp *domain.Employee,
) error {
	// 1. Update timestamp
	emp.UpdatedAt = time.Now().UTC()

	// 2. Validate domain rules
	if err := emp.Validate(); err != nil {
		return fmt.Errorf("invalid employee: %w", err)
	}

	// 3. Persist
	if err := s.repo.Update(ctx, emp); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		if errors.Is(err, sqlite.ErrDuplicateRecord) {
			return ErrEmployeeExists
		}
		return fmt.Errorf("failed to update employee: %w", err)
	}

	return nil
}

// ===============================================================================
// DELETE OPERATIONS
// ===============================================================================

// DeleteEmployee soft-deletes an employee by setting their deleted_at timestamp.
//
// Soft delete allows the employee to be restored later if needed and maintains
// referential integrity with related records (payroll results).
//
// Note: Due to CASCADE foreign keys, deleting an employee will also soft-delete:
//   - All payroll results for this employee
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist or is already deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.DeleteEmployee(ctx, empID)
func (s *EmployeeService) DeleteEmployee(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		return fmt.Errorf("failed to delete employee: %w", err)
	}
	return nil
}

// RestoreEmployee restores a soft-deleted employee by clearing their deleted_at timestamp.
//
// This operation reverses a soft delete, making the employee active again.
//
// Note: This does NOT automatically restore related records (payroll results).
// Those must be restored separately if needed.
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist or is not deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.RestoreEmployee(ctx, empID)
func (s *EmployeeService) RestoreEmployee(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.Restore(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		return fmt.Errorf("failed to restore employee: %w", err)
	}
	return nil
}

// HardDeleteEmployee permanently deletes an employee from the database.
//
// WARNING: This is irreversible. The employee and all audit logs are permanently removed.
//
// Use cases:
//   - GDPR compliance (right to be forgotten)
//   - Test data cleanup
//   - Data purging of very old soft-deleted records
//
// Note: Due to CASCADE foreign keys, hard deleting an employee will also permanently delete:
//   - All payroll results for this employee
//   - All audit logs for these records
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Only use for legitimate data purging
//	err := service.HardDeleteEmployee(ctx, empID)
func (s *EmployeeService) HardDeleteEmployee(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.HardDelete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrEmployeeNotFound
		}
		return fmt.Errorf("failed to hard delete employee: %w", err)
	}
	return nil
}

// ===============================================================================
// QUERY OPERATIONS
// ===============================================================================

// GetEmployee retrieves an employee by their ID.
//
// Only returns active (non-deleted) employees.
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	emp, err := service.GetEmployee(ctx, empID)
//	if errors.Is(err, ErrEmployeeNotFound) {
//	    // Handle not found
//	}
func (s *EmployeeService) GetEmployee(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Employee, error) {
	emp, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee: %w", err)
	}
	return emp, nil
}

// GetEmployeeIncludingDeleted retrieves an employee by their ID,
// including soft-deleted employees.
//
// This is useful for:
//   - Archive views in the TUI
//   - Audit trail examination
//   - Restore workflows (user needs to see deleted employees)
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist at all
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// For archive view showing all employees
//	emp, err := service.GetEmployeeIncludingDeleted(ctx, empID)
//	if emp.DeletedAt != nil {
//	    fmt.Println("This employee is archived")
//	}
func (s *EmployeeService) GetEmployeeIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Employee, error) {
	emp, err := s.repo.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee (including deleted): %w", err)
	}
	return emp, nil
}

// ListEmployees retrieves all active (non-deleted) employees across all organizations.
//
// Employees are returned in the order determined by the repository
// (typically by organization, then by serial number).
//
// Returns an empty slice if no employees exist.
//
// Note: For most use cases, you'll want ListEmployeesByOrganization instead,
// since employees are typically managed per organization.
//
// Returns:
//   - Empty slice if no employees found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	emps, err := service.ListEmployees(ctx)
//	for _, emp := range emps {
//	    fmt.Printf("#%d: %s\n", emp.SerialNum, emp.FullName)
//	}
func (s *EmployeeService) ListEmployees(
	ctx context.Context,
) ([]*domain.Employee, error) {
	emps, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees: %w", err)
	}
	return emps, nil
}

// ListEmployeesIncludingDeleted retrieves all employees across all organizations,
// including soft-deleted ones.
//
// This is useful for:
//   - Archive views in the TUI
//   - Administrative interfaces
//   - Audit and reporting
//
// Employees are returned in the order determined by the repository.
// Check emp.DeletedAt to distinguish active vs deleted employees.
//
// Returns an empty slice if no employees exist.
//
// Returns:
//   - Empty slice if no employees found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	emps, err := service.ListEmployeesIncludingDeleted(ctx)
//	for _, emp := range emps {
//	    if emp.DeletedAt != nil {
//	        fmt.Printf("#%d: %s (archived)\n", emp.SerialNum, emp.FullName)
//	    } else {
//	        fmt.Printf("#%d: %s\n", emp.SerialNum, emp.FullName)
//	    }
//	}
func (s *EmployeeService) ListEmployeesIncludingDeleted(
	ctx context.Context,
) ([]*domain.Employee, error) {
	emps, err := s.repo.FindAllIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees (including deleted): %w", err)
	}
	return emps, nil
}

// ===============================================================================
// EMPLOYEE-SPECIFIC QUERIES
// ===============================================================================

// GetEmployeeByOrgAndSerialNum retrieves an employee by their organization
// and serial number.
//
// This is the natural way to look up employees from a user's perspective,
// since serial numbers are displayed as "Employee #5" in the organization.
//
// Only returns active (non-deleted) employees.
//
// Returns:
//   - ErrEmployeeNotFound if employee doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// User selects "Employee #5" in the TUI
//	emp, err := service.GetEmployeeByOrgAndSerialNum(ctx, orgID, 5)
func (s *EmployeeService) GetEmployeeByOrgAndSerialNum(
	ctx context.Context,
	orgID uuid.UUID,
	serialNum int,
) (*domain.Employee, error) {
	emp, err := s.repo.FindByOrgAndSerialNum(ctx, orgID, serialNum)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrEmployeeNotFound
		}
		return nil, fmt.Errorf("failed to get employee by org and serial num: %w", err)
	}
	return emp, nil
}

// ListEmployeesByOrganization retrieves all active employees in an organization.
//
// This is the primary way to list employees, since employees are managed
// per organization in the TUI.
//
// Employees are returned ordered by serial number (Employee #1, #2, #3, ...).
//
// Returns an empty slice if the organization has no employees.
//
// Returns:
//   - Empty slice if no employees found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	emps, err := service.ListEmployeesByOrganization(ctx, orgID)
//	for _, emp := range emps {
//	    fmt.Printf("Employee #%d: %s\n", emp.SerialNum, emp.FullName)
//	}
func (s *EmployeeService) ListEmployeesByOrganization(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.Employee, error) {
	emps, err := s.repo.FindByOrganization(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees by organization: %w", err)
	}
	return emps, nil
}

// ListEmployeesByOrganizationIncludingDeleted retrieves all employees in an organization,
// including soft-deleted ones.
//
// This is useful for:
//   - Archive views showing former employees
//   - Audit trail examination
//   - HR reporting
//
// Employees are returned ordered by serial number.
// Check emp.DeletedAt to distinguish active vs deleted employees.
//
// Returns an empty slice if the organization has no employees.
//
// Returns:
//   - Empty slice if no employees found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	emps, err := service.ListEmployeesByOrganizationIncludingDeleted(ctx, orgID)
//	for _, emp := range emps {
//	    status := "Active"
//	    if emp.DeletedAt != nil {
//	        status = "Archived"
//	    }
//	    fmt.Printf("Employee #%d: %s (%s)\n", emp.SerialNum, emp.FullName, status)
//	}
func (s *EmployeeService) ListEmployeesByOrganizationIncludingDeleted(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.Employee, error) {
	emps, err := s.repo.FindByOrganizationIncludingDeleted(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list employees by organization (including deleted): %w", err)
	}
	return emps, nil
}

// ===============================================================================
// EMPLOYEE ERRORS
// ===============================================================================

var (
	// ErrEmployeeNotFound is returned when an employee cannot be found
	// by their ID or other identifying criteria.
	ErrEmployeeNotFound = errors.New("employee not found")

	// ErrEmployeeExists is returned when attempting to create an employee
	// with a CIN or CNSS number that already exists in the system.
	ErrEmployeeExists = errors.New("employee already exists")
)
