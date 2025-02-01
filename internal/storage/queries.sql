-- name: GetRecordByID :one
SELECT * FROM coredns_records
WHERE id = $1 AND zone = $2;

-- name: ListRecordsByZone :many
SELECT * FROM coredns_records
WHERE zone = $1
ORDER BY name, record_type;

-- name: ListRecordsByName :many
SELECT * FROM coredns_records
WHERE zone = $1 AND name = $2
ORDER BY id;

-- name: ListRecordsByType :many
SELECT * FROM coredns_records
WHERE zone = $1 AND record_type = $2
ORDER BY name;

-- name: CreateRecord :one
INSERT INTO coredns_records (
    zone,
    name,
    ttl,
    content,
    record_type
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: UpdateRecord :one
UPDATE coredns_records
SET 
    name = $2,
    ttl = $3,
    content = $4,
    record_type = $5
WHERE id = $1 AND zone = $6
RETURNING *;

-- name: DeleteRecord :exec
DELETE FROM coredns_records
WHERE id = $1 AND zone = $2;

-- name: ListRecords :many
SELECT * FROM coredns_records
WHERE zone = $1
ORDER BY name, record_type;

-- name: ListZones :many
SELECT DISTINCT zone 
FROM coredns_records 
ORDER BY zone;
