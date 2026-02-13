package sqlite_adapter

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite/sqldb"
	"github.com/iamoeg/bootdev-capstone/internal/domain"
)

// ============================================================================
// Employee Repository
// ============================================================================

// EmployeeRepository implements employee data persistence using SQLite.
// It provides CRUD operations with soft delete support, audit logging,
// and serial number management for per-organization employee numbering.
type EmployeeRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewEmployeeRepository creates a new employee repository instance.
func NewEmployeeRepository(db *sql.DB) *EmployeeRepository {
	return &EmployeeRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindByID retrieves an active (non-deleted) employee by ID.
// Returns ErrRecordNotFound if the employee doesn't exist or is soft-deleted.
func (r *EmployeeRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	row, err := r.queries.GetEmployee(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get employee", err)
	}

	emp, err := rowToEmployee(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	return emp, nil
}

// FindByIDIncludingDeleted retrieves an employee by ID, including soft-deleted records.
// Returns ErrRecordNotFound if the employee doesn't exist.
func (r *EmployeeRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Employee, error) {
	row, err := r.queries.GetEmployeeIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get employee (including deleted)", err)
	}

	emp, err := rowToEmployee(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	return emp, nil
}

// FindByOrgAndSerialNum retrieves an active employee by organization ID and serial number.
// Returns ErrRecordNotFound if no matching employee exists or if the employee is soft-deleted.
func (r *EmployeeRepository) FindByOrgAndSerialNum(ctx context.Context, orgID uuid.UUID, serialNum int) (*domain.Employee, error) {
	params := sqldb.GetEmployeeByOrgAndSerialNumParams{
		OrgID:     orgID.String(),
		SerialNum: int64(serialNum),
	}
	row, err := r.queries.GetEmployeeByOrgAndSerialNum(ctx, params)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get employee by org id and serial num", err)
	}

	emp, err := rowToEmployee(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if emp.DeletedAt != nil {
		return nil, ErrRecordNotFound
	}

	return emp, nil
}

// FindByOrganization retrieves all active (non-deleted) employees for a given organization.
// Returns an empty slice if no employees exist for the organization.
func (r *EmployeeRepository) FindByOrganization(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error) {
	rows, err := r.queries.ListEmployeesByOrganization(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employees by organization", err)
	}

	emps := make([]*domain.Employee, 0, len(rows))
	for _, row := range rows {
		emp, err := rowToEmployee(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
		}

		emps = append(emps, emp)
	}

	return emps, nil
}

// FindByOrganizationIncludingDeleted retrieves all employees for a given organization,
// including soft-deleted records. Returns an empty slice if no employees exist.
func (r *EmployeeRepository) FindByOrganizationIncludingDeleted(ctx context.Context, orgID uuid.UUID) ([]*domain.Employee, error) {
	rows, err := r.queries.ListEmployeesByOrganizationIncludingDeleted(ctx, orgID.String())
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employees by organization (including deleted)", err)
	}

	emps := make([]*domain.Employee, 0, len(rows))
	for _, row := range rows {
		e, err := rowToEmployee(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
		}

		emps = append(emps, e)
	}

	return emps, nil
}

// FindAll retrieves all active (non-deleted) employees across all organizations.
// Returns an empty slice if no employees exist.
func (r *EmployeeRepository) FindAll(ctx context.Context) ([]*domain.Employee, error) {
	rows, err := r.queries.ListEmployees(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employees", err)
	}

	emps := make([]*domain.Employee, 0, len(rows))
	for _, row := range rows {
		e, err := rowToEmployee(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
		}
		emps = append(emps, e)
	}

	return emps, nil
}

// FindAllIncludingDeleted retrieves all employees across all organizations,
// including soft-deleted records. Returns an empty slice if no employees exist.
func (r *EmployeeRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.Employee, error) {
	rows, err := r.queries.ListEmployeesIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list employees (including deleted)", err)
	}

	emps := make([]*domain.Employee, 0, len(rows))
	for _, row := range rows {
		e, err := rowToEmployee(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "employee", err)
		}
		emps = append(emps, e)
	}

	return emps, nil
}

// GetNextSerialNumber returns the next available serial number for employees
// within a given organization. Serial numbers are per-organization and start at 1.
// Returns 1 if the organization has no employees yet.
func (r *EmployeeRepository) GetNextSerialNumber(ctx context.Context, orgID uuid.UUID) (int, error) {
	sn, err := r.queries.GetNextSerialNumber(ctx, orgID.String())
	if err != nil {
		return 0, fmt.Errorf(FmtDBQueryErr, "get next serial number", err)
	}
	return int(sn), nil
}

