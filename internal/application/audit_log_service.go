package application

import (
	"context"
	"fmt"
	"time"

	sqlite "github.com/iamoeg/bootdev-capstone/internal/adapter/sqlite"
)

// AuditLog is the application-layer representation of an audit log entry.
type AuditLog struct {
	ID        string
	TableName string
	RecordID  string
	Action    string
	Before    string
	After     string
	Timestamp time.Time
}

type auditLogReader interface {
	FindRecent(ctx context.Context, limit int) ([]*sqlite.AuditLog, error)
}

// AuditLogService provides read-only access to audit log entries.
type AuditLogService struct {
	repo auditLogReader
}

// NewAuditLogService creates a new AuditLogService with the given repository.
func NewAuditLogService(repo auditLogReader) *AuditLogService {
	return &AuditLogService{repo: repo}
}

// FindRecent returns the most recent audit log entries (up to limit).
func (s *AuditLogService) FindRecent(ctx context.Context, limit int) ([]*AuditLog, error) {
	rows, err := s.repo.FindRecent(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to load audit logs: %w", err)
	}
	logs := make([]*AuditLog, len(rows))
	for i, r := range rows {
		logs[i] = &AuditLog{
			ID:        r.ID,
			TableName: r.TableName,
			RecordID:  r.RecordID,
			Action:    r.Action,
			Before:    r.Before,
			After:     r.After,
			Timestamp: r.Timestamp,
		}
	}
	return logs, nil
}
