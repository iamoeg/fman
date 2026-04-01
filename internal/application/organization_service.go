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

// organizationRepository defines the minimal interface that OrganizationService
// needs from its persistence layer.
//
// The repository is responsible for:
//   - Persisting organization data
//   - Handling database transactions
//   - Creating audit logs
//   - Managing soft deletes
type organizationRepository interface {
	// Create persists a new organization to the database.
	// Returns ErrDuplicateRecord if business identifiers already exist.
	Create(ctx context.Context, org *domain.Organization) error

	// Update modifies an existing organization in the database.
	// Returns ErrRecordNotFound if organization doesn't exist or is soft-deleted.
	// Returns ErrDuplicateRecord if updated business identifiers conflict.
	Update(ctx context.Context, org *domain.Organization) error

	// Delete soft-deletes an organization by setting deleted_at timestamp.
	// Returns ErrRecordNotFound if organization doesn't exist or is already deleted.
	Delete(ctx context.Context, id uuid.UUID) error

	// Restore un-deletes a soft-deleted organization by clearing deleted_at.
	// Returns ErrRecordNotFound if organization doesn't exist or is not deleted.
	Restore(ctx context.Context, id uuid.UUID) error

	// HardDelete permanently deletes an organization from the database.
	// This is irreversible and should only be used for data purging (GDPR, test cleanup).
	// Returns ErrRecordNotFound if organization doesn't exist.
	HardDelete(ctx context.Context, id uuid.UUID) error

	// FindByID retrieves an organization by its ID.
	// Only returns active (non-deleted) organizations.
	// Returns ErrRecordNotFound if not found or soft-deleted.
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error)

	// FindByIDIncludingDeleted retrieves an organization by its ID,
	// including soft-deleted organizations.
	// Returns ErrRecordNotFound if not found.
	FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Organization, error)

	// FindAll retrieves all active (non-deleted) organizations.
	// Returns empty slice if none found.
	FindAll(ctx context.Context) ([]*domain.Organization, error)

	// FindAllIncludingDeleted retrieves all organizations,
	// including soft-deleted ones.
	// Returns empty slice if none found.
	FindAllIncludingDeleted(ctx context.Context) ([]*domain.Organization, error)
}

// ===============================================================================
// SERVICE IMPLEMENTATION
// ===============================================================================

// OrganizationService provides business logic for managing organizations.
// It orchestrates domain validation, UUID generation, timestamp management,
// and persistence operations.
//
// Responsibilities:
//   - Generate UUIDs and timestamps
//   - Validate domain business rules
//   - Coordinate with repository layer
//   - Translate repository errors to service-level errors
//
// This service is intentionally simple since organizations are the root entity
// in the system with minimal business logic beyond validation.
type OrganizationService struct {
	repo organizationRepository
}

// NewOrganizationService creates a new OrganizationService with the given repository.
//
// The repository parameter should be an implementation of organizationRepository,
// typically *sqlite.OrganizationRepository in production or a mock in tests.
func NewOrganizationService(repo organizationRepository) *OrganizationService {
	return &OrganizationService{
		repo: repo,
	}
}

// ===============================================================================
// CREATE OPERATIONS
// ===============================================================================

// CreateOrganization creates a new organization in the system.
//
// The service will:
//  1. Generate a UUID if org.ID is uuid.Nil
//  2. Set CreatedAt, UpdatedAt timestamps to current UTC time
//  3. Ensure DeletedAt is nil (not soft-deleted)
//  4. Validate all domain business rules
//  5. Persist to database
//
// The org parameter is modified in-place, allowing the caller to access
// the generated ID and timestamps after creation.
//
// Returns:
//   - ErrOrganizationExists if business identifiers (ICE, IF, RC, etc.) are duplicates
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	org := &domain.Organization{
//	    Name: "ACME Corp",
//	    LegalForm: domain.LegalFormSARL,
//	    ICENum: "123456789012345",
//	}
//	err := service.CreateOrganization(ctx, org)
//	fmt.Println(org.ID) // UUID was generated
func (s *OrganizationService) CreateOrganization(
	ctx context.Context,
	org *domain.Organization,
) error {
	// 1. Generate UUID if not provided
	if org.ID == uuid.Nil {
		org.ID = uuid.New()
	}

	// 2. Set timestamps
	now := time.Now().UTC()
	org.CreatedAt = now
	org.UpdatedAt = now
	org.DeletedAt = nil

	// 3. Validate domain rules
	if err := org.Validate(); err != nil {
		return fmt.Errorf("invalid organization: %w", err)
	}

	// 4. Persist
	if err := s.repo.Create(ctx, org); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrDuplicateRecord) {
			return ErrOrganizationExists
		}
		return fmt.Errorf("failed to create organization: %w", err)
	}

	return nil
}

