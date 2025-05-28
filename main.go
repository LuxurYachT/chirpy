package main

import _ "github.com/lib/pq"
import (
	"net/http"
	"sync/atomic"
	"encoding/json"
	"fmt"
	"strings"
	"database/sql"
	"github.com/joho/godotenv"
	"chirpy/internal/database"
	"os"
	"github.com/google/uuid"
	"sort"
)

type apiConfig struct {
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	Platform string
}

func (cfg *apiConfig) mwMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func (w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w,r)
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
		Email string
	}

	decoder := json.NewDecoder(r.Body)
	params := cred{}
	err := decoder.Decode(&params)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
		return
	}

	dbUser, err := cfg.dbQueries.CreateUser(r.Context(), params.Email)
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

func (cfg *apiConfig) createChirp(w http.ResponseWriter, r *http.Request) {
	type cred struct {
		Body string `json:"body"`
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

	userUUID, err := uuid.Parse(params.User_id)
	if err != nil {
		res := fmt.Sprintf(`{"error":"%v"}`, err)
		formJsonResponse(w, 500, res)
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

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}

	birdcfg := apiConfig{}
	birdcfg.Platform = os.Getenv("PLATFORM")
	birdcfg.dbQueries = database.New(db)
	var birdmux = http.NewServeMux()
	birdmux.Handle("/app/", http.StripPrefix("/app", birdcfg.mwMetricsInc(http.FileServer(http.Dir(".")))))
	birdmux.HandleFunc("GET /admin/healthz", readiness)
	birdmux.HandleFunc("GET /admin/metrics", birdcfg.metrics)
	birdmux.HandleFunc("POST /admin/reset", birdcfg.ressetmetrics)
	birdmux.HandleFunc("POST /api/users", birdcfg.createUser)
	birdmux.HandleFunc("POST /api/chirps", birdcfg.createChirp)
	birdmux.HandleFunc("GET /api/chirps", birdcfg.GetChirps)
	birdmux.HandleFunc("GET /api/chirps/{chirpid}", birdcfg.GetChirpByID)

	var birdserver http.Server
	birdserver.Addr = ":8080"
	birdserver.Handler = birdmux
	birdserver.ListenAndServe()
}

func readiness(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func validate_chirp(r *http.Request) (bool, error) {
	type chirp struct {
		Body string
	}

	decoder := json.NewDecoder(r.Body)
	ch := chirp{}
	err := decoder.Decode(&ch)
	if err != nil {
		return false, err
	}
	if len(ch.Body) <= 140 {
		return true, nil
	} else {
		return false, fmt.Errorf("Chirp too long")
	}
}

func formJsonResponse(w http.ResponseWriter, status int, resp string) {
		raw := json.RawMessage(resp)
		jsr, err := json.Marshal(raw)
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		w.Write(jsr)
}

func respondWithJson(w http.ResponseWriter, status int, resp []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp)
}

func stripProfane(content string) string {
	words := strings.Split(content, " ")
	profane := []string{"kerfuffle", "sharbert", "fornax"}
	for i, word := range words {
		input := strings.ToLower(word)
		for _, cen := range profane {
			if input == cen {
				words[i] = "****"
			}
		}
	}
	clean := strings.Join(words, " ")
	return clean
}

func mapChirp(c database.Chirp) (Chirp) {
	chirp := Chirp{
		ID:        c.ID,
		CreatedAt: c.CreatedAt, 
		UpdatedAt: c.UpdatedAt,
		Body:	   c.Body,
		User_id:   c.UserID,
	}
	return chirp
}