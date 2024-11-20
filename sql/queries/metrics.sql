-- name: InsertMetrics :one
INSERT INTO metric_events (created_at, tenant_id, resource_id, event_type, data)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: BulkInsertMetrics :many
WITH input_data AS (
    SELECT
        UNNEST(@created_ats::TIMESTAMPTZ[]) AS created_at,
        UNNEST(@tenant_ids::UUID[]) AS tenant_id,
        UNNEST(@resource_ids::UUID[]) AS resource_id,
        UNNEST(CAST(@event_types::text[] AS metric_event_type[])) AS event_type,
        UNNEST(@datas::JSONB[]) AS data
)
INSERT INTO metric_events (created_at, tenant_id, resource_id, event_type, data)
SELECT
    input_data.created_at,
    input_data.tenant_id,
    input_data.resource_id,
    input_data.event_type,
    input_data.data
FROM input_data
RETURNING *;

-- name: GetMetrics :many
SELECT
    time_bucket(COALESCE(sqlc.narg('interval')::interval, '1 minute'), minute) as bucket,
    SUM(succeeded_count)::int as succeeded_count,
    SUM(failed_count)::int as failed_count
FROM
    metric_events_summary
WHERE
    tenant_id = @tenantId::uuid AND
    -- timestamptz makes this fast, apparently: 
    -- https://www.timescale.com/forum/t/very-slow-query-planning-time-in-postgresql/255/8
    minute > @createdAfter::timestamptz AND
    minute < @createdBefore::timestamptz
GROUP BY bucket
ORDER BY bucket;

SELECT
    time_bucket('1 hour', minute) as bucket,
    SUM(succeeded_count)::int as succeeded_count,
    SUM(failed_count)::int as failed_count
FROM
    metric_events_summary
WHERE
    tenant_id = '3719706a-b1b6-4b7b-a57e-0b246c25e3c3' AND
    resource_id = '7cd2e8cf-4362-4453-b522-5a2da5eaa1f6' AND
    -- timestamptz makes this fast, apparently: 
    -- https://www.timescale.com/forum/t/very-slow-query-planning-time-in-postgresql/255/8
    minute > NOW() - INTERVAL '12 hours' AND
    minute < NOW()
GROUP BY bucket
ORDER BY bucket;