package sqlite_adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite/sqldb"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
	"github.com/iamoeg/bootdev-capstone/pkg/money"
)

// ============================================================================
// Employee Compensation Package Repository
// ============================================================================

// EmployeeCompensationPackageRepository implements compensation package data persistence using SQLite.
// It provides CRUD operations with soft delete support, audit logging, and usage guards to protect
// historical artifacts from modification when referenced by employees or payroll results.
//
// Key Design Decisions:
//   - Compensation packages are historical artifacts - once referenced by payroll, they're permanent
//   - Update/Delete/HardDelete are guarded - they fail if the package is in use
//   - All mutations are wrapped in transactions with audit logging
type EmployeeCompensationPackageRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewCompensationPackageRepository creates a new compensation package repository instance.
func NewCompensationPackageRepository(db *sql.DB) *EmployeeCompensationPackageRepository {
	return &EmployeeCompensationPackageRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindByID retrieves an active (non-deleted) compensation package by ID.
// Returns ErrRecordNotFound if the package doesn't exist or is soft-deleted.
func (r *EmployeeCompensationPackageRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
	row, err := r.queries.GetEmployeeCompensationPackage(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get employee compensation package", err)
	}

	pkg, err := rowToCompensationPackage(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	return pkg, nil
}

// FindByIDIncludingDeleted retrieves a compensation package by ID, including soft-deleted records.
// Returns ErrRecordNotFound if the package doesn't exist.
func (r *EmployeeCompensationPackageRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.EmployeeCompensationPackage, error) {
	row, err := r.queries.GetEmployeeCompensationPackageIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get employee compensation package (including deleted)", err)
	}

	pkg, err := rowToCompensationPackage(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	return pkg, nil
}

// FindAll retrieves all active (non-deleted) compensation packages.
// Returns an empty slice if no packages exist.
func (r *EmployeeCompensationPackageRepository) FindAll(ctx context.Context) ([]*domain.EmployeeCompensationPackage, error) {
	rows, err := r.queries.ListEmployeeCompensationPackages(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employee compensation packages", err)
	}

	pkgs := make([]*domain.EmployeeCompensationPackage, 0, len(rows))
	for _, row := range rows {
		pkg, err := rowToCompensationPackage(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
		}
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// FindAllIncludingDeleted retrieves all compensation packages, including soft-deleted records.
// Returns an empty slice if no packages exist.
func (r *EmployeeCompensationPackageRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.EmployeeCompensationPackage, error) {
	rows, err := r.queries.ListEmployeeCompensationPackagesIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employee compensation packages (including deleted)", err)
	}

	pkgs := make([]*domain.EmployeeCompensationPackage, 0, len(rows))
	for _, row := range rows {
		pkg, err := rowToCompensationPackage(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
		}
		pkgs = append(pkgs, pkg)
	}

	return pkgs, nil
}

// CountEmployeesUsing returns the number of active employees currently using this compensation package.
// This count is used to guard Update/Delete operations - packages in use cannot be modified or deleted.
func (r *EmployeeCompensationPackageRepository) CountEmployeesUsing(ctx context.Context, id uuid.UUID) (int64, error) {
	c, err := r.queries.CountEmployeesUsingCompensationPackage(ctx, id.String())
	if err != nil {
		return 0, fmt.Errorf(FmtDBQueryErr, "count employees using compensation package", err)
	}

	return c, nil
}

// CountPayrollResultsUsing returns the number of payroll results that reference this compensation package.
// This count is used to guard Update/Delete operations - packages referenced by payroll are permanent
// historical artifacts and cannot be modified or deleted.
func (r *EmployeeCompensationPackageRepository) CountPayrollResultsUsing(ctx context.Context, id uuid.UUID) (int64, error) {
	c, err := r.queries.CountPayrollResultsUsingCompensationPackage(ctx, id.String())
	if err != nil {
		return 0, fmt.Errorf(FmtDBQueryErr, "count payroll results using compensation package", err)
	}

	return c, nil
}

// ============================================================================
// Mutation Methods
// ============================================================================

// Create persists a new compensation package and creates an audit log entry.
// Returns ErrDuplicateRecord in case of UNIQUE constraint violations.
// The operation is atomic - both the package and audit log are created in a single transaction.
func (r *EmployeeCompensationPackageRepository) Create(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	params := compensationPackageToCreateParams(pkg)
	row, err := qtx.CreateEmployeeCompensationPackage(ctx, params)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return ErrDuplicateRecord
		}
		return fmt.Errorf(FmtDBQueryErr, "create employee compensation package", err)
	}

	pkgCreated, err := rowToCompensationPackage(row)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		CompensationPackageTableName,
		pkgCreated.ID.String(),
		DBActionCreate,
		nil,
		pkgCreated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Update modifies an existing compensation package and creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the package doesn't exist or is soft-deleted.
// Returns ErrDuplicateRecord in case of UNIQUE constraint violations.
// Returns ErrCompensationPackageInUse if the package is referenced by any employees or payroll results.
// The operation is atomic - both the update and audit log are created in a single transaction.
//
// IMPORTANT: Compensation packages referenced by payroll results are historical artifacts and cannot be modified.
// This protects the integrity of the audit trail and ensures payroll calculations remain accurate.
func (r *EmployeeCompensationPackageRepository) Update(ctx context.Context, pkg *domain.EmployeeCompensationPackage) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	pkgOldRow, err := qtx.GetEmployeeCompensationPackage(ctx, pkg.ID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee compensation package", err)
	}

	pkgOld, err := rowToCompensationPackage(pkgOldRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	// Guard: Prevent modification of packages in use
	if err := r.checkNotInUse(ctx, qtx, pkgOld.ID); err != nil {
		return err
	}

	params := compensationPackageToUpdateParams(pkg)
	pkgUpdatedRow, err := qtx.UpdateEmployeeCompensationPackage(ctx, params)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return ErrDuplicateRecord
		}
		return fmt.Errorf(FmtDBQueryErr, "update employee compensation package", err)
	}

	pkgUpdated, err := rowToCompensationPackage(pkgUpdatedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		CompensationPackageTableName,
		pkg.ID.String(),
		DBActionUpdate,
		pkgOld,
		pkgUpdated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Delete soft-deletes a compensation package by setting deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the package doesn't exist or is already soft-deleted.
// Returns ErrCompensationPackageInUse if the package is referenced by any employees or payroll results.
// The operation is atomic - both the soft delete and audit log are created in a single transaction.
//
// IMPORTANT: Compensation packages referenced by payroll results are historical artifacts and cannot be deleted.
// This protects the integrity of the audit trail and ensures payroll calculations remain accurate.
func (r *EmployeeCompensationPackageRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	pkgRow, err := qtx.GetEmployeeCompensationPackage(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee compensation package", err)
	}

	pkg, err := rowToCompensationPackage(pkgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	// Guard: Prevent deletion of packages in use
	if err := r.checkNotInUse(ctx, qtx, pkg.ID); err != nil {
		return err
	}

	params := compensationPackageToDeleteParams(pkg)
	pkgDeletedRow, err := qtx.DeleteEmployeeCompensationPackage(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "delete employee compensation package", err)
	}

	pkgDeleted, err := rowToCompensationPackage(pkgDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		CompensationPackageTableName,
		pkg.ID.String(),
		DBActionDelete,
		pkg,
		pkgDeleted,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Restore un-deletes a soft-deleted compensation package by clearing deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the package doesn't exist.
// The operation is atomic - both the restoration and audit log are created in a single transaction.
func (r *EmployeeCompensationPackageRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	pkgRow, err := qtx.GetEmployeeCompensationPackageIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee compensation package (including deleted)", err)
	}

	pkg, err := rowToCompensationPackage(pkgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	params := compensationPackageToRestoreParams(pkg)
	pkgRestoredRow, err := qtx.RestoreEmployeeCompensationPackage(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "restore employee compensation package", err)
	}

	pkgRestored, err := rowToCompensationPackage(pkgRestoredRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		CompensationPackageTableName,
		pkg.ID.String(),
		DBActionRestore,
		pkg,
		pkgRestored,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// HardDelete permanently removes a compensation package from the database.
// Creates an audit log entry before deletion (audit log survives the deletion).
// Returns ErrRecordNotFound if the package doesn't exist.
// Returns ErrCompensationPackageInUse if the package is referenced by any employees or payroll results.
// WARNING: This operation is irreversible. Use Delete() for soft deletion instead.
// The operation is atomic - both the deletion and audit log are created in a single transaction.
//
// IMPORTANT: Even soft-deleted packages cannot be hard-deleted if they're referenced by payroll results.
// This ensures the audit trail and payroll calculation integrity are never compromised.
func (r *EmployeeCompensationPackageRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	pkgRow, err := qtx.GetEmployeeCompensationPackageIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee compensation package (including deleted)", err)
	}

	pkg, err := rowToCompensationPackage(pkgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee compensation package", err)
	}

	// Guard: Prevent hard deletion of packages in use (even if soft-deleted)
	if err := r.checkNotInUse(ctx, qtx, pkg.ID); err != nil {
		return err
	}

	if err := qtx.HardDeleteEmployeeCompensationPackage(ctx, pkg.ID.String()); err != nil {
		return fmt.Errorf(FmtDBQueryErr, "hard delete employee compensation package", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		CompensationPackageTableName,
		pkg.ID.String(),
		DBActionHardDelete,
		pkg,
		nil,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// ============================================================================
// Usage Guards
// ============================================================================

// checkNotInUse verifies that a compensation package is not referenced by any entities.
// Returns ErrCompensationPackageInUse if the package is referenced by:
//   - Any active employees (via compensation_package_id)
//   - Any payroll results (historical artifact protection)
//
// This guard prevents modification or deletion of packages that are in use, which would:
//   - Break foreign key constraints
//   - Compromise payroll calculation accuracy
//   - Violate audit trail integrity
func (r *EmployeeCompensationPackageRepository) checkNotInUse(
	ctx context.Context,
	qtx *sqldb.Queries,
	id uuid.UUID,
) error {
	empCount, err := qtx.CountEmployeesUsingCompensationPackage(ctx, id.String())
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "count employees using compensation package", err)
	}
	if empCount > 0 {
		return ErrCompensationPackageInUse
	}

	resultCount, err := qtx.CountPayrollResultsUsingCompensationPackage(ctx, id.String())
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "count payroll results using compensation package", err)
	}
	if resultCount > 0 {
		return ErrCompensationPackageInUse
	}

	return nil
}

// ============================================================================
// Transaction Support
// ============================================================================

// WithTx returns a new repository instance that uses the provided transaction.
// This allows repository methods to participate in transactions managed by the caller
// (typically the application service layer).
func (r *EmployeeCompensationPackageRepository) WithTx(tx *sql.Tx) *EmployeeCompensationPackageRepository {
	return &EmployeeCompensationPackageRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToCompensationPackage converts a sqlc-generated EmployeeCompensationPackage row to a domain.EmployeeCompensationPackage.
// Returns an error if any field fails to parse (UUID, timestamps, money).
func rowToCompensationPackage(row sqldb.EmployeeCompensationPackage) (*domain.EmployeeCompensationPackage, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "id as uuid", err)
	}

	createdAt, err := time.Parse(DBTimeFormat, row.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "created_at as time", err)
	}

	updatedAt, err := time.Parse(DBTimeFormat, row.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "updated_at as time", err)
	}

	var deletedAt *time.Time
	if row.DeletedAt.Valid {
		t, err := time.Parse(DBTimeFormat, row.DeletedAt.String)
		if err != nil {
			return nil, fmt.Errorf(FmtColumnParsingErr, "deleted_at as time", err)
		}
		deletedAt = &t
	}

	return &domain.EmployeeCompensationPackage{
		ID:         id,
		Currency:   money.Currency(row.Currency),
		BaseSalary: money.FromCents(row.BaseSalaryCents),
		CreatedAt:  createdAt,
		UpdatedAt:  updatedAt,
		DeletedAt:  deletedAt,
	}, nil
}

