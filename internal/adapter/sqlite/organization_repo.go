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
// Organization Repository
// ============================================================================

// OrganizationRepository implements organization data persistence using SQLite.
// It provides CRUD operations with soft delete support and audit logging.
type OrganizationRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewOrganizationRepository creates a new organization repository instance.
func NewOrganizationRepository(db *sql.DB) *OrganizationRepository {
	return &OrganizationRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindByID retrieves an active (non-deleted) organization by ID.
// Returns ErrRecordNotFound if the organization doesn't exist or is soft-deleted.
func (r *OrganizationRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	row, err := r.queries.GetOrganization(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get organization", err)
	}

	org, err := rowToOrganization(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "organization", err)
	}
	return org, nil
}

// FindByIDIncludingDeleted retrieves an organization by ID, including soft-deleted records.
// Returns ErrRecordNotFound if the organization doesn't exist.
func (r *OrganizationRepository) FindByIDIncludingDeleted(ctx context.Context, id uuid.UUID) (*domain.Organization, error) {
	row, err := r.queries.GetOrganizationIncludingDeleted(ctx, id.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrRecordNotFound
		}
		return nil, fmt.Errorf(FmtDBQueryErr, "get organization (including deleted)", err)
	}

	org, err := rowToOrganization(row)
	if err != nil {
		return nil, fmt.Errorf(FmtRowParsingErr, "organization", err)
	}
	return org, nil
}

// FindAll retrieves all active (non-deleted) organizations.
// Returns an empty slice if no organizations exist.
func (r *OrganizationRepository) FindAll(ctx context.Context) ([]*domain.Organization, error) {
	rows, err := r.queries.ListOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list organizations", err)
	}

	orgs := make([]*domain.Organization, 0, len(rows))
	for _, row := range rows {
		o, err := rowToOrganization(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "organization", err)
		}
		orgs = append(orgs, o)
	}

	return orgs, nil
}

// FindAllIncludingDeleted retrieves all organizations, including soft-deleted records.
// Returns an empty slice if no organizations exist.
func (r *OrganizationRepository) FindAllIncludingDeleted(ctx context.Context) ([]*domain.Organization, error) {
	rows, err := r.queries.ListOrganizationsIncludingDeleted(ctx)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list organizations (including deleted)", err)
	}

	orgs := make([]*domain.Organization, 0, len(rows))
	for _, row := range rows {
		org, err := rowToOrganization(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "organization", err)
		}
		orgs = append(orgs, org)
	}

	return orgs, nil
}

// ============================================================================
// Mutation Methods
// ============================================================================

// Create persists a new organization and creates an audit log entry.
// The operation is atomic - both the organization and audit log are created in a single transaction.
func (r *OrganizationRepository) Create(ctx context.Context, org *domain.Organization) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	params := organizationToCreateParams(org)
	row, err := qtx.CreateOrganization(ctx, params)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			return ErrDuplicateRecord
		}
		return fmt.Errorf(FmtDBQueryErr, "create organization", err)
	}

	orgCreated, err := rowToOrganization(row)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	if err = createAuditLog(
		ctx,
		qtx,
		OrganizationTableName,
		orgCreated.ID.String(),
		DBActionCreate,
		nil,
		orgCreated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Update modifies an existing organization and creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the organization doesn't exist or is soft-deleted.
// The operation is atomic - both the update and audit log are created in a single transaction.
func (r *OrganizationRepository) Update(ctx context.Context, org *domain.Organization) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	orgOldRow, err := qtx.GetOrganization(ctx, org.ID.String())
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrRecordNotFound
		}
		return fmt.Errorf(FmtDBQueryErr, "get organization", err)
	}

	orgOld, err := rowToOrganization(orgOldRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	params := organizationToUpdateParams(org)
	orgUpdatedRow, err := qtx.UpdateOrganization(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "update organization", err)
	}

	orgUpdated, err := rowToOrganization(orgUpdatedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		OrganizationTableName,
		org.ID.String(),
		DBActionUpdate,
		orgOld,
		orgUpdated,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Delete soft-deletes an organization by setting deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the organization doesn't exist.
// The operation is atomic - both the soft delete and audit log are created in a single transaction.
func (r *OrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	orgRow, err := qtx.GetOrganization(ctx, id.String())
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "get organization", err)
	}

	org, err := rowToOrganization(orgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	params := organizationToDeleteParams(org)
	orgDeletedRow, err := qtx.DeleteOrganization(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "delete organization", err)
	}

	orgDeleted, err := rowToOrganization(orgDeletedRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		OrganizationTableName,
		org.ID.String(),
		DBActionDelete,
		org,
		orgDeleted,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// Restore un-deletes a soft-deleted organization by clearing deleted_at timestamp.
// Creates an audit log entry with before/after snapshots.
// Returns ErrRecordNotFound if the organization doesn't exist.
// The operation is atomic - both the restoration and audit log are created in a single transaction.
func (r *OrganizationRepository) Restore(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	deletedOrgRow, err := qtx.GetOrganizationIncludingDeleted(ctx, id.String())
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "get organization (including deleted)", err)
	}

	deletedOrg, err := rowToOrganization(deletedOrgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	params := organizationToRestoreParams(deletedOrg)
	restoredOrgRow, err := qtx.RestoreOrganization(ctx, params)
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "restore organization", err)
	}

	restoredOrg, err := rowToOrganization(restoredOrgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		OrganizationTableName,
		deletedOrg.ID.String(),
		DBActionRestore,
		deletedOrg,
		restoredOrg,
	); err != nil {
		return fmt.Errorf(FmtAuditLogErr, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(FmtCommitTxErr, err)
	}

	return nil
}

