package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/iamoeg/bootdev-capstone/internal/domain"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
)

// ===============================================================================
// REPOSITORY INTERFACE
// ===============================================================================

// compensationPackageRepository defines the minimal interface that
// CompensationPackageService needs from its persistence layer.
//
// The repository is responsible for:
//   - Persisting compensation package data
//   - Handling database transactions
//   - Creating audit logs
//   - Managing soft deletes
//   - Checking usage by employees and payroll results (historical protection)
type compensationPackageRepository interface {
	// Create persists a new compensation package to the database.
	Create(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error

	// Update modifies an existing compensation package in the database.
	// Returns ErrRecordNotFound if package doesn't exist or is soft-deleted.
	// Returns ErrCompensationPackageInUse if package is referenced by employees
	// or payroll results (historical artifact protection).
	Update(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error

	// Rename updates only the name field of an existing compensation package.
	// Unlike Update, it does NOT enforce in-use guards — name is display metadata.
	// Returns ErrRecordNotFound if package doesn't exist or is soft-deleted.
	Rename(ctx context.Context, id uuid.UUID, name string) error

	// Delete soft-deletes a compensation package by setting deleted_at timestamp.
	// Returns ErrRecordNotFound if package doesn't exist or is already deleted.
	// Returns ErrCompensationPackageInUse if package is referenced by employees
	// or payroll results.
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore un-deletes a soft-deleted compensation package by clearing deleted_at.
	// Returns ErrRecordNotFound if package doesn't exist or is not deleted.
	Restore(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes a compensation package from the database.
	// This is irreversible and should only be used for data purging (GDPR, test cleanup).
	// Returns ErrRecordNotFound if package doesn't exist.
	// Returns ErrCompensationPackageInUse if package is referenced by employees
	// or payroll results.
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves a compensation package by its ID.
	// Only returns active (non-deleted) packages.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error)

	// FindByIDIncludingDeleted retrieves a compensation package by its ID,
	// including soft-deleted packages.
	// Returns ErrRecordNotFound if not found.
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error)

	// FindAll retrieves all active (non-deleted) compensation packages for the given org.
	// Returns empty slice if none found.
	FindAll(ctx context.Context, orgID uuid.UUID) ([]*domain.EmployeeCompensationPackage, error)

	// FindAllIncludingDeleted retrieves all compensation packages for the given org,
	// including soft-deleted ones.
	// Returns empty slice if none found.
	FindAllIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.EmployeeCompensationPackage, error)

	// CountEmployeesUsing returns the count of employees (including soft-deleted)
	// that reference this compensation package.
	// This is used to enforce historical artifact protection.
	CountEmployeesUsing(ctx context.Context, pkgID uuid.UUID) (int64, error)

	// CountPayrollResultsUsing returns the count of payroll results
	// that reference this compensation package.
	// This is used to enforce historical artifact protection.
	CountPayrollResultsUsing(ctx context.Context, pkgID uuid.UUID) (int64, error)
}

// ===============================================================================
// SERVICE IMPLEMENTATION
// ===============================================================================

// CompensationPackageService provides business logic for managing employee
// compensation packages.
//
// It orchestrates domain validation, UUID generation, timestamp management,
// and persistence operations, with special emphasis on historical artifact
// protection.
//
// Responsibilities:
//   - Generate UUIDs and timestamps
//   - Validate domain business rules (SMIG, currency, positive amounts)
//   - Guard against modifying packages in use (historical protection)
//   - Coordinate with repository layer
//   - Translate repository errors to service-level errors
//
// Historical Artifact Protection:
//
// Compensation packages become immutable once referenced by:
//   - Employees (active or soft-deleted)
//   - Payroll results (immutable records)
//
// This ensures payroll calculations can always be traced back to the exact
// compensation that was used, which is critical for:
//   - Legal compliance and audit trails
//   - Historical accuracy of payroll records
//   - Tax and social security reporting
//
// Workflow for Modifying In-Use Packages:
//
// If you need to change a compensation package that is in use:
//  1. Create a new package with the desired values
//  2. Reassign employees from old package to new package (using EmployeeService)
//  3. Optionally delete the old package (if no longer needed)
//
// This preserves historical integrity while allowing business changes.
type CompensationPackageService struct {
	repo compensationPackageRepository
}

// NewCompensationPackageService creates a new CompensationPackageService
// with the given repository.
//
// The repository parameter should be an implementation of
// compensationPackageRepository, typically *sqlite.EmployeeCompensationPackageRepository
// in production or a mock in tests.
func NewCompensationPackageService(repo compensationPackageRepository) *CompensationPackageService {
	return &CompensationPackageService{
		repo: repo,
	}
}

// ===============================================================================
// CREATE OPERATIONS
// ===============================================================================

// CreateCompensationPackage creates a new compensation package in the system.
//
// The service will:
//  1. Generate a UUID if pkg.ID is uuid.Nil
//  2. Set CreatedAt, UpdatedAt timestamps to current UTC time
//  3. Ensure DeletedAt is nil (not soft-deleted)
//  4. Validate all domain business rules (SMIG, currency, positive amounts)
//  5. Persist to database
//
// The pkg parameter is modified in-place, allowing the caller to access
// the generated ID and timestamps after creation.
//
// Returns:
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	pkg := &domain.EmployeeCompensationPackage{
//	    BaseSalary: money.FromCents(500000), // 5000.00 MAD
//	    Currency: domain.CurrencyMAD,
//	}
//	err := service.CreateCompensationPackage(ctx, pkg)
//	fmt.Println(pkg.ID) // UUID was generated
func (s *CompensationPackageService) CreateCompensationPackage(
	ctx context.Context,
	pkg *domain.EmployeeCompensationPackage,
) error {
	// 1. Generate UUID if not provided
	if pkg.ID == uuid.Nil {
		pkg.ID = uuid.New()
	}

	// 2. Set timestamps
	now := time.Now().UTC()
	pkg.CreatedAt = now
	pkg.UpdatedAt = now
	pkg.DeletedAt = nil

	// 3. Validate domain rules
	if err := pkg.Validate(); err != nil {
		return fmt.Errorf("invalid compensation package: %w", err)
	}

	// 4. Persist
	if err := s.repo.Create(ctx, pkg); err != nil {
		return fmt.Errorf("failed to create compensation package: %w", err)
	}

	return nil
}

// ===============================================================================
// UPDATE OPERATIONS
// ===============================================================================

// UpdateCompensationPackage updates an existing compensation package in the system.
//
// The service will:
//  1. Check if package is in use by employees or payroll results
//  2. If in use, return ErrCompensationPackageInUse
//  3. Update the UpdatedAt timestamp to current UTC time
//  4. Validate all domain business rules
//  5. Persist changes to database
//
// The pkg.ID must be set to the package being updated.
// The pkg.CreatedAt and pkg.DeletedAt fields are not modified.
//
// Historical Protection:
//
// Updates are blocked if the package is referenced by:
//   - Any employees (active or soft-deleted)
//   - Any payroll results
//
// This ensures historical payroll accuracy is never compromised.
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist or is soft-deleted
//   - ErrCompensationPackageInUse if package is referenced (cannot be modified)
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	pkg, _ := service.GetCompensationPackage(ctx, pkgID)
//	newSalary, _ := money.FromMAD(5500.00)
//	pkg.BaseSalary = newSalary
//	err := service.UpdateCompensationPackage(ctx, pkg)
//	if errors.Is(err, ErrCompensationPackageInUse) {
//	    // Cannot modify - package is in use
//	    // User must create a new package instead
//	}
func (s *CompensationPackageService) UpdateCompensationPackage(
	ctx context.Context,
	pkg *domain.EmployeeCompensationPackage,
) error {
	// 1. Check if package is in use (historical protection)
	inUse, err := s.IsCompensationPackageInUse(ctx, pkg.ID)
	if err != nil {
		return err // Already wrapped by checkInUse
	}
	if inUse {
		return ErrCompensationPackageInUse
	}

	// 2. Update timestamp
	pkg.UpdatedAt = time.Now().UTC()

	// 3. Validate domain rules
	if err := pkg.Validate(); err != nil {
		return fmt.Errorf("invalid compensation package: %w", err)
	}

	// 4. Persist
	if err := s.repo.Update(ctx, pkg); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrCompensationPackageNotFound
		}
		// Repository also checks usage - defense in depth
		if errors.Is(err, sqlite.ErrCompensationPackageInUse) {
			return ErrCompensationPackageInUse
		}
		return fmt.Errorf("failed to update compensation package: %w", err)
	}

	return nil
}

