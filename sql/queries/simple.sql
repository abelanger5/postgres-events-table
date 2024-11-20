-- name: InsertSimple :one
INSERT INTO simple_events (created_at, tenant_id, resource_id, data)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: BulkInsertSimple :many
WITH input_data AS (
    SELECT
        UNNEST(@created_ats::TIMESTAMPTZ[]) AS created_at,
        UNNEST(@tenant_ids::UUID[]) AS tenant_id,
        UNNEST(@resource_ids::UUID[]) AS resource_id,
        UNNEST(@datas::JSONB[]) AS data
)
INSERT INTO simple_events (created_at, tenant_id, resource_id, data)
SELECT
    input_data.created_at,
    input_data.tenant_id,
    input_data.resource_id,
    input_data.data
FROM input_data
RETURNING *;

-- name: ListEventsByResourceID :many
SELECT * 
FROM simple_events 
WHERE resource_id = $1 AND tenant_id = $2 
ORDER BY id ASC;

-- name: GetEventByID :one
SELECT *
FROM simple_events
WHERE id = $1;