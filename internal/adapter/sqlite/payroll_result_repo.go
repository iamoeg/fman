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
// Payroll Result Repository
// ============================================================================

// PayrollResultRepository implements payroll result data persistence using SQLite.
// It provides CRUD operations with soft delete support and audit logging.
//
// Note: Payroll results are immutable historical records once created.
// There is no Update method - if changes are needed, delete and recreate the entire period.
type PayrollResultRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewPayrollResultRepository creates a new payroll result repository instance.
func NewPayrollResultRepository(db *sql.DB) *PayrollResultRepository {
	return &PayrollResultRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindByID retrieves an active (non-deleted) payroll result by ID.
// Returns ErrRecordNotFound if the payroll result doesn't exist or is soft-deleted.
func (r *PayrollResultRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
	row, err := r.queries.GetPayrollResult(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll result", err)
	}

	res, err := rowToPayrollResult(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	return res, nil
}

// FindByIDIncludingDeleted retrieves a payroll result by ID, including soft-deleted records.
// Returns ErrRecordNotFound if the payroll result doesn't exist.
func (r *PayrollResultRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.PayrollResult, error) {
	row, err := r.queries.GetPayrollResultIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get payroll result (including deleted)", err)
	}

	res, err := rowToPayrollResult(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	return res, nil
}

// FindByPeriod retrieves all active (non-deleted) payroll results for a specific payroll period.
// Returns an empty slice if no results exist for the period.
func (r *PayrollResultRepository) FindByPeriod(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResultsByPayrollPeriod(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results by payroll period", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindByPeriodIncludingDeleted retrieves all payroll results for a specific payroll period,
// including soft-deleted records. Returns an empty slice if no results exist for the period.
func (r *PayrollResultRepository) FindByPeriodIncludingDeleted(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResultsByPayrollPeriodIncludingDeleted(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results by payroll period (including deleted)", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindByEmployee retrieves all active (non-deleted) payroll results for a specific employee.
// Returns an empty slice if no payroll history exists for the employee.
// This provides the complete payroll history for an employee across all periods.
func (r *PayrollResultRepository) FindByEmployee(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResultsByEmployee(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results by employee", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindByEmployeeIncludingDeleted retrieves all payroll results for a specific employee,
// including soft-deleted records. Returns an empty slice if no payroll history exists.
func (r *PayrollResultRepository) FindByEmployeeIncludingDeleted(ctx context.Context, id uuid.UUID) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResultsByEmployeeIncludingDeleted(ctx, id.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results by employee (including deleted)", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindAll retrieves all active (non-deleted) payroll results across all periods and employees.
// Returns an empty slice if no payroll results exist.
func (r *PayrollResultRepository) FindAll(ctx context.Context) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResults(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// FindAllIncludingDeleted retrieves all payroll results, including soft-deleted records.
// Returns an empty slice if no payroll results exist.
func (r *PayrollResultRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.PayrollResult, error) {
	rows, err := r.queries.ListPayrollResultsIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list payroll results (including deleted)", err)
	}

	results := make([]*domain.PayrollResult, 0, len(rows))
	for _, row := range rows {
		res, err := rowToPayrollResult(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "payroll result", err)
		}
		results = append(results, res)
	}

	return results, nil
}

// ============================================================================
// Mutation Methods
// ============================================================================

// Create persists a new payroll result and creates an audit log entry.
// The operation is atomic - both the payroll result and audit log are created in a single transaction.
func (r *PayrollResultRepository) Create(ctx context.Context, res *domain.PayrollResult) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	params := payrollResultToCreateParams(res)
	row, err := qtx.CreatePayrollResult(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "create payroll result", err)
	}

	resCreated, err := rowToPayrollResult(row)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		PayrollResultTableName,
		resCreated.ID.String(),
		DBActionCreate,
		nil,
		resCreated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Delete soft-deletes a payroll result by setting deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the payroll result doesn't exist or is already deleted.
// The operation is atomic - both the soft delete and audit log are created in a single transaction.
func (r *PayrollResultRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	resOldRow, err := qtx.GetPayrollResult(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "delete payroll result", err)
	}

	resOld, err := rowToPayrollResult(resOldRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	params := payrollResultToDeleteParams(resOld)
	resDeletedRow, err := qtx.DeletePayrollResult(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "deleted payroll result", err)
	}

	resDeleted, err := rowToPayrollResult(resDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		PayrollResultTableName,
		resOld.ID.String(),
		DBActionDelete,
		resOld,
		resDeleted,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Restore un-deletes a soft-deleted payroll result by clearing deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the payroll result doesn't exist.
// The operation is atomic - both the restoration and audit log are created in a single transaction.
func (r *PayrollResultRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	resDeletedRow, err := qtx.GetPayrollResultIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get payroll result including deleted", err)
	}

	resDeleted, err := rowToPayrollResult(resDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	params := payrollResultToRestoreParams(resDeleted)
	resRestoredRow, err := qtx.RestorePayrollResult(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "restore payroll result", err)
	}

	resRestored, err := rowToPayrollResult(resRestoredRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		PayrollResultTableName,
		resDeleted.ID.String(),
		DBActionRestore,
		resDeleted,
		resRestored,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// HardDelete permanently removes a payroll result from the database.
// Creates an audit log entry before deletion (audit log survives the deletion).
// Returns ErrRecordNotFound if the payroll result doesn't exist.
// WARNING: This operation is irreversible. Use Delete() for soft deletion instead.
// The operation is atomic - both the deletion and audit log are created in a single transaction.
func (r *PayrollResultRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	resOldRow, err := qtx.GetPayrollResultIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "delete payroll result (including deleted)", err)
	}

	resOld, err := rowToPayrollResult(resOldRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "payroll result", err)
	}

	if err := qtx.HardDeletePayrollResult(ctx, resOld.ID.String()); err != nil {
		return fmt.Errorf(FmtDBQueryErr, "hard delete payroll result", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		PayrollResultTableName,
		resOld.ID.String(),
		DBActionHardDelete,
		resOld,
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
func (r *PayrollResultRepository) WithTx(tx *sql.Tx) *PayrollResultRepository {
	return &PayrollResultRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToPayrollResult converts a sqlc-generated PayrollResult row to a domain.PayrollResult.
// Returns an error if any field fails to parse.
func rowToPayrollResult(row sqldb.PayrollResult) (*domain.PayrollResult, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "id as uuid", err)
	}

	payrollPeriodID, err := uuid.Parse(row.PayrollPeriodID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "payroll_period_id as uuid", err)
	}

	employeeID, err := uuid.Parse(row.EmployeeID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "employee_id as uuid", err)
	}

	compensationPkgID, err := uuid.Parse(row.CompensationPackageID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "compensation_package_id as uuid", err)
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

	return &domain.PayrollResult{
		ID:                                 id,
		PayrollPeriodID:                    payrollPeriodID,
		EmployeeID:                         employeeID,
		CompensationPackageID:              compensationPkgID,
		Currency:                           money.Currency(row.Currency),
		BaseSalary:                         money.FromCents(row.BaseSalaryCents),
		SeniorityBonus:                     money.FromCents(row.SeniorityBonusCents),
		GrossSalary:                        money.FromCents(row.GrossSalaryCents),
		TotalOtherBonus:                    money.FromCents(row.TotalOtherBonusCents),
		GrossSalaryGrandTotal:              money.FromCents(row.GrossSalaryGrandTotalCents),
		TotalExemptions:                    money.FromCents(row.TotalExemptionsCents),
		TaxableGrossSalary:                 money.FromCents(row.TaxableGrossSalaryCents),
		SocialAllowanceEmployeeContrib:     money.FromCents(row.SocialAllowanceEmployeeContribCents),
		SocialAllowanceEmployerContrib:     money.FromCents(row.SocialAllowanceEmployerContribCents),
		JobLossCompensationEmployeeContrib: money.FromCents(row.JobLossCompensationEmployeeContribCents),
		JobLossCompensationEmployerContrib: money.FromCents(row.JobLossCompensationEmployerContribCents),
		TrainingTaxEmployerContrib:         money.FromCents(row.TrainingTaxEmployerContribCents),
		FamilyBenefitsEmployerContrib:      money.FromCents(row.FamilyBenefitsEmployerContribCents),
		TotalCNSSEmployeeContrib:           money.FromCents(row.TotalCnssEmployeeContribCents),
		TotalCNSSEmployerContrib:           money.FromCents(row.TotalCnssEmployerContribCents),
		AMOEmployeeContrib:                 money.FromCents(row.AmoEmployeeContribCents),
		AMOEmployerContrib:                 money.FromCents(row.AmoEmployerContribCents),
		TaxableNetSalary:                   money.FromCents(row.TaxableNetSalaryCents),
		IncomeTax:                          money.FromCents(row.IncomeTaxCents),
		RoundingAmount:                     money.FromCents(row.RoundingAmountCents),
		NetToPay:                           money.FromCents(row.NetToPayCents),
		CreatedAt:                          createdAt,
		UpdatedAt:                          updatedAt,
		DeletedAt:                          deletedAt,
	}, nil
}

// payrollResultToCreateParams converts a domain.PayrollResult to sqlc CreatePayrollResultParams.
func payrollResultToCreateParams(res *domain.PayrollResult) sqldb.CreatePayrollResultParams {
	return sqldb.CreatePayrollResultParams{
		ID:                                      res.ID.String(),
		PayrollPeriodID:                         res.PayrollPeriodID.String(),
		EmployeeID:                              res.EmployeeID.String(),
		CompensationPackageID:                   res.CompensationPackageID.String(),
		Currency:                                res.Currency.String(),
		BaseSalaryCents:                         res.BaseSalary.Cents(),
		SeniorityBonusCents:                     res.SeniorityBonus.Cents(),
		GrossSalaryCents:                        res.GrossSalary.Cents(),
		TotalOtherBonusCents:                    res.TotalOtherBonus.Cents(),
		GrossSalaryGrandTotalCents:              res.GrossSalaryGrandTotal.Cents(),
		TotalExemptionsCents:                    res.TotalExemptions.Cents(),
		TaxableGrossSalaryCents:                 res.TaxableGrossSalary.Cents(),
		SocialAllowanceEmployeeContribCents:     res.SocialAllowanceEmployeeContrib.Cents(),
		SocialAllowanceEmployerContribCents:     res.SocialAllowanceEmployerContrib.Cents(),
		JobLossCompensationEmployeeContribCents: res.JobLossCompensationEmployeeContrib.Cents(),
		JobLossCompensationEmployerContribCents: res.JobLossCompensationEmployerContrib.Cents(),
		TrainingTaxEmployerContribCents:         res.TrainingTaxEmployerContrib.Cents(),
		FamilyBenefitsEmployerContribCents:      res.FamilyBenefitsEmployerContrib.Cents(),
		TotalCnssEmployeeContribCents:           res.TotalCNSSEmployeeContrib.Cents(),
		TotalCnssEmployerContribCents:           res.TotalCNSSEmployerContrib.Cents(),
		AmoEmployeeContribCents:                 res.AMOEmployeeContrib.Cents(),
		AmoEmployerContribCents:                 res.AMOEmployerContrib.Cents(),
		TaxableNetSalaryCents:                   res.TaxableNetSalary.Cents(),
		IncomeTaxCents:                          res.IncomeTax.Cents(),
		RoundingAmountCents:                     res.RoundingAmount.Cents(),
		NetToPayCents:                           res.NetToPay.Cents(),
		CreatedAt:                               res.CreatedAt.Format(DBTimeFormat),
		UpdatedAt:                               res.UpdatedAt.Format(DBTimeFormat),
	}
}

// payrollResultToDeleteParams converts a domain.PayrollResult to sqlc DeletePayrollResultParams.
// Sets both deleted_at and updated_at to current time.
func payrollResultToDeleteParams(res *domain.PayrollResult) sqldb.DeletePayrollResultParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.DeletePayrollResultParams{
		ID:        res.ID.String(),
		UpdatedAt: now,
		DeletedAt: stringToNullString(now),
	}
}

// payrollResultToRestoreParams converts a domain.PayrollResult to sqlc RestorePayrollResultParams.
// Sets updated_at to current time.
func payrollResultToRestoreParams(res *domain.PayrollResult) sqldb.RestorePayrollResultParams {
	return sqldb.RestorePayrollResultParams{
		ID:        res.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// ============================================================================
// Constants
// ============================================================================

const (
	// Table name constant for audit logging.
	PayrollResultTableName = "payroll_result"
)