// RenameCompensationPackage updates only the name of an existing compensation package.
// Unlike UpdateCompensationPackage, this is allowed even when the package is in use
// by employees or payroll results — the name is display metadata, not a financial term.
func (s *CompensationPackageService) RenameCompensationPackage(
	ctx context.Context,
	id uuid.UUID,
	name string,
) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("name is required")
	}
	if err := s.repo.Rename(ctx, id, name); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrCompensationPackageNotFound
		}
		return fmt.Errorf("failed to rename compensation package: %w", err)
	}
	return nil
}

// ===============================================================================
// DELETE OPERATIONS
// ===============================================================================

// DeleteCompensationPackage soft-deletes a compensation package by setting
// its deleted_at timestamp.
//
// Soft delete allows the package to be restored later if needed and maintains
// referential integrity with related records (employees, payroll results).
//
// Historical Protection:
//
// Deletion is blocked if the package is referenced by:
//   - Any employees (active or soft-deleted)
//   - Any payroll results
//
// This ensures historical payroll accuracy is never compromised.
// Even soft-deleted packages must remain intact if referenced.
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist or is already deleted
//   - ErrCompensationPackageInUse if package is referenced (cannot be deleted)
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.DeleteCompensationPackage(ctx, pkgID)
//	if errors.Is(err, ErrCompensationPackageInUse) {
//	    // Cannot delete - employees are using this package
//	    // User must first reassign employees to different package
//	}
func (s *CompensationPackageService) DeleteCompensationPackage(
	ctx context.Context,
	id uuid.UUID,
) error {
	// 1. Check if package is in use (historical protection)
	inUse, err := s.IsCompensationPackageInUse(ctx, id)
	if err != nil {
		return err // Already wrapped by checkInUse
	}
	if inUse {
		return ErrCompensationPackageInUse
	}

	// 2. Persist soft delete
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrCompensationPackageNotFound
		}
		// Repository also checks usage - defense in depth
		if errors.Is(err, sqlite.ErrCompensationPackageInUse) {
			return ErrCompensationPackageInUse
		}
		return fmt.Errorf("failed to delete compensation package: %w", err)
	}

	return nil
}

