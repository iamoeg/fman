-- name: GetEmployee :one
SELECT *
FROM employee
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: GetEmployeeIncludingDeleted :one
SELECT *
FROM employee
WHERE id = ?;

-- name: GetEmployeeByOrgAndSerialNum :one
SELECT *
FROM employee
WHERE
    org_id = ?
    AND serial_num = ?;

-- name: ListEmployees :many
SELECT *
FROM employee
WHERE deleted_at IS NULL
ORDER BY org_id, serial_num;

-- name: ListEmployeesIncludingDeleted :many
SELECT *
FROM employee
ORDER BY org_id, serial_num;

-- name: ListEmployeesByOrganization :many
SELECT *
FROM employee
WHERE
    org_id = ?
    AND deleted_at IS NULL
ORDER BY serial_num;

-- name: ListEmployeesByOrganizationIncludingDeleted :many
SELECT *
FROM employee
WHERE org_id = ?
ORDER BY serial_num;

-- name: CreateEmployee :one
INSERT INTO employee(
    id,
    org_id,
    serial_num,
    full_name,
    display_name,
    address,
    email_address,
    phone_number,
    birth_date,
    gender,
    marital_status,
    num_dependents,
    num_kids,
    cin_num,
    cnss_num,
    hire_date,
    position,
    compensation_package_id,
    bank_rib,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
    ?
) RETURNING *;

-- name: UpdateEmployee :one
UPDATE employee
SET
    full_name = ?,
    display_name = ?,
    address = ?,
    email_address = ?,
    phone_number = ?,
    birth_date = ?,
    gender = ?,
    marital_status = ?,
    num_dependents = ?,
    num_kids = ?,
    cin_num = ?,
    cnss_num = ?,
    hire_date = ?,
    position = ?,
    compensation_package_id = ?,
    bank_rib = ?,
    updated_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteEmployee :one
UPDATE employee
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: RestoreEmployee :one
UPDATE employee
SET
    updated_at = ?,
    deleted_at = NULL
WHERE
    id = ?
    AND deleted_at IS NOT NULL
RETURNING *;

-- name: HardDeleteEmployee :exec
DELETE FROM employee
WHERE id = ?;

-- name: CountEmployeesUsingCompensationPackage :one
SELECT COUNT(*)
FROM employee
WHERE
    compensation_package_id = ?
    AND deleted_at IS NULL;

-- name: GetNextSerialNumber :one
SELECT COALESCE(MAX(serial_num), 0) + 1
FROM employee
WHERE org_id = ?;