// ============================================================================
// Mutation Methods
// ============================================================================

// Create persists a new employee and creates an audit log entry.
// The operation is atomic - both the employee and audit log are created in a single transaction.
func (r *EmployeeRepository) Create(ctx context.Context, emp *domain.Employee) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	params := employeeToCreateParams(emp)
	row, err := qtx.CreateEmployee(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrDuplicateRecord
		}
		return fmt.Errorf(FmtDBQueryErr, "create employee", err)
	}

	empCreated, err := rowToEmployee(row)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		EmployeeTableName,
		empCreated.ID.String(),
		DBActionCreate,
		nil,
		empCreated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Update modifies an existing employee and creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the employee doesn't exist or is soft-deleted.
// Note: org_id and serial_num cannot be updated (immutable identity fields).
// The operation is atomic - both the update and audit log are created in a single transaction.
func (r *EmployeeRepository) Update(ctx context.Context, emp *domain.Employee) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	empOldRow, err := qtx.GetEmployee(ctx, emp.ID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee", err)
	}

	empOld, err := rowToEmployee(empOldRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	params := employeeToUpdateParams(emp)
	empUpdatedRow, err := qtx.UpdateEmployee(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "update employee", err)
	}

	empUpdated, err := rowToEmployee(empUpdatedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		EmployeeTableName,
		emp.ID.String(),
		DBActionUpdate,
		empOld,
		empUpdated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Delete soft-deletes an employee by setting deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the employee doesn't exist or is already deleted.
// The operation is atomic - both the soft delete and audit log are created in a single transaction.
func (r *EmployeeRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	empRow, err := qtx.GetEmployee(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee", err)
	}

	emp, err := rowToEmployee(empRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	params := employeeToDeleteParams(emp)
	empDeletedRow, err := qtx.DeleteEmployee(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "delete employee", err)
	}

	empDeleted, err := rowToEmployee(empDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		EmployeeTableName,
		emp.ID.String(),
		DBActionDelete,
		emp,
		empDeleted,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Restore un-deletes a soft-deleted employee by clearing deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the employee doesn't exist.
// The operation is atomic - both the restoration and audit log are created in a single transaction.
func (r *EmployeeRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	empDeletedRow, err := qtx.GetEmployeeIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee (including deleted)", err)
	}

	empDeleted, err := rowToEmployee(empDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	params := employeeToRestoreParams(empDeleted)
	empRestoredRow, err := qtx.RestoreEmployee(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "restore employee", err)
	}

	empRestored, err := rowToEmployee(empRestoredRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		EmployeeTableName,
		empDeleted.ID.String(),
		DBActionRestore,
		empDeleted,
		empRestored,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// HardDelete permanently removes an employee from the database.
// Creates an audit log entry before deletion (audit log survives the deletion).
// Returns ErrRecordNotFound if the employee doesn't exist.
// WARNING: This operation is irreversible. Use Delete() for soft deletion instead.
// The operation is atomic - both the deletion and audit log are created in a single transaction.
func (r *EmployeeRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	empRow, err := qtx.GetEmployeeIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get employee (including deleted)", err)
	}

	emp, err := rowToEmployee(empRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "employee", err)
	}

	if err := qtx.HardDeleteEmployee(ctx, emp.ID.String()); err != nil {
		return fmt.Errorf(FmtDBQueryErr, "hard delete employee", err)
	}

	if err := createAuditLog(
		ctx, qtx,
		EmployeeTableName,
		emp.ID.String(),
		DBActionHardDelete,
		emp,
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
func (r *EmployeeRepository) WithTx(tx *sql.Tx) *EmployeeRepository {
	return &EmployeeRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToEmployee converts a sqlc-generated Employee row to a domain.Employee.
// Returns an error if any field fails to parse.
func rowToEmployee(row sqldb.Employee) (*domain.Employee, error) {
	id, err := uuid.Parse(row.ID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "id as uuid", err)
	}

	orgID, err := uuid.Parse(row.OrgID)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "org_id as uuid", err)
	}

	birthDate, err := time.Parse(DBTimeFormat, row.BirthDate)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "birth_date as time", err)
	}

	hireDate, err := time.Parse(DBTimeFormat, row.HireDate)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "hire_date as time", err)
	}

	compPackID, err := uuid.Parse(row.CompensationPackageID)
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

	return &domain.Employee{
		ID:                    id,
		OrgID:                 orgID,
		SerialNum:             int(row.SerialNum),
		FullName:              row.FullName,
		DisplayName:           nullStringToString(row.DisplayName),
		Address:               nullStringToString(row.Address),
		EmailAddress:          nullStringToString(row.EmailAddress),
		PhoneNumber:           nullStringToString(row.PhoneNumber),
		BirthDate:             birthDate,
		Gender:                domain.GenderEnum(row.Gender),
		MaritalStatus:         domain.MaritalStatusEnum(row.MaritalStatus),
		NumDependents:         int(row.NumDependents),
		NumKids:               int(row.NumKids),
		CINNum:                row.CinNum,
		CNSSNum:               nullStringToString(row.CnssNum),
		HireDate:              hireDate,
		Position:              row.Position,
		CompensationPackageID: compPackID,
		BankRIB:               nullStringToString(row.BankRib),
		CreatedAt:             createdAt,
		UpdatedAt:             updatedAt,
		DeletedAt:             deletedAt,
	}, nil

}

