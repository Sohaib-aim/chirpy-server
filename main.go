package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/Sohaib-aim/chirpy-server/internal/auth"
	"github.com/Sohaib-aim/chirpy-server/internal/database"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func healthzhandler(w http.ResponseWriter, r *http.Request){
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}


type apiConfig struct{
	fileserverHits atomic.Int32
	dbQueries *database.Queries
	platform string
	token_secret string
	polka_key string
}	

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request){
		cfg.fileserverHits.Add(1)
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) countHits(w http.ResponseWriter, r *http.Request){
	hits := cfg.fileserverHits.Load()

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`<html>
	<body>
    	<h1>Welcome, Chirpy Admin</h1>
    	<p>Chirpy has been visited %d times!</p>
  	</body>
	</html>`, hits)))

}

func (cfg *apiConfig) resetHits(w http.ResponseWriter, r *http.Request){
	if cfg.platform != "dev"{
		w.WriteHeader(http.StatusForbidden)
		return	
	}

	err := cfg.dbQueries.DeleteAllUsers(r.Context())
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cfg.fileserverHits.Store(0)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0\n"))
}

func (cfg *apiConfig) chirpCreator(w http.ResponseWriter, r *http.Request){
	type parameters struct{
		Body string `json:"body"`
		UserID uuid.UUID `json:"user_id"`
	}

	type errorResponse struct{
		Error string `json:"error"`
	}

	type chirpResponse struct {
    ID        uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Body      string    `json:"body"`
    UserID    uuid.UUID `json:"user_id"`
	}

	token_value, err := auth.GetBearerToken(r.Header)
	if err != nil{
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	userID, err := auth.ValidateJWT(token_value, cfg.token_secret)
	if err != nil{
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	decoder := json.NewDecoder(r.Body)
	var params parameters
	if err := decoder.Decode(&params); err != nil{
		log.Printf("error decoding json.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errors := errorResponse{
			Error:"error decoding json",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	if len(params.Body) > 140{
		log.Printf("Chirp is too long.")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		errors := errorResponse{
			Error:"Chirp is too long",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	words := strings.Split(params.Body, " ")

	for i, word := range words{
		switch strings.ToLower(word){
		case "kerfuffle", "sharbert", "fornax":
			words[i] = "****"
		}
	}
	cleaned_res := strings.Join(words, " ")
	params.Body = cleaned_res

	chirp, err := cfg.dbQueries.CreateChirp(r.Context(), database.CreateChirpParams{Body: params.Body, UserID: userID})
	if err != nil{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		errors := errorResponse{
			Error: "error creating chirp",
		}

		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	chirpRes := chirpResponse{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	data, _ := json.Marshal(chirpRes)
	w.Write(data)
}

func (cfg *apiConfig) createUser(w http.ResponseWriter, r *http.Request){

	type parameters struct{
		Password string `json:"password"`
		Email string `json:"email"`
	}

	type errorResponse struct{
		Error string `json:"error"`
	}

	type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
	}

	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		errors := errorResponse{
			Error: "error decoding the user email",
		}

		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		errors := errorResponse{
			Error: "error hashing password",
		}

		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	user, err := cfg.dbQueries.CreateUser(r.Context(), database.CreateUserParams{Email:params.Email, HashedPassword: hashedPassword})
	if err != nil{
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		log.Print(err)
		errors := errorResponse{
			Error: "error creating user",
		}

		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	userRes := userResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	data, err := json.Marshal(userRes)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}

func (cfg *apiConfig) retrieveChirps(w http.ResponseWriter, r *http.Request){

	type chirpResponse struct {
    ID        uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Body      string    `json:"body"`
    UserID    uuid.UUID `json:"user_id"`
	}

	chirps, err := cfg.dbQueries.GetAllChirps(r.Context())

	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("error getting chirps"))
		return
	}

	responses := make([]chirpResponse, 0, len(chirps))

    for _, chirp := range chirps {
        responses = append(responses, chirpResponse{
            ID:        chirp.ID,
            CreatedAt: chirp.CreatedAt,
            UpdatedAt: chirp.UpdatedAt,
            Body:      chirp.Body,
            UserID:    chirp.UserID,
        })
    }

	res, _ := json.Marshal(responses)
	w.WriteHeader(200)
	w.Write(res)
}

func (cfg *apiConfig) getChirp(w http.ResponseWriter, r *http.Request){

	type chirpResponse struct {
    ID        uuid.UUID `json:"id"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    Body      string    `json:"body"`
    UserID    uuid.UUID `json:"user_id"`
	}


	id := r.PathValue("chirpId")
	parsed_id, err := uuid.Parse(id)
	if err != nil{
		log.Fatalf("invalid uuid format")
		return
	}
	chirp, err := cfg.dbQueries.GetOneChirp(r.Context(), parsed_id)
	if err != nil{
		w.WriteHeader(404)
		w.Write([]byte("no chirp with that Id"))
		return
	}

	chirpRes := chirpResponse{
		ID: chirp.ID,
		CreatedAt: chirp.CreatedAt,
		UpdatedAt: chirp.UpdatedAt,
		Body: chirp.Body,
		UserID: chirp.UserID,
	}

	resp, _ := json.Marshal(chirpRes)
	w.WriteHeader(200)
	w.Write(resp)
}

func (cfg *apiConfig) loginUser(w http.ResponseWriter, r *http.Request){

	type parameters struct{
		Password string `json:"password"`
		Email string `json:"email"`
	}

	type errorResponse struct{
		Error string `json:"error"`
	}

	type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	Token string        `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed bool `json:"is_chirpy_red"`
	}

	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil{
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(http.StatusInternalServerError)
		errors := errorResponse{
			Error : "error decoding the response...",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	user, err := cfg.dbQueries.GetUserByEmail(r.Context(), params.Email)
	if err != nil{
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(401)
		errors := errorResponse{
			Error : "Incorrect email",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	match, err := auth.CheckPasswordHash(params.Password,user.HashedPassword)

	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if match == false{
		w.Header().Set("Content-Type","application/json")
		w.WriteHeader(401)
		errors := errorResponse{
			Error : "Incorrect password",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return
	}

	tokenString, err := auth.MakeJWT(user.ID, cfg.token_secret, time.Hour)
	if err != nil{
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	refresh_token := auth.MakeRefreshToken()
	refreshExpiresAt := time.Now().Add(60 * 24 * time.Hour)
	now := time.Now()

	_, err = cfg.dbQueries.CreateRefreshToken(r.Context(), database.CreateRefreshTokenParams{
		Token: refresh_token,
		CreatedAt: now,
		UpdatedAt: now,
		UserID: user.ID,
		ExpiresAt: refreshExpiresAt,
	})

	userRes := userResponse{
		ID: user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Email: user.Email,
		Token: tokenString,
		RefreshToken: refresh_token,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	data, _ := json.Marshal(userRes)
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) refreshToken(w http.ResponseWriter, r *http.Request){

	type errorResponse struct{
		Error string `json:"error"`
	}

	type userResponse struct{
		Token string `json:"token"`
	}

	bearer_token, err := auth.GetBearerToken(r.Header)
	if err != nil{
		errors := errorResponse{
			Error: "error getting the refresh bearer token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}
	user, err := cfg.dbQueries.GetUserFromRefreshToken(r.Context(), bearer_token)
	if err != nil{
		errors := errorResponse{
			Error: "user with this refresh token does not exist",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}
	token, err := auth.MakeJWT(user.UserID, cfg.token_secret, time.Hour)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	tokenRes := userResponse{
		Token: token,
	}

	data, _ := json.Marshal(tokenRes)
	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(200)
	w.Write(data)

}

func (cfg *apiConfig) revokeRefreshToken(w http.ResponseWriter, r *http.Request){
	type errorResponse struct{
		Error string `json:"error"`
	}

	bearer_token, err := auth.GetBearerToken(r.Header)
	if err != nil{
		errors := errorResponse{
			Error: "error getting the refresh bearer token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	err = cfg.dbQueries.RevokeRefreshToken(r.Context(), bearer_token)
	if err != nil{
		errors := errorResponse{
			Error: "error revoking the refresh token",
		}
		data, _ := json.Marshal(errors)
		w.Write(data)
		return	
	}
	w.WriteHeader(204)
}

func (cfg *apiConfig) updateCredentials(w http.ResponseWriter, r *http.Request){

	type parameters struct{
		Password string `json:"password"`
		Email string `json:"email"`
	}

	type errorResponse struct{
		Error string `json:"error"`
	}

	type userResponse struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
	IsChirpyRed bool `json:"is_chirpy_red"`
	}

	bearer_token, err := auth.GetBearerToken(r.Header)
	if err != nil{
		errors := errorResponse{
			Error: "error getting the refresh bearer token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	userID, err := auth.ValidateJWT(bearer_token, cfg.token_secret)
	if err != nil{
		errors := errorResponse{
			Error: "error getting the refresh bearer token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	var params parameters

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil{
		errors := errorResponse{
			Error: "error decoding the response",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	hashed_passwd, err := auth.HashPassword(params.Password)
	if err != nil{
		errors := errorResponse{
			Error: "error hashing the password",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}
	
	user, err := cfg.dbQueries.UpdateUserCredentials(r.Context(), database.UpdateUserCredentialsParams{
		Email: params.Email, 
		HashedPassword: hashed_passwd,
		ID: userID,
	})
	if err != nil{
		errors := errorResponse{
			Error: "error updating the credentials",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	userRes := userResponse{
		ID: user.ID,
		Email: user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		IsChirpyRed: user.IsChirpyRed.Bool,
	}

	data, _ := json.Marshal(userRes)
	w.WriteHeader(200)
	w.Write(data)
}

func (cfg *apiConfig) chirpDelete(w http.ResponseWriter, r *http.Request){

	type errorResponse struct{
		Error string `json:"error"`
	}

	bearer_token, err := auth.GetBearerToken(r.Header)
	if err != nil{
		errors := errorResponse{
			Error: "error getting access token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	userID, err := auth.ValidateJWT(bearer_token, cfg.token_secret)
	if err != nil{
		errors := errorResponse{
			Error: "invalid access token",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(403)
		w.Write(data)
		return
	}
	
	chirpID := r.PathValue("chirpId")
	parsedID, err := uuid.Parse(chirpID)
	if err != nil {
    	errors := errorResponse{
        	Error: "invalid chirp ID",
    	}
    	data, _ := json.Marshal(errors)
    	w.WriteHeader(http.StatusBadRequest)
    	w.Write(data)
    	return
	}

	result, err := cfg.dbQueries.DeleteChirp(r.Context(), database.DeleteChirpParams{ID: parsedID, UserID: userID})
	if err != nil{
		errors := errorResponse{
			Error: "error deleting the chirp",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(403)
		w.Write(data)
		return
	}

	rows, err := result.RowsAffected()
	if err != nil{
		w.WriteHeader(403)
		return
	}

	if rows == 0{
		w.WriteHeader(403)
		return
	}

	w.WriteHeader(204)
}

func (cfg *apiConfig) updateChirpyRed(w http.ResponseWriter, r *http.Request){
	
	type errorResponse struct{
		Error string `json:"error"`
	}

	type data struct{
		UserID string `json:"user_id"`
	}

	type parameters struct{
		Event string `json:"event"`
		Data data `json:"data"`
	}

	apiKey, err := auth.GetApiKey(r.Header)
	if err != nil{
		errors := errorResponse{
			Error: "error getting the api key",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(data)
		return
	}

	if apiKey != cfg.polka_key{
		w.WriteHeader(401)
		return
	}

	var params parameters
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&params); err != nil{
		errors := errorResponse{
			Error: "error decoding the body",
		}
		data, _ := json.Marshal(errors)
		w.WriteHeader(http.StatusBadRequest)
		w.Write(data)
		return
	}

	if params.Event != "user.upgraded"{
		w.WriteHeader(204)
		return
	}

	parsed_id, _ := uuid.Parse(params.Data.UserID)

	rows, err := cfg.dbQueries.UpdateChirpyRed(r.Context(), parsed_id)
	if err != nil{
		errors := errorResponse{
			Error: "error updating the user plan",
		}
		data, err := json.Marshal(errors)
		if err != nil {
    		w.WriteHeader(http.StatusBadRequest)
    		return
		}
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(data)
		return
	}

	if rows == 0{
		w.WriteHeader(404)
		return
	}

	w.WriteHeader(http.StatusNoContent)

}

func main(){
	godotenv.Load()
	db_url := os.Getenv("DB_URL")
	platform := os.Getenv("PLATFORM")
	token_secret := os.Getenv("TOKEN_SECRET")
	polka_key := os.Getenv("POLKA_KEY")
	db, err := sql.Open("postgres", db_url)
	if err != nil{
		log.Fatal("error opening database connection")
	}
	dbQueries := database.New(db)
	mux := http.NewServeMux()
	apiCfg := &apiConfig{
		dbQueries: dbQueries,
		platform: platform,
		token_secret: token_secret,
		polka_key: polka_key,
	}
	fileserver := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", apiCfg.middlewareMetricsInc(fileserver))
	mux.HandleFunc("GET /api/healthz", healthzhandler)
	mux.HandleFunc("GET /admin/metrics", apiCfg.countHits)
	mux.HandleFunc("POST /admin/reset", apiCfg.resetHits)
	mux.HandleFunc("POST /api/chirps", apiCfg.chirpCreator)
	mux.HandleFunc("POST /api/users", apiCfg.createUser)
	mux.HandleFunc("GET /api/chirps", apiCfg.retrieveChirps)
	mux.HandleFunc("GET /api/chirps/{chirpId}", apiCfg.getChirp)
	mux.HandleFunc("POST /api/login", apiCfg.loginUser)
	mux.HandleFunc("POST /api/refresh", apiCfg.refreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.revokeRefreshToken)
	mux.HandleFunc("PUT /api/users", apiCfg.updateCredentials)
	mux.HandleFunc("DELETE /api/chirps/{chirpId}", apiCfg.chirpDelete)
	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.updateChirpyRed)

	server := &http.Server{
		Handler: mux,
		Addr: ":8080",
	}

	server.ListenAndServe()
}