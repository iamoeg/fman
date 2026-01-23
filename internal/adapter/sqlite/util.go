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
// Constants
// ============================================================================

// DBTimeFormat is the standard time format used for all timestamp columns in SQLite.
// Uses RFC3339 for ISO 8601 compliance and human readability.
const (
	DBTimeFormat = time.RFC3339
)

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
// Errors
// ============================================================================

// Sentinel error for unsupported database actions.
var (
	ErrDBActionNotSupported = errors.New("sqlite: database action not supported")
)
