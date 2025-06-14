-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (token, user_id, created_at, updated_at, expires_at, revoked_at)
VALUES (
    $1,
    $2,
    NOW(),
    NOW(),
    $3,
    null
);

-- name: GetRefreshToken :one
select * from refresh_tokens
where token = $1;

-- name: RevokeToken :exec
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token = $1;