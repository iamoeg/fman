-- name: GetPayrollResult :one
SELECT *
FROM payroll_result
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: GetPayrollResultIncludingDeleted :one
SELECT *
FROM payroll_result
WHERE id = ?;

-- name: ListPayrollResults :many
SELECT *
FROM payroll_result
WHERE deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListPayrollResultsIncludingDeleted :many
SELECT *
FROM payroll_result
ORDER BY created_at DESC;

-- name: ListPayrollResultsByPayrollPeriod :many
SELECT *
FROM payroll_result
WHERE
    payroll_period_id = ?
    AND deleted_at IS NULL
ORDER BY employee_id;

-- name: ListPayrollResultsByPayrollPeriodIncludingDeleted :many
SELECT *
FROM payroll_result
WHERE payroll_period_id = ?
ORDER BY employee_id;

-- name: ListPayrollResultsByEmployee :many
SELECT *
FROM payroll_result
WHERE
    employee_id = ?
    AND deleted_at IS NULL
ORDER BY created_at DESC;

-- name: ListPayrollResultsByEmployeeIncludingDeleted :many
SELECT *
FROM payroll_result
WHERE employee_id = ?
ORDER BY created_at DESC;

-- name: CreatePayrollResult :one
INSERT INTO payroll_result(
    id,
    payroll_period_id,
    employee_id,
    compensation_package_id,
    currency,
    base_salary_cents,
    seniority_bonus_cents,
    gross_salary_cents,
    total_other_bonus_cents,
    gross_salary_grand_total_cents,
    total_exemptions_cents,
    taxable_gross_salary_cents,
    social_allowance_employee_contrib_cents,
    social_allowance_employer_contrib_cents,
    job_loss_compensation_employee_contrib_cents,
    job_loss_compensation_employer_contrib_cents,
    training_tax_employer_contrib_cents,
    family_benefits_employer_contrib_cents,
    total_cnss_employee_contrib_cents,
    total_cnss_employer_contrib_cents,
    amo_employee_contrib_cents,
    amo_employer_contrib_cents,
    taxable_net_salary_cents,
    income_tax_cents,
    rounding_amount_cents,
    net_to_pay_cents,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: DeletePayrollResult :exec
UPDATE payroll_result
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: RestorePayrollResult :one
UPDATE payroll_result
SET
    updated_at = ?,
    deleted_at = NULL
WHERE
    id = ?
    AND deleted_at IS NOT NULL
RETURNING *;

-- name: HardDeletePayrollResult :exec
DELETE FROM payroll_result
WHERE id = ?;

-- name: CountPayrollResultsUsingCompensationPackage :one
SELECT COUNT(*)
FROM payroll_result
WHERE
    compensation_package_id = ?
    AND deleted_at IS NULL;
