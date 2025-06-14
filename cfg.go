package main

import (
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries      *database.Queries
	Platform       string
	Secret         string
}

func (cfg *apiConfig) mwMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) metrics(w http.ResponseWriter, r *http.Request) {
	count := int(cfg.fileserverHits.Load())
	hits := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, count)
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(hits))
}

func (cfg *apiConfig) ressetmetrics(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
	err := cfg.dbQueries.ResetUsers(r.Context())
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request) {
	type cred struct {
		Password string
		Email    string
	}

	decoder := json.NewDecoder(r.Body)
	creds := cred{}
	err := decoder.Decode(&creds)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	hash, err := auth.HashPassword(creds.Password)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	params := database.CreateUserParams{
		Email:          creds.Email,
		HashedPassword: hash,
	}

	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), params)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
	}

	jsr, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 201, jsr)
}

func (cfg *apiConfig) Login(w http.ResponseWriter, r *http.Request) {
	type cred struct {
		Password string
		Email    string
	}

	decoder := json.NewDecoder(r.Body)
	creds := cred{}
	err := decoder.Decode(&creds)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	dbUser, err := cfg.dbQueries.GetUserByEmail(r.Context(), creds.Email)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	err = auth.CheckPasswordHash(dbUser.HashedPassword, creds.Password)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
	}

	token, err := auth.MakeJWT(dbUser.ID, cfg.Secret)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	reftok, _ := auth.MakeRefreshTOken()
	refparams := database.CreateRefreshTokenParams{
		Token:     reftok,
		UserID:    dbUser.ID,
		ExpiresAt: time.Now().Add(time.Hour * 24 * 60),
	}

	cfg.dbQueries.CreateRefreshToken(r.Context(), refparams)

	user := User{
		ID:        dbUser.ID,
		CreatedAt: dbUser.CreatedAt,
		UpdatedAt: dbUser.UpdatedAt,
		Email:     dbUser.Email,
		Token:     token,
		Refresh:   reftok,
	}

	jsr, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 200, jsr)
}

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type cred struct {
		Body    string `json:"body"`
		User_id string `json:"user_id"`
	}

	decoder := json.NewDecoder(r.Body)
	params := cred{}
	err := decoder.Decode(&params)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	var par database.CreateChirpParams
	par.Body = params.Body
	par.UserID = userUUID

	dbChirp, err := cfg.dbQueries.CreateChirp(r.Context(), par)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	chirp := mapChirp(dbChirp)

	jsr, err := json.Marshal(chirp)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 201, jsr)
}

func (cfg *apiConfig) GetChirps(w http.ResponseWriter, r *http.Request) {
	dbChirps, err := cfg.dbQueries.GetChirps(r.Context())
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}
	chirps := []Chirp{}
	for _, c := range dbChirps {
		chirps = append(chirps, mapChirp(c))
	}
	sort.Slice(chirps, func(i, j int) bool {
		return chirps[i].CreatedAt.Before(chirps[j].CreatedAt)
	})
	jsr, err := json.Marshal(chirps)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 200, jsr)
}

func (cfg *apiConfig) GetChirpByID(w http.ResponseWriter, r *http.Request) {
	idstr := r.PathValue("chirpid")
	chirpID, err := uuid.Parse(idstr)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	dbChirp, err := cfg.dbQueries.GetChirp(r.Context(), chirpID)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}
	chirp := mapChirp(dbChirp)

	jsr, err := json.Marshal(chirp)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 200, jsr)
}

func (cfg *apiConfig) Refresh(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	session, err := cfg.dbQueries.GetRefreshToken(r.Context(), token)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	if session.ExpiresAt.Before(time.Now()) {
		res := `{"error":"session expired"}`
		formJsonResponse(w, 401, res)
		return
	}

	if session.RevokedAt.Valid {
		res := `{"error":"session revoked"}`
		formJsonResponse(w, 401, res)
		return
	} else {
		jwt, err := auth.MakeJWT(session.UserID, cfg.Secret)
		if err != nil {
			res := fmt.Sprintf(`{"error":"%v"}`, err)
			formJsonResponse(w, 500, res)
			return
		}

		res := fmt.Sprintf(`{"token":"%v"}`, jwt)
		formJsonResponse(w, 200, res)
	}
}

func (cfg *apiConfig) Revoke(w http.ResponseWriter, r *http.Request) {
	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	err = cfg.dbQueries.RevokeToken(r.Context(), token)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) UpdatePassword(w http.ResponseWriter, r *http.Request) {
	type cred struct {
		Password string
		Email    string
	}

	decoder := json.NewDecoder(r.Body)
	creds := cred{}
	err := decoder.Decode(&creds)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	hash, err := auth.HashPassword(creds.Password)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	userParam := database.UpdateUserParams{
		Email:          creds.Email,
		HashedPassword: hash,
		ID:             userUUID,
	}

	user, err := cfg.dbQueries.UpdateUser(r.Context(), userParam)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	jsr, err := json.Marshal(user)
	if err != nil {
		panic(err)
	}
	respondWithJson(w, 200, jsr)
}

func (cfg *apiConfig) DeleteChirp(w http.ResponseWriter, r *http.Request) {
	idstr := r.PathValue("chirpid")
	chirpID, err := uuid.Parse(idstr)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}

	userUUID, err := auth.ValidateJWT(token, cfg.Secret)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 401, res)
		return
	}
}