// ===============================================================================
// UPDATE OPERATIONS
// ===============================================================================

// UpdateOrganization updates an existing organization in the system.
//
// The service will:
//  1. Update the UpdatedAt timestamp to current UTC time
//  2. Validate all domain business rules
//  3. Persist changes to database
//
// The org.ID must be set to the organization being updated.
// The org.CreatedAt and org.DeletedAt fields are not modified.
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist or is soft-deleted
//   - ErrOrganizationExists if updated business identifiers conflict with another org
//   - Domain validation errors if business rules are violated
//   - Wrapped repository errors for other failures
//
// Example:
//
//	org, _ := service.GetOrganization(ctx, orgID)
//	org.Name = "Updated Name"
//	err := service.UpdateOrganization(ctx, org)
func (s *OrganizationService) UpdateOrganization(
	ctx context.Context,
	org *domain.Organization,
) error {
	// 1. Update timestamp
	org.UpdatedAt = time.Now().UTC()

	// 2. Validate domain rules
	if err := org.Validate(); err != nil {
		return fmt.Errorf("invalid organization: %w", err)
	}

	// 3. Persist
	if err := s.repo.Update(ctx, org); err != nil {
		// Translate repository errors to service errors
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		if errors.Is(err, sqlite.ErrDuplicateRecord) {
			return ErrOrganizationExists
		}
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// ===============================================================================
// DELETE OPERATIONS
// ===============================================================================

// DeleteOrganization soft-deletes an organization by setting its deleted_at timestamp.
//
// Soft delete allows the organization to be restored later if needed and maintains
// referential integrity with related records (employees, payroll periods).
//
// Note: Due to CASCADE foreign keys, deleting an organization will also soft-delete:
//   - All employees in the organization
//   - All payroll periods for the organization
//   - All payroll results in those periods
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist or is already deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.DeleteOrganization(ctx, orgID)
func (s *OrganizationService) DeleteOrganization(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("failed to delete organization: %w", err)
	}
	return nil
}

// RestoreOrganization restores a soft-deleted organization by clearing its deleted_at timestamp.
//
// This operation reverses a soft delete, making the organization active again.
//
// Note: This does NOT automatically restore related records (employees, payroll periods).
// Those must be restored separately if needed.
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist or is not deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	err := service.RestoreOrganization(ctx, orgID)
func (s *OrganizationService) RestoreOrganization(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.Restore(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("failed to restore organization: %w", err)
	}
	return nil
}

// HardDeleteOrganization permanently deletes an organization from the database.
//
// WARNING: This is irreversible. The organization and all audit logs are permanently removed.
//
// Use cases:
//   - GDPR compliance (right to be forgotten)
//   - Test data cleanup
//   - Data purging of very old soft-deleted records
//
// Note: Due to CASCADE foreign keys, hard deleting an organization will also permanently delete:
//   - All employees in the organization
//   - All payroll periods for the organization
//   - All payroll results in those periods
//   - All audit logs for these records
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// Only use for legitimate data purging
//	err := service.HardDeleteOrganization(ctx, orgID)
func (s *OrganizationService) HardDeleteOrganization(
	ctx context.Context,
	id uuid.UUID,
) error {
	if err := s.repo.HardDelete(ctx, id); err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return ErrOrganizationNotFound
		}
		return fmt.Errorf("failed to hard delete organization: %w", err)
	}
	return nil
}

