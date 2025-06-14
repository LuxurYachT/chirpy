package main

import (
	"chirpy/internal/database"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Println(err)
	}

	birdcfg := apiConfig{}
	birdcfg.Platform = os.Getenv("PLATFORM")
	birdcfg.Secret = os.Getenv("SECRET")
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
	birdmux.HandleFunc("POST /api/login", birdcfg.Login)
	birdmux.HandleFunc("POST /api/refresh", birdcfg.Refresh)
	birdmux.HandleFunc("POST /api/revoke", birdcfg.Revoke)
	birdmux.HandleFunc("PUT /api/users", birdcfg.UpdatePassword)
	birdmux.HandleFunc("DELETE /api/chirps/{chirpid}", birdcfg.DeleteChirp)

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

func Validate_chirp(r *http.Request) (bool, error) {
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
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write([]byte(resp))
}

func respondWithJson(w http.ResponseWriter, status int, resp []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(resp)
}

func StripProfane(content string) string {
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

func mapChirp(c database.Chirp) Chirp {
	chirp := Chirp{
		ID:        c.ID,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
		Body:      c.Body,
		User_id:   c.UserID,
	}
	return chirp
}
