-- name: ListChirps :many
SELECT * FROM chirps
ORDER BY created_at;

-- name: ListChirpsByAuthor :many
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at;