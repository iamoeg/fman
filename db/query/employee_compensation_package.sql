-- name: GetEmployeeCompensationPackage :one
SELECT *
FROM employee_compensation_package
WHERE id = ?;

-- name: ListEmployeeCompensationPackages :many
SELECT *
FROM employee_compensation_package;

-- name: CreateEmployeeCompensationPackage :one
INSERT INTO employee_compensation_package(
    id,
    currency,
    base_salary_cents,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?
) RETURNING *;

-- name: UpdateEmployeeCompensationPackage :one
UPDATE employee_compensation_package
SET
    currency = ?,
    base_salary_cents = ?,
    updated_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteEmployeeCompensationPackage :exec
UPDATE employee_compensation_package
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL;

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