// ===============================================================================
// QUERY OPERATIONS
// ===============================================================================

// GetOrganization retrieves an organization by its ID.
//
// Only returns active (non-deleted) organizations.
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist or is soft-deleted
//   - Wrapped repository errors for other failures
//
// Example:
//
//	org, err := service.GetOrganization(ctx, orgID)
//	if errors.Is(err, ErrOrganizationNotFound) {
//	    // Handle not found
//	}
func (s *OrganizationService) GetOrganization(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Organization, error) {
	org, err := s.repo.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}
	return org, nil
}

// GetOrganizationIncludingDeleted retrieves an organization by its ID,
// including soft-deleted organizations.
//
// This is useful for:
//   - Archive views in the TUI
//   - Audit trail examination
//   - Restore workflows (user needs to see deleted orgs)
//
// Returns:
//   - ErrOrganizationNotFound if organization doesn't exist at all
//   - Wrapped repository errors for other failures
//
// Example:
//
//	// For archive view showing all organizations
//	org, err := service.GetOrganizationIncludingDeleted(ctx, orgID)
//	if org.DeletedAt != nil {
//	    fmt.Println("This organization is archived")
//	}
func (s *OrganizationService) GetOrganizationIncludingDeleted(
	ctx context.Context,
	id uuid.UUID,
) (*domain.Organization, error) {
	org, err := s.repo.FindByIDIncludingDeleted(ctx, id)
	if err != nil {
		if errors.Is(err, sqlite.ErrRecordNotFound) {
			return nil, ErrOrganizationNotFound
		}
		return nil, fmt.Errorf("failed to get organization (including deleted): %w", err)
	}
	return org, nil
}

// ListOrganizations retrieves all active (non-deleted) organizations.
//
// Organizations are returned in the order determined by the repository
// (typically alphabetical by name).
//
// Returns an empty slice if no organizations exist.
//
// Returns:
//   - Empty slice if no organizations found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	orgs, err := service.ListOrganizations(ctx)
//	for _, org := range orgs {
//	    fmt.Println(org.Name)
//	}
func (s *OrganizationService) ListOrganizations(
	ctx context.Context,
) ([]*domain.Organization, error) {
	orgs, err := s.repo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations: %w", err)
	}
	return orgs, nil
}

// ListOrganizationsIncludingDeleted retrieves all organizations,
// including soft-deleted ones.
//
// This is useful for:
//   - Archive views in the TUI
//   - Administrative interfaces
//   - Audit and reporting
//
// Organizations are returned in the order determined by the repository.
// Check org.DeletedAt to distinguish active vs deleted organizations.
//
// Returns an empty slice if no organizations exist.
//
// Returns:
//   - Empty slice if no organizations found (not an error)
//   - Wrapped repository errors for failures
//
// Example:
//
//	orgs, err := service.ListOrganizationsIncludingDeleted(ctx)
//	for _, org := range orgs {
//	    if org.DeletedAt != nil {
//	        fmt.Printf("%s (archived)\n", org.Name)
//	    } else {
//	        fmt.Println(org.Name)
//	    }
//	}
func (s *OrganizationService) ListOrganizationsIncludingDeleted(
	ctx context.Context,
) ([]*domain.Organization, error) {
	orgs, err := s.repo.FindAllIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list organizations (including deleted): %w", err)
	}
	return orgs, nil
}

// ===============================================================================
// ORGANIZATION ERRORS
// ===============================================================================

var (
	// ErrOrganizationNotFound is returned when an organization cannot be found
	// by its ID or other identifying criteria.
	ErrOrganizationNotFound = errors.New("organization not found")

	// ErrOrganizationExists is returned when attempting to create or update
	// an organization with business identifiers (ICE, IF, RC, CNSS, RIB) that
	// already exist in the system.
	ErrOrganizationExists = errors.New("organization already exists")
)
