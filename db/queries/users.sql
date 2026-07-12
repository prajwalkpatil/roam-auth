-- name: GetUsers :many
SELECT public.users.id AS id, name, email, created_at 
FROM public.users
INNER JOIN auth.users ON public.users.id = auth.users.id
LIMIT $1 OFFSET $2;