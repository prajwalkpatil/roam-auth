-- name: GetUsers :many
SELECT public.users.id AS id, name, email, created_at 
FROM public.users
INNER JOIN auth.users 
ON public.users.id = auth.users.id
LIMIT $1 OFFSET $2;

-- name: CreateAuthUser :one
INSERT INTO auth.users (email) 
VALUES ($1)
RETURNING id; 

-- name: CreatePublicUser :one
INSERT INTO public.users (id, name)
VALUES ($1, $2)
RETURNING id; 

-- name: CreateUserPassword :one
INSERT INTO auth.passwords (id, hashed_password)
VALUES ($1, $2)
RETURNING id;

-- name: GetUserPassword :one
SELECT hashed_password
FROM auth.passwords
WHERE id = $1;

-- name: GetUserPasswordFromEmail :one
SELECT auth.users.id as id, auth.users.email as email, hashed_password
FROM auth.users
INNER JOIN auth.passwords 
ON auth.users.id = auth.passwords.id
WHERE auth.users.email = $1;

-- name: AddRefreshToken :one
INSERT INTO auth.tokens (id, refresh_token, expires_at)
VALUES ($1, $2, $3)
RETURNING id, refresh_token, expires_at;

-- name: DeleteRefreshTokens :execrows
DELETE FROM auth.tokens 
WHERE id = $1;

-- name: DeleteRefreshToken :execrows
DELETE FROM auth.tokens
WHERE id = $1 AND refresh_token = $2;

-- name: GetRefreshToken :one
SELECT id, refresh_token, expires_at
FROM auth.tokens
WHERE id = $1 AND refresh_token = $2;

-- name: ReplaceRefreshToken :execrows
UPDATE auth.tokens
SET refresh_token = sqlc.arg(new_refresh_token), expires_at = sqlc.arg(expires_at)
WHERE refresh_token = sqlc.arg(old_refresh_token)
RETURNING id, refresh_token, expires_at;

-- name: GetUserFromRefreshToken :many
SELECT auth.users.id as id, auth.users.email as email 
FROM auth.tokens
INNER JOIN auth.users
ON auth.users.id = auth.tokens.id
WHERE auth.tokens.refresh_token = $1;