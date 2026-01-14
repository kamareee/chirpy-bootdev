package main

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/kamareee/chirpy-bootdev/internal/auth"
	"github.com/kamareee/chirpy-bootdev/internal/database"
)

func (cfg *apiConfig) handlerUpdateUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	accessToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header", err)
		return
	}

	userId, err := auth.ValidateJWT(accessToken, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid token", err)
		return
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't hash password", err)
		return
	}

	returnedUser, err := cfg.db.UpdateUserPasswordEmail(r.Context(), database.UpdateUserPasswordEmailParams{
		ID:             userId,
		HashedPassword: hashedPassword,
		Email:          params.Email,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error. Couldn't update user", err)
		return
	}

	type User struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Email     string    `json:"email"`
	}

	payload := User{
		ID:        returnedUser.ID,
		CreatedAt: returnedUser.CreatedAt,
		UpdatedAt: returnedUser.UpdatedAt,
		Email:     returnedUser.Email,
	}

	respondWithJSON(w, http.StatusOK, payload)
}
