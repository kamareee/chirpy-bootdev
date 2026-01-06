-- name: ListChirps :many
SELECT * FROM chirps
ORDER BY created_at;