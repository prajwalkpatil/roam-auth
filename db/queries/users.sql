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
INSERT INTO auth.passwords (id, encrypted_password)
VALUES ($1, $2)
RETURNING id;

-- name: GetUserPasswords :many
SELECT auth.passwords.id AS id, 
auth.users.email AS email, 
auth.passwords.encrypted_password AS password 
FROM auth.passwords
INNER JOIN auth.users
ON auth.users.id = auth.passwords.id
LIMIT $1 OFFSET $2;