// compensationPackageToCreateParams converts a domain.EmployeeCompensationPackage to sqlc CreateEmployeeCompensationPackageParams.
// Timestamps are taken from the domain object (set by application service).
func compensationPackageToCreateParams(pkg *domain.EmployeeCompensationPackage) sqldb.CreateEmployeeCompensationPackageParams {
	return sqldb.CreateEmployeeCompensationPackageParams{
		ID:              pkg.ID.String(),
		Currency:        pkg.Currency.String(),
		BaseSalaryCents: pkg.BaseSalary.Cents(),
		CreatedAt:       pkg.CreatedAt.Format(DBTimeFormat),
		UpdatedAt:       pkg.UpdatedAt.Format(DBTimeFormat),
	}
}

// compensationPackageToUpdateParams converts a domain.EmployeeCompensationPackage to sqlc UpdateEmployeeCompensationPackageParams.
// Note: UpdatedAt is set to current time, not taken from the domain object.
func compensationPackageToUpdateParams(pkg *domain.EmployeeCompensationPackage) sqldb.UpdateEmployeeCompensationPackageParams {
	return sqldb.UpdateEmployeeCompensationPackageParams{
		ID:              pkg.ID.String(),
		Currency:        pkg.Currency.String(),
		BaseSalaryCents: pkg.BaseSalary.Cents(),
		UpdatedAt:       time.Now().Format(DBTimeFormat),
	}
}