// RestoreCompensationPackage restores a soft-deleted compensation package
// by clearing its deleted_at timestamp.
//
// This operation reverses a soft delete, making the package active again.
//
// No usage check is performed on restore because:
//   - Packages can only be deleted if not in use (enforced by Delete)
//   - Therefore, restored packages cannot be in an invalid state
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist or is not deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.RestoreCompensationPackage(ctx, pkgID)
func (s *CompensationPackageService) RestoreCompensationPackage(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.Restore(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrCompensationPackageNotFound
		}
		return fmt.Errorf("failed to restore compensation package: %w", err)
	}
	return nil
}

// HardDeleteCompensationPackage permanently deletes a compensation package
// from the database.
//
// WARNING: This is irreversible. The package and all audit logs are permanently removed.
//
// Historical Protection:
//
// Hard deletion is blocked if the package is referenced by:
//   - Any employees (active or soft-deleted)
//   - Any payroll results
//
// This is the strictest protection - historical artifacts cannot be purged
// if they're part of any record (even soft-deleted records).
//
// Use cases:
//   - GDPR compliance (right to be forgotten) - only after all references removed
//   - Test data cleanup
//   - Data purging of very old, unused, soft-deleted packages
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist
//   - ErrCompensationPackageInUse if package is referenced (cannot be hard deleted)
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Only use for legitimate data purging
//	// Must first ensure no employees or payroll results reference this package
//	err := service.HardDeleteCompensationPackage(ctx, pkgID)
func (s *CompensationPackageService) HardDeleteCompensationPackage(
	ctx context.Context,
	id uuid.UUID,
) error {
	// 1. Check if package is in use (historical protection)
	// This is CRITICAL for hard delete - permanent removal
	inUse, err := s.IsCompensationPackageInUse(ctx, id)
	if err != nil {
		return err // Already wrapped by checkInUse
	}
	if inUse {
		return ErrCompensationPackageInUse
	}

	// 2. Persist hard delete
	if err := s.repo.HardDelete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrCompensationPackageNotFound
		}
		// Repository also checks usage - defense in depth
		if errors.Is(err, sqlite.ErrCompensationPackageInUse) {
			return ErrCompensationPackageInUse
		}
		return fmt.Errorf("failed to hard delete compensation package: %w", err)
	}

	return nil
}

