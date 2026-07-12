-- name: GetUsers :many
SELECT * FROM public.users 
ORDER BY id ASC 
LIMIT $1 OFFSET $2;