// employeeToCreateParams converts a domain.Employee to sqlc CreateEmployeeParams.
func employeeToCreateParams(emp *domain.Employee) sqldb.CreateEmployeeParams {
	return sqldb.CreateEmployeeParams{
		ID:                    emp.ID.String(),
		OrgID:                 emp.OrgID.String(),
		SerialNum:             int64(emp.SerialNum),
		FullName:              emp.FullName,
		DisplayName:           stringToNullString(emp.DisplayName),
		Address:               stringToNullString(emp.Address),
		EmailAddress:          stringToNullString(emp.EmailAddress),
		PhoneNumber:           stringToNullString(emp.PhoneNumber),
		BirthDate:             emp.BirthDate.Format(DBTimeFormat),
		Gender:                string(emp.Gender),
		MaritalStatus:         string(emp.MaritalStatus),
		NumDependents:         int64(emp.NumDependents),
		NumKids:               int64(emp.NumKids),
		CinNum:                emp.CINNum,
		CnssNum:               stringToNullString(emp.CNSSNum),
		HireDate:              emp.HireDate.Format(DBTimeFormat),
		Position:              emp.Position,
		CompensationPackageID: emp.CompensationPackageID.String(),
		BankRib:               stringToNullString(emp.BankRIB),
		CreatedAt:             emp.CreatedAt.Format(DBTimeFormat),
		UpdatedAt:             emp.UpdatedAt.Format(DBTimeFormat),
	}
}

// employeeToUpdateParams converts a domain.Employee to sqlc UpdateEmployeeParams.
// Note: UpdatedAt is set to current time, not taken from the domain object.
// Note: org_id and serial_num are immutable and excluded from updates.
func employeeToUpdateParams(emp *domain.Employee) sqldb.UpdateEmployeeParams {
	return sqldb.UpdateEmployeeParams{
		FullName:              emp.FullName,
		DisplayName:           stringToNullString(emp.DisplayName),
		Address:               stringToNullString(emp.Address),
		EmailAddress:          stringToNullString(emp.EmailAddress),
		PhoneNumber:           stringToNullString(emp.PhoneNumber),
		BirthDate:             emp.BirthDate.Format(DBTimeFormat),
		Gender:                string(emp.Gender),
		MaritalStatus:         string(emp.MaritalStatus),
		NumDependents:         int64(emp.NumDependents),
		NumKids:               int64(emp.NumKids),
		CinNum:                emp.CINNum,
		CnssNum:               stringToNullString(emp.CNSSNum),
		HireDate:              emp.HireDate.Format(DBTimeFormat),
		Position:              emp.Position,
		CompensationPackageID: emp.CompensationPackageID.String(),
		BankRib:               stringToNullString(emp.BankRIB),
		UpdatedAt:             time.Now().Format(DBTimeFormat),
		ID:                    emp.ID.String(),
	}
}

// employeeToDeleteParams converts a domain.Employee to sqlc DeleteEmployeeParams.
// Sets both deleted_at and updated_at to current time.
func employeeToDeleteParams(emp *domain.Employee) sqldb.DeleteEmployeeParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.DeleteEmployeeParams{
		ID:        emp.ID.String(),
		UpdatedAt: now,
		DeletedAt: sql.NullString{
			String: now,
			Valid:  true,
		},
	}
}

// employeeToRestoreParams converts a domain.Employee to sqlc RestoreEmployeeParams.
// Sets updated_at to current time.
func employeeToRestoreParams(emp *domain.Employee) sqldb.RestoreEmployeeParams {
	return sqldb.RestoreEmployeeParams{
		ID:        emp.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// ============================================================================
// Constants
// ============================================================================

// Table name constant for audit logging.
const (
	EmployeeTableName = "employee"
)