// ===============================================================================
// QUERY OPERATIONS
// ===============================================================================

// GetCompensationPackage retrieves a compensation package by its ID.
//
// Only returns active (non-deleted) packages.
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	pkg, err := service.GetCompensationPackage(ctx, pkgID)
//	if errors.Is(err, ErrCompensationPackageNotFound) {
//	    // Handle not found
//	}
func (s *CompensationPackageService) GetCompensationPackage(
	ctx context.Context,
	id uuid.UUID,
) (*domain.EmployeeCompensationPackage, error) {
	pkg, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrCompensationPackageNotFound
		}
		return nil, fmt.Errorf("failed to get compensation package: %w", err)
	}
	return pkg, nil
}

// GetCompensationPackageIncludingDeleted retrieves a compensation package by its ID,
// including soft-deleted packages.
//
// This is useful for:
//   - Archive views in the TUI
//   - Audit trail examination
//   - Restore workflows (user needs to see deleted packages)
//   - Viewing historical payroll calculation details
//
// Returns:
//   - ErrCompensationPackageNotFound if package doesn't exist at all
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// For archive view showing all packages
//	pkg, err := service.GetCompensationPackageIncludingDeleted(ctx, pkgID)
//	if pkg.DeletedAt != nil {
//	    fmt.Println("This package is archived")
//	}
func (s *CompensationPackageService) GetCompensationPackageIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (*domain.EmployeeCompensationPackage, error) {
	pkg, err := s.repo.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrCompensationPackageNotFound
		}
		return nil, fmt.Errorf("failed to get compensation package (including deleted): %w", err)
	}
	return pkg, nil
}

// ListCompensationPackages retrieves all active (non-deleted) compensation packages.
//
// Packages are returned in the order determined by the repository.
//
// Returns an empty slice if no packages exist.
//
// Returns:
//   - Empty slice if no packages found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	pkgs, err := service.ListCompensationPackages(ctx)
//	for _, pkg := range pkgs {
//	    fmt.Printf("%s: %s\n", pkg.ID, pkg.BaseSalary)
//	}
func (s *CompensationPackageService) ListCompensationPackages(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.EmployeeCompensationPackage, error) {
	pkgs, err := s.repo.FindAll(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list compensation packages: %w", err)
	}
	return pkgs, nil
}

// ListCompensationPackagesIncludingDeleted retrieves all compensation packages,
// including soft-deleted ones.
//
// This is useful for:
//   - Archive views in the TUI
//   - Administrative interfaces
//   - Audit and reporting
//   - Understanding historical compensation structures
//
// Packages are returned in the order determined by the repository.
// Check pkg.DeletedAt to distinguish active vs deleted packages.
//
// Returns an empty slice if no packages exist.
//
// Returns:
//   - Empty slice if no packages found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	pkgs, err := service.ListCompensationPackagesIncludingDeleted(ctx)
//	for _, pkg := range pkgs {
//	    if pkg.DeletedAt != nil {
//	        fmt.Printf("%s (archived): %s\n", pkg.ID, pkg.BaseSalary)
//	    } else {
//	        fmt.Printf("%s: %s\n", pkg.ID, pkg.BaseSalary)
//	    }
//	}
func (s *CompensationPackageService) ListCompensationPackagesIncludingDeleted(
	ctx context.Context,
	orgID uuid.UUID,
) ([]*domain.EmployeeCompensationPackage, error) {
	pkgs, err := s.repo.FindAllIncludingDeleted(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list compensation packages (including deleted): %w", err)
	}
	return pkgs, nil
}

