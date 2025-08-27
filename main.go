package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"github.com/mrtuandao/chirpy/internal/auth"
	"github.com/mrtuandao/chirpy/internal/database"
)

type apiConfig struct {
	fileserverHits atomic.Int32
}

func (cfg *apiConfig) middlewareMetricInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) getMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/hmtl")
	content := fmt.Sprintf(`<html>
  <body>
    <h1>Welcome, Chirpy Admin</h1>
    <p>Chirpy has been visited %d times!</p>
  </body>
</html>`, cfg.fileserverHits.Load())
	w.Write([]byte(content))
}

func (cfg *apiConfig) resetHandler(w http.ResponseWriter, r *http.Request) {
	cfg.fileserverHits.Store(0)
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	resp := errorResp{
		Err: msg,
	}
	data, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(data)
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	w.WriteHeader(code)
	data, err := json.Marshal(payload)
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(data)	
}

func cleaningText(t string) string {
	badWords := []string{"kerfuffle", "sharbert", "fornax"}
	words := strings.Split(t, " ")
	for i, word := range words {
		if slices.Contains(badWords, strings.ToLower(word)) {
			words[i] = "****"
		}
	}

	return strings.Join(words, " ")
}

type createChirpReq struct {
	Body string `json:"body"`
	UserID string `json:"user_id"`
}

type createUserReq struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type loginReq struct {
	Email string `json:"email"`
	Password string `json:"password"`
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}

type Chirp struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Body      string `json:"body"`
	UserID    uuid.UUID `json:"user_id"`
}

type errorResp struct {
	Err string `json:"error"`
}

func main() {
	godotenv.Load()
	dbURL := os.Getenv("DB_URL")
	db, _ := sql.Open("postgres", dbURL)
	dbQueries := database.New(db)

	mux := http.NewServeMux()
	cfg := &apiConfig{}
	fmt.Println(cfg.fileserverHits.Load())
	mux.Handle("/app/", http.StripPrefix("/app", cfg.middlewareMetricInc(http.FileServer(http.Dir(".")))))
	mux.HandleFunc("/admin/metrics", cfg.getMetrics)
	mux.HandleFunc("/admin/reset", cfg.resetHandler)
	mux.HandleFunc("/api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	})

	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		items, err := dbQueries.GetAllChirps(r.Context())
		if err != nil {
			respondWithError(w, 500, fmt.Sprintf("Can not get all chirps: %v", err))
			return 			
		}
		chirps := []Chirp{}
		for _, chirp := range items {
			chirps = append(chirps, Chirp{
				ID: chirp.ID, 
				CreatedAt: chirp.CreatedAt.Time, 
				UpdatedAt: chirp.UpdatedAt.Time, 
				Body: chirp.Body.String, 
				UserID: chirp.UserID.UUID,
			})
		}

		respondWithJSON(w, 200, chirps)
	})

	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		chirpID := r.PathValue("chirpID")
		fmt.Printf("chirpID: %v\n", chirpID)
		idDB, err := uuid.Parse(chirpID)
		fmt.Printf("idDB: %v\n", idDB)
		if err != nil {
			respondWithError(w, 404, fmt.Sprintf("Can not create chirp: %v", err))
			return 
		}
		chirpInDB, err := dbQueries.GetChirpByID(r.Context(), idDB)
		if err != nil {
			respondWithError(w, 404, fmt.Sprintf("Can not create chirp: %v", err))
			return 
		}
		// if chirpInDB.ID == nil {
		// 	respondWithError(w, 404, "Not found")
		// }
		chirp := Chirp{
			ID: chirpInDB.ID,
			CreatedAt: chirpInDB.CreatedAt.Time,
			UpdatedAt: chirpInDB.UpdatedAt.Time,
			Body: chirpInDB.Body.String,
			UserID: chirpInDB.UserID.UUID,
		}
		respondWithJSON(w, 200, chirp)
	})

	mux.HandleFunc("POST /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		chirp_sent := createChirpReq{}
		err := decoder.Decode(&chirp_sent)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		if len(chirp_sent.Body) > 140 {
			respondWithError(w, 400, "Chirp is too long")
			return
		}
		user_id, err := uuid.Parse(chirp_sent.UserID)
		if err != nil {
			respondWithError(w, 500, fmt.Sprintf("Can not create chirp: %v", err))
			return 
		}
		db_chirp, err := dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{
			Body: sql.NullString{
				String: chirp_sent.Body, 
				Valid: true, 
			}, 
			UserID: uuid.NullUUID{
				UUID: user_id, 
				Valid: true, 
			},
		})

		if err != nil {
			respondWithError(w, 500, fmt.Sprintf("Can not create chirp: %v", err))
			return 
		}
		
		chirp := Chirp{
			ID: db_chirp.ID, 
			CreatedAt: db_chirp.CreatedAt.Time, 
			UpdatedAt: db_chirp.UpdatedAt.Time, 
			Body: db_chirp.Body.String, 
			UserID: db_chirp.UserID.UUID,
		}
		respondWithJSON(w, 201, chirp)
	})

	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		create_user_req := createUserReq{}
		err := decoder.Decode(&create_user_req)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
		}
		hashed_password, err := auth.HashPassword(create_user_req.Password)
		if err != nil {
			respondWithError(w, 500, fmt.Sprintf("Can not create user %v", err))
			return 
		}
		db_user, err := dbQueries.CreateUser(
			r.Context(), 
			database.CreateUserParams{
				Email: sql.NullString{
					String: create_user_req.Email,
					Valid: true,
				},
				HashedPassword: hashed_password,
			},
		)
		if err != nil {
			respondWithError(w, 500, fmt.Sprintf("Can not create user %v", err))
			return 
		}
		user := User{
			ID: db_user.ID, 
			CreatedAt: db_user.CreatedAt.Time, 
			UpdatedAt: db_user.UpdatedAt.Time,
			Email: db_user.Email.String,
		}

		respondWithJSON(w, 201, user)
	})

	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		decoder := json.NewDecoder(r.Body)
		login_req := loginReq{}
		err := decoder.Decode(&login_req)
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		user, err := dbQueries.GetUserByEmail(
			r.Context(), 
			sql.NullString{
				String: login_req.Email, 
				Valid: true,
		})
		if err != nil {
			respondWithError(w, 500, "Something went wrong")
			return
		}
		if err := auth.CheckPasswordHash(login_req.Password, user.HashedPassword); err != nil {
			respondWithError(w, 401, "")
			return 
		}
		respondWithJSON(w, 200, map[string]interface{}{
			"id": user.ID, 
			"created_at": user.CreatedAt.Time,
			"updated_at": user.UpdatedAt.Time,
			"email": user.Email.String, 
		})
	})

	mux.HandleFunc("POST /admin/reset", func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("PLATFORM") != "dev" {
			respondWithError(w, 403, "Go away kids")
			return 
		}
		db.Query("DELETE FROM users")
		w.WriteHeader(200)
	})

	server := http.Server{
		Addr: ":8080",
		Handler: mux,
	}
	err := server.ListenAndServe()
	if err != nil {
		fmt.Println(err)
	}

}
