-- name: BulkInsertLogs :copyfrom
INSERT INTO logs (created_at, tenant_id, resource_id, log)
VALUES (
    $1, $2, $3, $4
);

-- name: ListLogs :many
SELECT * 
FROM logs 
WHERE created_at >= $1 AND created_at <= $2 AND tenant_id = $3 AND resource_id = $4
ORDER BY created_at DESC;