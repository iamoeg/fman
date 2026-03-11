-- name: GetEmployeeCompensationPackage :one
SELECT *
FROM employee_compensation_package
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: GetEmployeeCompensationPackageIncludingDeleted :one
SELECT *
FROM employee_compensation_package
WHERE id = ?;

-- name: ListEmployeeCompensationPackagesByOrg :many
SELECT *
FROM employee_compensation_package
WHERE org_id = ? AND deleted_at IS NULL;

-- name: ListEmployeeCompensationPackagesByOrgIncludingDeleted :many
SELECT *
FROM employee_compensation_package
WHERE org_id = ?;

-- name: CreateEmployeeCompensationPackage :one
INSERT INTO employee_compensation_package(
    id,
    org_id,
    name,
    currency,
    base_salary_cents,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: UpdateEmployeeCompensationPackage :one
UPDATE employee_compensation_package
SET
    name = ?,
    currency = ?,
    base_salary_cents = ?,
    updated_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteEmployeeCompensationPackage :one
UPDATE employee_compensation_package
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: RestoreEmployeeCompensationPackage :one
UPDATE employee_compensation_package
SET
    updated_at = ?,
    deleted_at = NULL
WHERE
    id = ?
    AND deleted_at IS NOT NULL
RETURNING *;

-- name: HardDeleteEmployeeCompensationPackage :exec
DELETE FROM employee_compensation_package
WHERE id = ?;
