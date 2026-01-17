package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/kamareee/chirpy-bootdev/internal/auth"
	"github.com/kamareee/chirpy-bootdev/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	fileServerHits atomic.Int32
	db             *database.Queries
	platform       string
	jwtSecret      string
	polkaKey       string
}

type User struct {
	ID          uuid.UUID `json:"id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Email       string    `json:"email"`
	IsChirpyRed bool      `json:"is_chirpy_red"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string    `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

func main() {
	const filepathRoot = "."
	const port = "8080"

	err := godotenv.Load(".env")
	if err != nil {
		log.Printf("No .env file found or error loading it: %v", err)
	}
	dbURL := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	jwtSecret := os.Getenv("JWT_SECRET")
	polkaKey := os.Getenv("POLKA_KEY")

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer dbConn.Close()

	dbQueries := database.New(dbConn)

	apiCfg := apiConfig{
		fileServerHits: atomic.Int32{},
		db:             dbQueries,
		platform:       platform,
		jwtSecret:      jwtSecret,
		polkaKey:       polkaKey,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(filepathRoot)))))
	mux.Handle("/app/assets/", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/assets/", http.FileServer(http.Dir("./assets")))))

	mux.HandleFunc("GET /api/healthz", handlerHealthz)
	//mux.HandleFunc("POST /api/validate_chirp", handlerChirpsValidate)

	// mux.HandleFunc("GET /api/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerAdminMetrics)
	mux.HandleFunc("POST /admin/reset", apiCfg.handlerReset)

	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefresh)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevoke)
	mux.HandleFunc("POST /api/users", apiCfg.handlerCreateUsers)
	mux.HandleFunc("POST /api/chirps", apiCfg.handlerCreateChirps)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerListChirps)
	mux.HandleFunc("GET /api/chirps/{chirpID}", apiCfg.handlerGetChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", apiCfg.handlerDeleteChirp)

	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUsers)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerPolkaWebhooks)

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: mux,
	}

	log.Printf("Serving files from %s on port %s", filepathRoot, port)
	log.Fatal(srv.ListenAndServe())
}

// Handler functions
func (cfg *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decoder parameters", err)
		return
	}

	user, err := cfg.db.GetUserByEmail(r.Context(), params.Email)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password", err)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password, user.HashedPassword)
	if err != nil || !match {
		respondWithError(w, http.StatusUnauthorized, "Invalid email or password", err)
		return
	}

	// Generate JWT token with 1 hour expiration
	token, err := auth.MakeJWT(user.ID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token", err)
		return
	}

	// Generate refresh token
	refreshToken, err := auth.MakeRefreshToken()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate refresh token", err)
		return
	}

	// Store refresh token in database with 60 days expiration
	_, err = cfg.db.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token:     refreshToken,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(60 * 24 * time.Hour),
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to store refresh token", err)
		return
	}

	type responseUser struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	type response struct {
		responseUser
		Token        string `json:"token"`
		RefreshToken string `json:"refresh_token"`
	}

	payload := response{
		responseUser: responseUser{
			ID:          user.ID,
			CreatedAt:   user.CreatedAt,
			UpdatedAt:   user.UpdatedAt,
			Email:       user.Email,
			IsChirpyRed: user.IsChirpyRed,
		},
		Token:        token,
		RefreshToken: refreshToken,
	}
	respondWithJSON(w, http.StatusOK, payload)
}

func (cfg *apiConfig) handlerRefresh(w http.ResponseWriter, r *http.Request) {
	// Extract refresh token from Authorization header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header", err)
		return
	}

	// Look up user ID from refresh token (validates token exists, not expired, not revoked)
	userID, err := cfg.db.GetUserIDFromRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid or expired refresh token", err)
		return
	}

	// Generate new JWT access token with 1 hour expiration
	accessToken, err := auth.MakeJWT(userID, cfg.jwtSecret, time.Hour)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate access token", err)
		return
	}

	type response struct {
		Token string `json:"token"`
	}

	respondWithJSON(w, http.StatusOK, response{Token: accessToken})
}

func (cfg *apiConfig) handlerRevoke(w http.ResponseWriter, r *http.Request) {
	// Extract refresh token from Authorization header
	refreshToken, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Missing or invalid authorization header", err)
		return
	}

	// Revoke the refresh token
	err = cfg.db.RevokeRefreshToken(r.Context(), refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to revoke token", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	chirpIDStr := r.PathValue("chirpID")
	chirpID, err := uuid.Parse(chirpIDStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp ID", err)
		return
	}

	chirp, err := cfg.db.GetChirp(r.Context(), chirpID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found", err)
		return
	}

	payload := Chirp{
		ID:        chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body:      chirp.Body,
		UserID:    chirp.UserID,
	}

	respondWithJSON(w, http.StatusOK, payload)
}

func (cfg *apiConfig) handlerListChirps(w http.ResponseWriter, r *http.Request) {
	var dbChirps []database.Chirp
	var err error

	authorIDStr := r.URL.Query().Get("author_id")
	if authorIDStr != "" {
		authorID, parseErr := uuid.Parse(authorIDStr)
		if parseErr != nil {
			respondWithError(w, http.StatusBadRequest, "Invalid author_id", parseErr)
			return
		}
		dbChirps, err = cfg.db.ListChirpsByAuthor(r.Context(), authorID)
	} else {
		dbChirps, err = cfg.db.ListChirps(r.Context())
	}

	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error. Couldn't list chirps", err)
		return
	}

	if r.URL.Query().Get("sort") == "desc" {
		sort.Slice(dbChirps, func(i, j int) bool {
			return dbChirps[i].CreatedAt.After(dbChirps[j].CreatedAt)
		})
	}

	var payload []Chirp
	for _, c := range dbChirps {
		payload = append(payload, Chirp{
			ID:        c.ID,
			CreatedAt: c.CreatedAt,
			UpdatedAt: c.UpdatedAt,
			Body:      c.Body,
			UserID:    c.UserID,
		})
	}
	respondWithJSON(w, http.StatusOK, payload)
}

func (cfg *apiConfig) handlerCreateUsers(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decoder parameters", err)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Something went wrong! Please try again later", err)
		return
	}

	user, err := cfg.db.CreateUser(r.Context(), database.CreateUserParams{
		Email:          params.Email,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error. Couldn't create user", err)
		return
	}

	type responseUser struct {
		ID          uuid.UUID `json:"id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Email       string    `json:"email"`
		IsChirpyRed bool      `json:"is_chirpy_red"`
	}

	payload := responseUser{
		ID:          user.ID,
		CreatedAt:   user.CreatedAt,
		UpdatedAt:   user.UpdatedAt,
		Email:       user.Email,
		IsChirpyRed: user.IsChirpyRed,
	}
	respondWithJSON(w, http.StatusCreated, payload)
}

