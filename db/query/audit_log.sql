-- name: ListAuditLogsRecent :many
SELECT *
FROM audit_log
ORDER BY timestamp DESC
LIMIT ?;

-- name: ListAuditLogsForRecord :many
SELECT *
FROM audit_log
WHERE
    table_name = ?
    AND record_id = ?
ORDER BY timestamp DESC;

-- name: ListAuditLogsByTable :many
SELECT *
FROM audit_log
WHERE table_name = ?
ORDER BY timestamp DESC
LIMIT ?;

-- name: ListAuditLogsByAction :many
SELECT *
FROM audit_log
WHERE action = ?
ORDER BY timestamp DESC
LIMIT ?;

-- name: CreateAuditLog :exec
INSERT INTO audit_log(
    id,
    table_name,
    record_id,
    action,
    before,
    after,
    timestamp
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
);
