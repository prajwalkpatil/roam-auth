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