// ===============================================================================
// USAGE QUERY OPERATIONS
// ===============================================================================

// IsCompensationPackageInUse checks if a compensation package is currently
// referenced by any employees or payroll results.
//
// This is useful for:
//   - UI logic (disable delete button if in use)
//   - Pre-flight checks before attempting modifications
//   - Displaying usage information to users
//
// Returns true if package is in use, false otherwise.
//
// Note: This includes references from soft-deleted employees, as they can be restored.
//
// Example:
//
//	inUse, err := service.IsCompensationPackageInUse(ctx, pkgID)
//	if inUse {
//	    fmt.Println("Cannot modify - package is in use")
//	}
func (s *CompensationPackageService) IsCompensationPackageInUse(
	ctx context.Context,
	pkgID uuid.UUID,
) (bool, error) {
	// Count ALL employees (including soft-deleted)
	// Rationale: Soft-deleted employees can be restored, so package must remain intact
	empCount, err := s.repo.CountEmployeesUsing(ctx, pkgID)
	if err != nil {
		return false, fmt.Errorf("failed to count employees using package: %w", err)
	}

	// Count ALL payroll results
	// Rationale: Payroll results are immutable historical records
	resultCount, err := s.repo.CountPayrollResultsUsing(ctx, pkgID)
	if err != nil {
		return false, fmt.Errorf("failed to count payroll results using package: %w", err)
	}

	return empCount > 0 || resultCount > 0, nil
}

// GetCompensationPackageUsageCount returns detailed usage counts for a
// compensation package.
//
// This is useful for:
//   - Displaying detailed usage information to users
//   - Understanding the scope of reassignment needed
//   - Audit and reporting
//
// Returns:
//   - employeeCount: Number of employees using this package (including soft-deleted)
//   - payrollResultCount: Number of payroll results using this package
//   - error: Any error that occurred during counting
//
// Example:
//
//	empCount, resultCount, err := service.GetCompensationPackageUsageCount(ctx, pkgID)
//	if empCount > 0 {
//	    fmt.Printf("%d employees must be reassigned before deletion\n", empCount)
//	}
//	if resultCount > 0 {
//	    fmt.Printf("Referenced in %d historical payroll calculations\n", resultCount)
//	}
func (s *CompensationPackageService) GetCompensationPackageUsageCount(
	ctx context.Context,
	pkgID uuid.UUID,
) (employeeCount, payrollResultCount int64, err error) {
	employeeCount, err = s.repo.CountEmployeesUsing(ctx, pkgID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count employees using package: %w", err)
	}

	payrollResultCount, err = s.repo.CountPayrollResultsUsing(ctx, pkgID)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to count payroll results using package: %w", err)
	}

	return employeeCount, payrollResultCount, nil
}

// ===============================================================================
// COMPENSATION PACKAGE ERRORS
// ===============================================================================

var (
	// ErrCompensationPackageNotFound is returned when a compensation package
	// cannot be found by its ID.
	ErrCompensationPackageNotFound = errors.New("compensation package not found")

	// ErrCompensationPackageExists is returned when attempting to create
	// a compensation package that duplicates existing data.
	ErrCompensationPackageExists = errors.New("compensation package already exists")

	// ErrCompensationPackageInUse is returned when attempting to modify or delete
	// a compensation package that is currently in use by employees or referenced
	// in payroll results. This protects historical data integrity.
	ErrCompensationPackageInUse = errors.New("compensation package is in use and cannot be modified")
)
