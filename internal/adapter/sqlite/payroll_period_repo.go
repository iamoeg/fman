package sqlite_adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/fman/internal/adapter/sqlite/sqldb"
	"github.com/iamoeg/fman/internal/domain"
)

// ============================================================================
// Payroll Period Repository
// ============================================================================

// PayrollPeriodRepository implements payroll period data persistence using SQLite.
// It provides CRUD operations with soft delete support, audit logging, and
// specialized workflow methods for payroll period finalization/unfinalization.
type PayrollPeriodRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewPayrollPeriodRepository creates a new payroll period repository instance.
func NewPayrollPeriodRepository(db *sql.DB) *PayrollPeriodRepository {
	return &PayrollPeriodRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindByID retrieves an active (non-deleted) payroll period by ID.
// Returns ErrRecordNotFound if the period doesn't exist or is soft-deleted.
func (r *PayrollPeriodRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
	row, err := r.queries.GetPayrollPeriod(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll period", err)
	}

	pp, err := rowToPayrollPeriod(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	return pp, nil
}

// FindByIDIncludingDeleted retrieves a payroll period by ID, including soft-deleted records.
// Returns ErrRecordNotFound if the period doesn't exist.
func (r *PayrollPeriodRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollPeriod, error) {
	row, err := r.queries.GetPayrollPeriodIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll period (including deleted)", err)
	}

	pp, err := rowToPayrollPeriod(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	return pp, nil
}

// FindByOrgYearMonth retrieves an active payroll period by organization, year, and month.
// Returns ErrRecordNotFound if no matching period exists or if it is soft-deleted.
// This is useful for checking if a payroll period already exists before creating a new one.
func (r *PayrollPeriodRepository) FindByOrgYearMonth(ctx context.Context, orgID uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
	params := sqldb.GetPayrollPeriodByOrgYearMonthParams{
		OrgID: orgID.String(),
		Year:  int64(year),
		Month: int64(month),
	}
	row, err := r.queries.GetPayrollPeriodByOrgYearMonth(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll period by organization, year, month", err)
	}

	pp, err := rowToPayrollPeriod(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	return pp, nil
}

// FindByOrgYearMonthIncludingDeleted retrieves a payroll period by organization, year, and month,
// including soft-deleted records.
// Returns ErrRecordNotFound if no matching period exists.
func (r *PayrollPeriodRepository) FindByOrgYearMonthIncludingDeleted(ctx context.Context, orgID uuid.UUID, year, month int) (*domain.PayrollPeriod, error) {
	params := sqldb.GetPayrollPeriodByOrgYearMonthIncludingDeletedParams{
		OrgID: orgID.String(),
		Year:  int64(year),
		Month: int64(month),
	}
	row, err := r.queries.GetPayrollPeriodByOrgYearMonthIncludingDeleted(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll period by organization, year, month (including deleted)", err)
	}

	pp, err := rowToPayrollPeriod(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	return pp, nil
}

// FindByOrganization retrieves all active (non-deleted) payroll periods for an organization.
// Periods are ordered by year and month descending (most recent first).
// Returns an empty slice if no periods exist.
func (r *PayrollPeriodRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListPayrollPeriodsByOrganization(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll periods by organization", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// FindByOrganizationIncludingDeleted retrieves all payroll periods for an organization,
// including soft-deleted records.
// Periods are ordered by year and month descending (most recent first).
// Returns an empty slice if no periods exist.
func (r *PayrollPeriodRepository) FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListPayrollPeriodsByOrganizationIncludingDeleted(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll periods by organization (including deleted)", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// FindAllDraft retrieves all active (non-deleted) payroll periods with DRAFT status.
// This is useful for finding periods that are still being edited and haven't been finalized.
// Returns an empty slice if no draft periods exist.
func (r *PayrollPeriodRepository) FindAllDraft(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListDraftPayrollPeriods(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list draft payroll periods", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// FindAllDraftIncludingDeleted retrieves all payroll periods with DRAFT status,
// including soft-deleted records.
// Returns an empty slice if no draft periods exist.
func (r *PayrollPeriodRepository) FindAllDraftIncludingDeleted(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListDraftPayrollPeriodsIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list draft payroll periods (including deleted)", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// FindAll retrieves all active (non-deleted) payroll periods across all organizations.
// Periods are ordered by organization ID, year, and month.
// Returns an empty slice if no periods exist.
func (r *PayrollPeriodRepository) FindAll(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListPayrollPeriods(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll periods", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// FindAllIncludingDeleted retrieves all payroll periods across all organizations,
// including soft-deleted records.
// Periods are ordered by organization ID, year, and month.
// Returns an empty slice if no periods exist.
func (r *PayrollPeriodRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollPeriod, error) {
	rows, err := r.queries.ListPayrollPeriodsIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll periods (including deleted)", err)
	}

	pps := make([]*domain.PayrollPeriod, 0, len(rows))

	for _, row := range rows {
		pp, err := rowToPayrollPeriod(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll period", err)
		}
		pps = append(pps, pp)
	}

	return pps, nil
}

// ============================================================================
// Mutation Methods
// ============================================================================

// Create persists a new payroll period and creates an audit log entry.
// Returns ErrDuplicateRecord in case of UNIQUE constraint violations.
// The operation is atomic - both the period and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) Create(ctx context.Context, period *domain.PayrollPeriod) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	params := payrollPeriodToCreateParams(period)
	row, err := qtx.CreatePayrollPeriod(ctx, params)
	if err != nil {
		if isUniqueConstraintViolation(err) {
			return ErrDuplicateRecord
		}
		return fmt.Errorf(FmtDBQueryErr, "create payroll period", err)
	}

	periodCreated, err := rowToPayrollPeriod(row)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		periodCreated.ID.String(),
		DBActionCreate,
		nil,
		periodCreated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Finalize transitions a payroll period from DRAFT to FINALIZED status.
// This is a specialized workflow method that:
// - Only works on periods with DRAFT status (enforced by SQL query WHERE clause)
// - Sets the finalized_at timestamp to current time
// - Creates an audit log entry with before/after snapshots
//
// Once finalized, a period should be treated as immutable (though Unfinalize exists for error correction).
// Returns ErrRecordNotFound if the period doesn't exist or is already finalized.
// The operation is atomic - both the status change and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) Finalize(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	periodRow, err := qtx.GetPayrollPeriod(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll period", err)
	}

	period, err := rowToPayrollPeriod(periodRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	params := payrollPeriodToFinalizeParams(period)
	periodFinalizedRow, err := qtx.FinalizePayrollPeriod(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "finalize payroll period", err)
	}

	periodFinalized, err := rowToPayrollPeriod(periodFinalizedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		period.ID.String(),
		DBActionUpdate,
		period,
		periodFinalized,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Unfinalize transitions a payroll period from FINALIZED back to DRAFT status.
// This is a specialized workflow method for error correction that:
// - Only works on periods with FINALIZED status (enforced by SQL query WHERE clause)
// - Clears the finalized_at timestamp
// - Creates an audit log entry with before/after snapshots
//
// This should only be used for error correction. Under normal workflow, periods
// should remain finalized once they reach that state.
// Returns ErrRecordNotFound if the period doesn't exist or is not finalized.
// The operation is atomic - both the status change and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) Unfinalize(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	periodRow, err := qtx.GetPayrollPeriod(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll period", err)
	}

	period, err := rowToPayrollPeriod(periodRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	params := payrollPeriodToUnfinalizeParams(period)
	periodUnfinalizedRow, err := qtx.UnfinalizePayrollPeriod(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "unfinalize payroll period", err)
	}

	periodUnfinalized, err := rowToPayrollPeriod(periodUnfinalizedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		period.ID.String(),
		DBActionUpdate,
		period,
		periodUnfinalized,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Delete soft-deletes a payroll period by setting deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the period doesn't exist or is already soft-deleted.
//
// Note: Soft-deleting a payroll period will CASCADE delete all associated payroll_results
// due to the foreign key relationship (payroll_result.payroll_period_id → payroll_period.id).
// The operation is atomic - both the soft delete and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	periodRow, err := qtx.GetPayrollPeriod(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll period", err)
	}

	period, err := rowToPayrollPeriod(periodRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	params := payrollPeriodToDeleteParams(period)
	periodDeletedRow, err := qtx.DeletePayrollPeriod(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "delete payroll period", err)
	}

	periodDeleted, err := rowToPayrollPeriod(periodDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		period.ID.String(),
		DBActionDelete,
		period,
		periodDeleted,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Restore un-deletes a soft-deleted payroll period by clearing deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the period doesn't exist.
//
// Note: Restoring a payroll period will NOT automatically restore any associated
// payroll_results that were cascade-deleted. Those must be restored separately if needed.
// The operation is atomic - both the restoration and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	periodRow, err := qtx.GetPayrollPeriodIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll period", err)
	}

	period, err := rowToPayrollPeriod(periodRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	params := payrollPeriodToRestoreParams(period)
	periodRestoredRow, err := qtx.RestorePayrollPeriod(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "restore payroll period", err)
	}

	periodRestored, err := rowToPayrollPeriod(periodRestoredRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		period.ID.String(),
		DBActionRestore,
		period,
		periodRestored,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// HardDelete permanently removes a payroll period from the database.
// Creates an audit log entry before deletion (audit log survives the deletion).
// Returns ErrRecordNotFound if the period doesn't exist.
//
// WARNING: This operation is irreversible. Use Delete() for soft deletion instead.
// Hard deletion will CASCADE delete all associated payroll_results permanently.
// The operation is atomic - both the deletion and audit log are created in a single transaction.
func (r *PayrollPeriodRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	periodRow, err := qtx.GetPayrollPeriodIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll period (including deleted)", err)
	}

	period, err := rowToPayrollPeriod(periodRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll period", err)
	}

	if delErr := qtx.HardDeletePayrollPeriod(ctx, period.ID.String()); delErr != nil {
		return fmt.Errorf(FmtDBQueryErr, "delete payroll period", delErr)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollPeriodTableName,
		period.ID.String(),
		DBActionHardDelete,
		period,
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
// Transaction Support
// ============================================================================

// WithTx returns a new repository instance that uses the provided transaction.
// This allows repository methods to participate in transactions managed by the caller.
func (r *PayrollPeriodRepository) WithTx(tx *sql.Tx) *PayrollPeriodRepository {
	return &PayrollPeriodRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToPayrollPeriod converts a sqlc-generated PayrollPeriod row to a domain.PayrollPeriod.
// Returns an error if any field fails to parse.
func rowToPayrollPeriod(row sqldb.PayrollPeriod) (*domain.PayrollPeriod, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "id as uuid", err)
	}

	orgID, err := uuid.Parse(row.OrgID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "org_id as uuid", err)
	}

	var finalizedAt *time.Time
	if row.FinalizedAt.Valid {
		t, parseErr := time.Parse(DBTimeFormat, row.FinalizedAt.String)
		if parseErr != nil {
			return nil, fmt.Errorf(FmtColumnParsingErr, "finalized_at as time", parseErr)
		}
		finalizedAt = &t
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

	return &domain.PayrollPeriod{
		ID:          id,
		OrgID:       orgID,
		Year:        int(row.Year),
		Month:       int(row.Month),
		Status:      domain.PayrollPeriodStatusEnum(row.Status),
		FinalizedAt: finalizedAt,
		CreatedAt:   createdAt,
		UpdatedAt:   updatedAt,
		DeletedAt:   deletedAt,
	}, nil
}

// payrollPeriodToCreateParams converts a domain.PayrollPeriod to sqlc CreatePayrollPeriodParams.
func payrollPeriodToCreateParams(pp *domain.PayrollPeriod) sqldb.CreatePayrollPeriodParams {
	var finalizedAtStr string
	if pp.FinalizedAt != nil {
		finalizedAtStr = pp.FinalizedAt.Format(DBTimeFormat)
	}

	return sqldb.CreatePayrollPeriodParams{
		ID:          pp.ID.String(),
		OrgID:       pp.OrgID.String(),
		Year:        int64(pp.Year),
		Month:       int64(pp.Month),
		Status:      string(pp.Status),
		FinalizedAt: stringToNullString(finalizedAtStr),
		CreatedAt:   pp.CreatedAt.Format(DBTimeFormat),
		UpdatedAt:   pp.UpdatedAt.Format(DBTimeFormat),
	}
}

// payrollPeriodToDeleteParams converts a domain.PayrollPeriod to sqlc DeletePayrollPeriodParams.
// Sets both deleted_at and updated_at to current time.
func payrollPeriodToDeleteParams(pp *domain.PayrollPeriod) sqldb.DeletePayrollPeriodParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.DeletePayrollPeriodParams{
		ID:        pp.ID.String(),
		UpdatedAt: now,
		DeletedAt: stringToNullString(now),
	}
}

// payrollPeriodToRestoreParams converts a domain.PayrollPeriod to sqlc RestorePayrollPeriodParams.
// Sets updated_at to current time.
func payrollPeriodToRestoreParams(pp *domain.PayrollPeriod) sqldb.RestorePayrollPeriodParams {
	return sqldb.RestorePayrollPeriodParams{
		ID:        pp.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// payrollPeriodToFinalizeParams converts a domain.PayrollPeriod to sqlc FinalizePayrollPeriodParams.
// Sets both finalized_at and updated_at to current time.
func payrollPeriodToFinalizeParams(pp *domain.PayrollPeriod) sqldb.FinalizePayrollPeriodParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.FinalizePayrollPeriodParams{
		ID:          pp.ID.String(),
		FinalizedAt: stringToNullString(now),
		UpdatedAt:   now,
	}
}

// payrollPeriodToUnfinalizeParams converts a domain.PayrollPeriod to sqlc UnfinalizePayrollPeriodParams.
// Sets updated_at to current time.
func payrollPeriodToUnfinalizeParams(period *domain.PayrollPeriod) sqldb.UnfinalizePayrollPeriodParams {
	return sqldb.UnfinalizePayrollPeriodParams{
		ID:        period.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// ============================================================================
// Constants
// ============================================================================

const (
	// PayrollPeriodTableName is the payroll period table name used for audit logging.
	PayrollPeriodTableName = "payroll_period"
)
