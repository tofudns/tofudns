-- User Queries
-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (
    email
) VALUES (
    $1
) RETURNING *;

-- Records Queries
-- name: GetRecordByID :one
SELECT * FROM coredns_records
WHERE id = $1 AND zone = $2 AND user_id = $3;

-- name: ListRecordsByZone :many
SELECT * FROM coredns_records
WHERE zone = $1 AND user_id = $2
ORDER BY name, record_type;

-- name: ListRecordsByName :many
SELECT * FROM coredns_records
WHERE zone = $1 AND name = $2 AND user_id = $3
ORDER BY id;

-- name: ListRecordsByType :many
SELECT * FROM coredns_records
WHERE zone = $1 AND record_type = $2 AND user_id = $3
ORDER BY name;

-- name: CreateRecord :one
INSERT INTO coredns_records (
    user_id,
    zone,
    name,
    ttl,
    content,
    record_type
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateRecord :one
UPDATE coredns_records
SET 
    name = $2,
    ttl = $3,
    content = $4,
    record_type = $5
WHERE id = $1 AND zone = $6 AND user_id = $7
RETURNING *;

-- name: DeleteRecord :exec
DELETE FROM coredns_records
WHERE id = $1 AND zone = $2 AND user_id = $3;

-- name: ListRecords :many
SELECT * FROM coredns_records
WHERE zone = $1 AND user_id = $2
ORDER BY name, record_type;

-- name: ListZones :many
SELECT DISTINCT zone 
FROM coredns_records 
WHERE user_id = $1
ORDER BY zone;

-- OTP Authentication Queries

-- name: CreateOTP :one
INSERT INTO otp_codes (
    email,
    code,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetLatestOTPByEmail :one
SELECT * FROM otp_codes
WHERE email = $1 AND consumed_at IS NULL AND expires_at > NOW()
ORDER BY created_at DESC
LIMIT 1;

-- name: ValidateAndConsumeOTP :one
UPDATE otp_codes
SET consumed_at = NOW()
WHERE id = (
    SELECT otp_codes.id FROM otp_codes
    WHERE otp_codes.email = $1 AND otp_codes.code = $2 AND otp_codes.consumed_at IS NULL AND otp_codes.expires_at > NOW()
    ORDER BY otp_codes.created_at DESC
    LIMIT 1
)
RETURNING *;