// HardDelete permanently removes an organization from the database.
// Creates an audit log entry before deletion (audit log survives the deletion).
// Returns ErrRecordNotFound if the organization doesn't exist.
// WARNING: This operation is irreversible. Use Delete() for soft deletion instead.
// The operation is atomic - both the deletion and audit log are created in a single transaction.
func (r *OrganizationRepository) HardDelete(ctx context.Context, id uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(FmtBeginTxErr, err)
	}
	defer tx.Rollback()

	qtx := r.queries.WithTx(tx)

	deletedOrgRow, err := qtx.GetOrganizationIncludingDeleted(ctx, id.String())
	if err != nil {
		return fmt.Errorf(FmtDBQueryErr, "get organization (including deleted)", err)
	}

	deletedOrg, err := rowToOrganization(deletedOrgRow)
	if err != nil {
		return fmt.Errorf(FmtRowParsingErr, "organization", err)
	}

	if err := qtx.HardDeleteOrganization(ctx, deletedOrg.ID.String()); err != nil {
		return fmt.Errorf(FmtDBQueryErr, "hard delete organization", err)
	}

	if err := createAuditLog(
		ctx,
		qtx,
		OrganizationTableName,
		deletedOrg.ID.String(),
		DBActionHardDelete,
		deletedOrg,
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
func (r *OrganizationRepository) WithTx(tx *sql.Tx) *OrganizationRepository {
	return &OrganizationRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToOrganization converts a sqlc-generated Organization row to a domain.Organization.
// Returns an error if any field fails to parse.
func rowToOrganization(row sqldb.Organization) (*domain.Organization, error) {
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

	return &domain.Organization{
		ID:        id,
		Name:      row.Name,
		Address:   nullStringToString(row.Address),
		Activity:  nullStringToString(row.Activity),
		LegalForm: domain.OrgLegalFormEnum(nullStringToString(row.LegalForm)),
		ICENum:    nullStringToString(row.IceNum),
		IFNum:     nullStringToString(row.IfNum),
		RCNum:     nullStringToString(row.RcNum),
		CNSSNum:   nullStringToString(row.CnssNum),
		BankRIB:   nullStringToString(row.BankRib),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: deletedAt,
	}, nil
}

// organizationToCreateParams converts a domain.Organization to sqlc CreateOrganizationParams.
func organizationToCreateParams(org *domain.Organization) sqldb.CreateOrganizationParams {
	return sqldb.CreateOrganizationParams{
		ID:        org.ID.String(),
		Name:      org.Name,
		Address:   stringToNullString(org.Address),
		Activity:  stringToNullString(org.Activity),
		LegalForm: stringToNullString(string(org.LegalForm)),
		IceNum:    stringToNullString(org.ICENum),
		IfNum:     stringToNullString(org.IFNum),
		RcNum:     stringToNullString(org.RCNum),
		CnssNum:   stringToNullString(org.CNSSNum),
		BankRib:   stringToNullString(org.BankRIB),
		CreatedAt: org.CreatedAt.Format(DBTimeFormat),
		UpdatedAt: org.UpdatedAt.Format(DBTimeFormat),
	}
}

// organizationToUpdateParams converts a domain.Organization to sqlc UpdateOrganizationParams.
// Note: UpdatedAt is set to current time, not taken from the domain object.
func organizationToUpdateParams(org *domain.Organization) sqldb.UpdateOrganizationParams {
	return sqldb.UpdateOrganizationParams{
		Name:      org.Name,
		Address:   stringToNullString(org.Address),
		Activity:  stringToNullString(org.Activity),
		LegalForm: stringToNullString(string(org.LegalForm)),
		IceNum:    stringToNullString(org.ICENum),
		IfNum:     stringToNullString(org.IFNum),
		RcNum:     stringToNullString(org.RCNum),
		CnssNum:   stringToNullString(org.CNSSNum),
		BankRib:   stringToNullString(org.BankRIB),
		UpdatedAt: time.Now().Format(DBTimeFormat),
		ID:        org.ID.String(),
	}
}

// organizationToDeleteParams converts a domain.Organization to sqlc DeleteOrganizationParams.
// Sets both deleted_at and updated_at to current time.
func organizationToDeleteParams(org *domain.Organization) sqldb.DeleteOrganizationParams {
	now := time.Now().Format(DBTimeFormat)
	return sqldb.DeleteOrganizationParams{
		ID:        org.ID.String(),
		UpdatedAt: now,
		DeletedAt: stringToNullString(now),
	}
}

// organizationToRestoreParams converts a domain.Organization to sqlc RestoreOrganizationParams.
// Sets updated_at to current time.
func organizationToRestoreParams(org *domain.Organization) sqldb.RestoreOrganizationParams {
	return sqldb.RestoreOrganizationParams{
		ID:        org.ID.String(),
		UpdatedAt: time.Now().Format(DBTimeFormat),
	}
}

// ============================================================================
// Constants
// ============================================================================

// Table name constant for audit logging.
const (
	OrganizationTableName = "organization"
)
