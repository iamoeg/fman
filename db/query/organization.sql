-- name: GetOrganization :one
SELECT *
FROM organization
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: GetOrganizationIncludingDeleted :one
SELECT *
FROM organization
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: ListOrganizations :many
SELECT *
FROM organization
WHERE
    id = ?
    AND deleted_at IS NULL
ORDER BY name;

-- name: ListOrganizationsIncludingDeleted :many
SELECT *
FROM organization
ORDER BY name;

-- name: CreateOrganization :one
INSERT INTO organization(
    id,
    name,
    address,
    activity,
    legal_form,
    ice_num,
    if_num,
    rc_num,
    cnss_num,
    bank_rib,
    created_at,
    updated_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
) RETURNING *;

-- name: UpdateOrganization :one
UPDATE organization
SET
    name = ?,
    address = ?,
    activity = ?,
    legal_form = ?,
    ice_num = ?,
    if_num = ?,
    rc_num = ?,
    cnss_num = ?,
    bank_rib = ?,
    updated_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL
RETURNING *;

-- name: DeleteOrganization :exec
UPDATE organization
SET
    updated_at = ?,
    deleted_at = ?
WHERE
    id = ?
    AND deleted_at IS NULL;

-- name: RestoreOrganization :one
UPDATE organization
SET
    updated_at = ?,
    deleted_at = NULL
WHERE
    id = ?
    AND deleted_at IS NOT NULL
RETURNING *;

-- name: HardDeleteOrganization :exec
DELETE FROM organization
WHERE id = ?;
