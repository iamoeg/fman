package sqlite_adapter

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite/sqldb"
)

// ============================================================================
// Audit Log (Adapter Type)
// ============================================================================

// AuditLog represents a database change audit record.
// This is an adapter-level type, not a domain entity.
// Audit logs are infrastructure for compliance and debugging.
type AuditLog struct {
	ID        string    // Unique identifier for this audit entry
	TableName string    // Name of the table that was modified
	RecordID  string    // ID of the record that was modified
	Action    string    // Type of action (CREATE, UPDATE, DELETE, RESTORE, HARD_DELETE)
	Before    string    // JSON snapshot of the record before modification (empty for CREATE)
	After     string    // JSON snapshot of the record after modification (empty for HARD_DELETE)
	Timestamp time.Time // When the modification occurred
}

// ============================================================================
// Audit Log Repository
// ============================================================================

// AuditLogRepository provides read-only access to audit log records.
// Audit logs are created automatically as a side effect of repository mutations
// via the createAuditLog helper function in util.go.
//
// This repository is intentionally read-only to prevent direct manipulation
// of the audit trail, ensuring its integrity for compliance and debugging.
type AuditLogRepository struct {
	db      *sql.DB
	queries *sqldb.Queries
}

// NewAuditLogRepository creates a new audit log repository instance.
func NewAuditLogRepository(db *sql.DB) *AuditLogRepository {
	return &AuditLogRepository{
		db:      db,
		queries: sqldb.New(db),
	}
}

// ============================================================================
// Query Methods
// ============================================================================

// FindForRecord retrieves all audit log entries for a specific record.
// Returns entries ordered by timestamp (oldest first).
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - tableName: Name of the table (e.g., "organization", "employee")
//   - recordID: UUID of the record to get audit history for
//
// Returns an empty slice if no audit logs exist for the record.
//
// Example:
//
//	logs, err := repo.FindForRecord(ctx, "employee", employeeID)
//	// logs contains complete history of changes to this employee
func (r *AuditLogRepository) FindForRecord(ctx context.Context, tableName string, recordID string) ([]*AuditLog, error) {
	params := sqldb.ListAuditLogsForRecordParams{
		TableName: tableName,
		RecordID:  recordID,
	}
	rows, err := r.queries.ListAuditLogsForRecord(ctx, params)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list audit logs for record", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := rowToAuditLog(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "audit log", err)
		}
		logs = append(logs, auditLog)
	}

	return logs, nil
}

// FindRecent retrieves the most recent audit log entries across all tables.
// Returns entries ordered by timestamp (newest first).
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - limit: Maximum number of entries to return (e.g., 100)
//
// Useful for displaying recent system activity or changes.
//
// Example:
//
//	logs, err := repo.FindRecent(ctx, 50)
//	// logs contains the 50 most recent changes across the entire system
func (r *AuditLogRepository) FindRecent(ctx context.Context, limit int) ([]*AuditLog, error) {
	rows, err := r.queries.ListAuditLogsRecent(ctx, int64(limit))
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list recent audit logs", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := rowToAuditLog(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "audit log", err)
		}
		logs = append(logs, auditLog)
	}

	return logs, nil
}

// FindByTable retrieves audit log entries for a specific table.
// Returns entries ordered by timestamp (newest first).
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - tableName: Name of the table (e.g., "organization", "employee")
//   - limit: Maximum number of entries to return
//
// Useful for understanding what changes have been made to a particular entity type.
//
// Example:
//
//	logs, err := repo.FindByTable(ctx, "payroll_result", 100)
//	// logs contains recent changes to payroll results
func (r *AuditLogRepository) FindByTable(ctx context.Context, tableName string, limit int) ([]*AuditLog, error) {
	params := sqldb.ListAuditLogsByTableParams{
		TableName: tableName,
		Limit:     int64(limit),
	}
	rows, err := r.queries.ListAuditLogsByTable(ctx, params)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list audit logs by table", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := rowToAuditLog(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "audit log", err)
		}
		logs = append(logs, auditLog)
	}

	return logs, nil
}

// FindByAction retrieves audit log entries for a specific action type.
// Returns entries ordered by timestamp (newest first).
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - action: Type of action (CREATE, UPDATE, DELETE, RESTORE, HARD_DELETE)
//   - limit: Maximum number of entries to return
//
// Useful for compliance queries like "show all deletions" or "show all restorations".
//
// Example:
//
//	logs, err := repo.FindByAction(ctx, "DELETE", 50)
//	// logs contains the 50 most recent soft-delete operations
func (r *AuditLogRepository) FindByAction(ctx context.Context, action string, limit int) ([]*AuditLog, error) {
	params := sqldb.ListAuditLogsByActionParams{
		Action: action,
		Limit:  int64(limit),
	}
	rows, err := r.queries.ListAuditLogsByAction(ctx, params)
	if err != nil {
		return nil, fmt.Errorf(FmtDBQueryErr, "list audit logs by action", err)
	}

	logs := make([]*AuditLog, 0, len(rows))
	for _, row := range rows {
		auditLog, err := rowToAuditLog(row)
		if err != nil {
			return nil, fmt.Errorf(FmtRowParsingErr, "audit log", err)
		}
		logs = append(logs, auditLog)
	}

	return logs, nil
}

// ============================================================================
// Transaction Support
// ============================================================================

// WithTx returns a new repository instance that uses the provided transaction.
// This allows audit log queries to participate in transactions managed by the caller.
//
// Note: In practice, this is rarely needed since audit logs are typically queried
// outside of transactions. The createAuditLog helper function in util.go handles
// transactional audit log creation.
func (r *AuditLogRepository) WithTx(tx *sql.Tx) *AuditLogRepository {
	return &AuditLogRepository{
		db:      r.db,
		queries: r.queries.WithTx(tx),
	}
}

// ============================================================================
// Helper Functions - Row Conversion
// ============================================================================

// rowToAuditLog converts a sqlc-generated AuditLog row to an adapter AuditLog type.
// Returns an error if the timestamp fails to parse.
func rowToAuditLog(row sqldb.AuditLog) (*AuditLog, error) {
	timestamp, err := time.Parse(DBTimeFormat, row.Timestamp)
	if err != nil {
		return nil, fmt.Errorf(FmtColumnParsingErr, "timestamp", err)
	}

	return &AuditLog{
		ID:        row.ID,
		TableName: row.TableName,
		RecordID:  row.RecordID,
		Action:    row.Action,
		Before:    row.Before.String, // NullString - will be empty for CREATE actions
		After:     row.After,         // Always populated (may be "null" for HARD_DELETE)
		Timestamp: timestamp,
	}, nil
}

// ============================================================================
// Constants
// ============================================================================

const (
	// Table name constant for consistency with other repositories.
	// Note: This is primarily for documentation - audit logs are created via
	// the createAuditLog helper, not through this repository.
	AuditLogTableName = "audit_log"
)
