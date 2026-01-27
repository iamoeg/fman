package sqlite_adapter

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite/sqldb"
	"github.com/iamoeg/bootdev-capstone/pkg/util"
)

// ============================================================================
// Audit Logging
// ============================================================================

// createAuditLog creates an audit log entry for a database mutation.
// It validates the action, marshals before/after snapshots to JSON,
// and persists the audit record using the provided transactional query executor.
//
// Parameters:
//   - ctx: Context for cancellation and deadlines
//   - qtx: Transactional query executor (ensures audit log is part of the same transaction)
//   - tableName: Name of the table being modified
//   - recordID: ID of the record being modified
//   - action: Type of action (CREATE, UPDATE, DELETE, RESTORE, HARD_DELETE)
//   - before: Snapshot of the record before modification (nil for CREATE)
//   - after: Snapshot of the record after modification (nil for HARD_DELETE)
//
// Returns an error if the action is not supported or if marshaling/persistence fails.
func createAuditLog(
	ctx context.Context,
	qtx *sqldb.Queries,
	tableName, recordID string,
	action DBActionEnum,
	before, after any,
) error {
	if !action.IsSupported() {
		return fmt.Errorf("%w: %s is not supported", ErrDBActionNotSupported, action)
	}

	var beforeJSON sql.NullString
	if before != nil {
		b, err := json.Marshal(before)
		if err != nil {
			return fmt.Errorf("failed to marshal before: %w", err)
		}
		beforeJSON = sql.NullString{String: string(b), Valid: true}
	}

	var afterJSON string
	if after != nil {
		a, err := json.Marshal(after)
		if err != nil {
			return fmt.Errorf("failed to marshal after: %w", err)
		}
		afterJSON = string(a)
	} else {
		// For HARD_DELETE, after is nil, so use "null" (valid JSON)
		afterJSON = "null"
	}

	return qtx.CreateAuditLog(ctx, sqldb.CreateAuditLogParams{
		ID:        uuid.New().String(),
		TableName: tableName,
		RecordID:  recordID,
		Action:    string(action),
		Before:    beforeJSON,
		After:     afterJSON,
		Timestamp: time.Now().UTC().Format(DBTimeFormat),
	})
}

// ============================================================================
// Database Action Enum
// ============================================================================

// DBActionEnum represents the type of database action recorded in audit logs.
type DBActionEnum string

// IsSupported returns true if the action is a valid, supported action type.
func (a DBActionEnum) IsSupported() bool {
	_, ok := supportedDBActions[a]
	return ok
}

// Supported database action types for audit logging.
const (
	DBActionCreate     DBActionEnum = "CREATE"      // Record was created
	DBActionUpdate     DBActionEnum = "UPDATE"      // Record was modified
	DBActionDelete     DBActionEnum = "DELETE"      // Record was soft-deleted
	DBActionRestore    DBActionEnum = "RESTORE"     // Record was un-deleted
	DBActionHardDelete DBActionEnum = "HARD_DELETE" // Record was permanently removed
)

// supportedDBActions is the set of valid database actions.
var supportedDBActions = map[DBActionEnum]struct{}{
	DBActionCreate:     {},
	DBActionUpdate:     {},
	DBActionDelete:     {},
	DBActionRestore:    {},
	DBActionHardDelete: {},
}

// SupportedDBActionsStr is a human-readable string of all supported actions.
// Useful for error messages and validation feedback.
var SupportedDBActionsStr = util.EnumMapToString(supportedDBActions)

// ============================================================================
// Helper Functions - Null String Conversion
// ============================================================================

// nullStringToString converts sql.NullString to string.
// Returns empty string if the NullString is not valid (SQL NULL).
func nullStringToString(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

// stringToNullString converts a string to sql.NullString.
// Empty strings are converted to SQL NULL (Valid: false).
func stringToNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{
		String: s,
		Valid:  true,
	}
}

// ============================================================================
// Constants
// ============================================================================

// DBTimeFormat is the standard time format used for all timestamp columns in SQLite.
// Uses RFC3339 for ISO 8601 compliance and human readability.
const (
	DBTimeFormat = time.RFC3339
)

// Error message format strings for consistent error wrapping.
const (
	FmtErrDBQuery       = "failed to perform database query %s: %w"
	FmtErrParseDBRow    = "failed to parse database row as %s: %w"
	FmtErrBeginDBTx     = "failed to begin database transaction: %w"
	FmtErrCommitDBTx    = "failed to commit database transaction: %w"
	FmtErrAuditLog      = "failed to create audit log entry: %w"
	FmtErrParseDBParams = "failed to parse %s as database query params: %w"
	FmtErrParseDBColumn = "failed to parse value %s: %w"
)

// ============================================================================
// Errors
// ============================================================================

// Sentinel errors for programmatic error handling.
var (
	ErrRecordNotFound       = errors.New("sqlite: record not found")
	ErrDBActionNotSupported = errors.New("sqlite: database action not supported")
)
