package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"sync/atomic"
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

func resporespondWithError(w http.ResponseWriter, code int, msg string) {
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

type parameters struct {
	Body string `json:"body"`
}

type errorResp struct {
	Err string `json:"error"`
}

type response struct {
	CleanedBody string `json:"cleaned_body"`
}

func main() {
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

	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {

		decoder := json.NewDecoder(r.Body)
		params := parameters{}
		err := decoder.Decode(&params)
		if err != nil {
			resporespondWithError(w, 500, "Something went wrong")
			return
		}
		if len(params.Body) > 140 {
			resporespondWithError(w, 400, "Chirp is too long")
			return
		}
		resp := response{
			CleanedBody: cleaningText(params.Body), 
		}
		respondWithJSON(w, 200, resp)
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