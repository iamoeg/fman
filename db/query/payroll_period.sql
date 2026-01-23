-- name: GetPayrollPeriod :one
SELECT *
FROM payroll_period
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: GetPayrollPeriodIncludingDeleted :one
SELECT *
FROM payroll_period
WHERE id = ?;

-- name: GetPayrollPeriodByOrgYearMonth :one
SELECT *
FROM payroll_period
WHERE
    org_id = ?
    AND year = ?
    AND month = ?
    AND deleted_at IS NULL;

-- name: GetPayrollPeriodByOrgYearMonthIncludingDeleted :one
SELECT *
FROM payroll_period
WHERE
    org_id = ?
    AND year = ?
    AND month = ?;

-- name: ListPayrollPeriods :many
SELECT *
FROM payroll_period
WHERE deleted_at IS NULL
ORDER BY
    year DESC,
    month DESC;

-- name: ListPayrollPeriodsIncludingDeleted :many
SELECT *
FROM payroll_period
ORDER BY
    year DESC,
    month DESC;

-- name: ListPayrollPeriodsByOrganization :many
SELECT *
FROM payroll_period
WHERE
    org_id = ?
    AND deleted_at IS NULL
ORDER BY
    year DESC,
    month DESC;

-- name: ListPayrollPeriodsByOrganizationIncludingDeleted :many
SELECT *
FROM payroll_period
WHERE
    org_id = ?
ORDER BY
    year DESC,
    month DESC;

-- name: ListDraftPayrollPeriods :many
SELECT *
FROM payroll_period
WHERE
    status = 'DRAFT'
    AND deleted_at IS NULL
ORDER BY
    year DESC,
    month DESC;

-- name: ListDraftPayrollPeriodsIncludingDeleted :many
SELECT *
FROM payroll_period
WHERE status = 'DRAFT'
ORDER BY
    year DESC,
    month DESC;

-- name: CreatePayrollPeriod :one
INSERT INTO payroll_period(
    id,
    org_id,
    year,
    month,
    status,
    finalized_at,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: FinalizePayrollPeriod :one
UPDATE payroll_period
SET
    status = 'FINALIZED',
    finalized_at = ?,
    updated_at = ?
WHERE
    id = ?
    AND status = 'DRAFT'
    AND deleted_at IS NULL
RETURNING *;

-- name: UnfinalizePayrollPeriod :one
UPDATE payroll_period
SET
    status = 'DRAFT',
    finalized_at = NULL,
    updated_at = ?
WHERE
    id = ?
    AND status = 'FINALIZED'
    AND deleted_at IS NULL
RETURNING *;

-- name: DeletePayrollPeriod :one
UPDATE payroll_period
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: RestorePayrollPeriod :one
UPDATE payroll_period
SET
    updated_at = ?,
    deleted_at = NULL
WHERE
    id = ?
    AND deleted_at IS NOT NULL
RETURNING *;

-- name: HardDeletePayrollPeriod :exec
DELETE FROM payroll_period
WHERE id = ?;