func (cfg *apiConfig) handlerPolkaWebhooks(w http.ResponseWriter, r *http.Request) {
	type polkaData struct {
		UserId string `json:"user_id"`
	}

	type parameters struct {
		Event string    `json:"event"`
		Data  polkaData `json:"data"`
	}

	// Validate API key
	apiKey, _ := auth.GetAPIKey(r.Header)
	if apiKey != cfg.polkaKey {
		w.WriteHeader(http.StatusUnauthorized)
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't decode parameters", err)
		return
	}

	if params.Event != "user:created" {
		w.WriteHeader(http.StatusNoContent)
	}
	userID, err := uuid.Parse(params.Data.UserId)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid user ID", err)
		return
	}
	_, err = cfg.db.UpgradeUserToChirpyRed(r.Context(), userID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "User not found", err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (cfg *apiConfig) handlerAdminMetrics(w http.ResponseWriter, r *http.Request) {
	hits := cfg.fileServerHits.Load()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w,
		`<html>
            <body>
                <h1>Welcome, Chirpy Admin</h1>
                <p>Chirpy has been visited %d times!</p>
            </body>
         </html>`, hits)
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, r *http.Request) {
	cfg.fileServerHits.Store(0)
	if cfg.platform != "dev" {
		respondWithError(w, http.StatusForbidden, "Resetting metrics is only allowed in dev environment", nil)
		return
	}
	err := cfg.db.DeleteUsers(r.Context())
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Database error. Couldn't reset users", err)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Users table deleted successfully"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileServerHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