// compensationPackageToDeleteParams converts a domain.EmployeeCompensationPackage to sqlc DeleteEmployeeCompensationPackageParams.
// Sets both deleted_at and updated_at to current time.
func compensationPackageToDeleteParams(pkg *domain.EmployeeCompensationPackage) sqldb.DeleteEmployeeCompensationPackageParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.DeleteEmployeeCompensationPackageParams{
		ID:        pkg.ID.String(),
		UpdatedAt: now,
		DeletedAt: stringToNullString(now),
	}
}

// compensationPackageToRestoreParams converts a domain.EmployeeCompensationPackage to sqlc RestoreEmployeeCompensationPackageParams.
// Sets updated_at to current time.
func compensationPackageToRestoreParams(pkg *domain.EmployeeCompensationPackage) sqldb.RestoreEmployeeCompensationPackageParams {
	return sqldb.RestoreEmployeeCompensationPackageParams{
		ID:        pkg.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// ============================================================================
// Constants
// ============================================================================

const (
	// Table name constant for audit logging.
	CompensationPackageTableName = "employee_compensation_package"
)

// ============================================================================
// Errors
// ============================================================================

// Sentinel errors for programmatic error handling.
var (
	// ErrCompensationPackageInUse is returned when attempting to update or delete a compensation package
	// that is currently referenced by employees or payroll results. This protects historical artifacts
	// from modification, ensuring audit trail integrity and payroll calculation accuracy.
	ErrCompensationPackageInUse = errors.New("sqlite: employee compensation package is in use (referenced by entities)")
)
