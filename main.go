package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	db "roam-auth/db/sqlc"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyExists error = errors.New("EMAIL_ALREADY_EXISTS")
var ErrRefreshTokenNotFound error = errors.New("REFRESH_TOKEN_NOT_FOUND")
var ErrInvalidJWT error = errors.New("INVALID_JWT")
var ErrExpiredJWT error = errors.New("EXPIRED_JWT")
var ErrEmailDoesNotExist error = errors.New("EMAIL_DOES_NOT_EXIST")
var ErrUserIdDoesNotExist error = errors.New("USER_ID_DOES_NOT_EXIST")
var ErrInvalidPassword error = errors.New("INVALID_PASSWORD")

var REFRESH_TOKEN_EXPIRY_DAYS int = 15
var REFRESH_TOKEN_COOKIE_NAME string = "Token"

var CLAIMS_CONTEXT_KEY = "claims"

var JWT_EXPIRY_MINUTES int = 15
var JWT_SIGNING_ALGO = jwt.SigningMethodHS256
var jwtSigningKey []byte

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	ID           string `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"-"`
	Valid        bool   `json:"-"`
}

type ProfileResponse struct {
	ID    string `json:"-"`
	Email string `json:"email"`
	Name  string `json:"name"`
}

type ErrorResponse struct {
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type UserJWTClaims struct {
	ID    string
	Email string
	jwt.RegisteredClaims
}

func hashPassword(password string) (string, error) {
	passBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(passBytes), err
}

func isValidPassword(hashedPassword string, inputPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(inputPassword))
	if err != nil {
		return false
	}
	return true
}

func createRefreshToken(ctx context.Context, queries *db.Queries, uid uuid.UUID, refreshCookie *http.Cookie) (string, error) {
	newRefreshToken, err := newRefreshToken()
	if err != nil {
		return "", err
	}

	if refreshCookie != nil {
		oldRefreshToken := refreshCookie.Value
		affectedRows, err := queries.ReplaceRefreshToken(ctx, db.ReplaceRefreshTokenParams{
			OldRefreshToken: oldRefreshToken,
			NewRefreshToken: newRefreshToken,
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
				Valid: true,
			},
		})
		if err != nil {
			return "", err
		}
		if affectedRows > 0 {
			return newRefreshToken, nil
		}
	}

	refreshResult, err := queries.AddRefreshToken(ctx, db.AddRefreshTokenParams{
		ID:           uid,
		RefreshToken: newRefreshToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
			Valid: true,
		},
	})
	if err != nil {
		fmt.Println("Error adding Refresh token: ", err)
		return "", err
	}
	return refreshResult.RefreshToken, nil
}

func loginUser(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, payload LoginRequest, refreshCookie *http.Cookie) (*LoginResponse, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	passwordResultRows, err := qtx.GetUserPasswordFromEmail(ctx, payload.Email)
	if err != nil {
		return nil, err
	}
	if len(passwordResultRows) == 0 {
		fmt.Println("Email does not exist")
		return nil, ErrEmailDoesNotExist
	}
	passwordResult := passwordResultRows[0]
	fmt.Println("passwordResult: ", passwordResult)

	isValid := isValidPassword(passwordResult.HashedPassword, payload.Password)
	if !isValid {
		return nil, ErrInvalidPassword
	}
	fmt.Println("isValidPassword: ", isValid)

	refreshToken, err := createRefreshToken(ctx, qtx, passwordResult.ID, refreshCookie)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	uid := passwordResult.ID.String()
	jwtString, err := createJWTString(uid, payload.Email)
	if err != nil {
		return nil, err
	}
	return &LoginResponse{
		ID:           uid,
		Email:        payload.Email,
		RefreshToken: refreshToken,
		Token:        jwtString,
		Valid:        true,
	}, nil
}

func signupUser(ctx context.Context, pool *pgxpool.Pool, queries *db.Queries, payload SignupRequest) (bool, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)

	qtx := queries.WithTx(tx)

	id, err := qtx.CreateAuthUser(ctx, payload.Email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return false, fmt.Errorf("%w: %v", ErrEmailAlreadyExists, err)
		}
		return false, err
	}
	id, err = qtx.CreatePublicUser(ctx, db.CreatePublicUserParams{
		ID:   id,
		Name: payload.Name,
	})
	if err != nil {
		return false, err
	}
	hashedPassword, err := hashPassword(payload.Password)
	if err != nil {
		return false, err
	}
	_, err = qtx.CreateUserPassword(ctx, db.CreateUserPasswordParams{
		ID:             id,
		HashedPassword: hashedPassword,
	})

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func getUserFromRefreshToken(ctx context.Context, queries *db.Queries, refreshToken string) (*LoginResponse, error) {
	rows, err := queries.GetUserFromRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, err
	}
	if len(rows) != 1 {
		return nil, ErrRefreshTokenNotFound
	}
	user := rows[0]
	return &LoginResponse{
		ID:    user.ID.String(),
		Email: user.Email,
	}, nil
}

func newRefreshToken() (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b), nil
}

func createJWTString(id string, email string) (string, error) {
	claims := UserJWTClaims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(JWT_EXPIRY_MINUTES) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(JWT_SIGNING_ALGO, claims)
	tokenString, err := token.SignedString(jwtSigningKey)
	if err != nil {
		return "", err
	}
	return tokenString, err
}

func createRefreshCookie(token string) *http.Cookie {
	return &http.Cookie{
		Name:     REFRESH_TOKEN_COOKIE_NAME,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
	}
}

func deleteRefreshToken(ctx context.Context, queries *db.Queries, id string, refreshToken string) (bool, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}
	result, err := queries.DeleteRefreshToken(ctx, db.DeleteRefreshTokenParams{
		ID:           uid,
		RefreshToken: refreshToken,
	})
	if err != nil {
		return false, err
	}
	if result < 1 {
		return false, nil
	}
	return true, nil
}

func getUserFromId(ctx context.Context, queries *db.Queries, id string) (*ProfileResponse, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return nil, err
	}
	resultRow, err := queries.GetUserFromId(ctx, uid)
	if err != nil {
		return nil, err
	}
	if len(resultRow) == 0 {
		return nil, ErrUserIdDoesNotExist
	}
	result := resultRow[0]
	return &ProfileResponse{
		ID:    result.ID.String(),
		Email: result.Email,
		Name:  result.Name,
	}, nil
}

func parseJWTClaims(tokenString string, signingKey []byte) (*UserJWTClaims, error) {
	parsedToken, err := jwt.ParseWithClaims(tokenString, &UserJWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidJWT
		}
		return signingKey, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredJWT
		}
		return nil, err
	}
	claims, ok := parsedToken.Claims.(*UserJWTClaims)
	if !ok || !parsedToken.Valid {
		return nil, ErrInvalidJWT
	}
	return claims, nil
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		authItems := strings.Split(authHeader, " ")
		if len(authItems) != 2 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jwtString := authItems[1]
		claims, err := parseJWTClaims(jwtString, jwtSigningKey)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), CLAIMS_CONTEXT_KEY, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	json.NewEncoder(w).Encode(ErrorResponse{
		Status: http.StatusBadRequest,
		Error:  err.Error(),
	})
}

func normalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}

func main() {
	godotenv.Load()
	pool, err := pgxpool.New(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println("Pool creation error: ", err)
		os.Exit(1)
	}
	defer pool.Close()

	jwtSigningKey = []byte(os.Getenv("JWT_SIGNING_KEY"))

	mux := http.NewServeMux()

	mux.HandleFunc("POST /refresh", func(w http.ResponseWriter, r *http.Request) {
		queries := db.New(pool)
		refreshToken, err := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		user, err := getUserFromRefreshToken(context.Background(), queries, refreshToken.Value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		jwtString, err := createJWTString(user.ID, user.Email)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		user.Token = jwtString
		json.NewEncoder(w).Encode(user)
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		queries := db.New(pool)
		var payload LoginRequest
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			fmt.Println("JSON decode error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		payload.Email = normalizeEmail(payload.Email)
		existingRefreshCookie, _ := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
		loginResponse, err := loginUser(context.Background(), pool, queries, payload, existingRefreshCookie)
		if err != nil {
			if errors.Is(err, ErrInvalidPassword) {
				writeError(w, ErrInvalidPassword)
				return
			} else if errors.Is(err, ErrEmailDoesNotExist) {
				writeError(w, ErrEmailDoesNotExist)
				return
			}
			fmt.Println("Login error: ", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		refreshCookie := createRefreshCookie(loginResponse.RefreshToken)
		http.SetCookie(w, refreshCookie)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse)
	})

	mux.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		queries := db.New(pool)
		var payload SignupRequest
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			fmt.Println("JSON decode error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		payload.Email = normalizeEmail(payload.Email)
		_, err = signupUser(context.Background(), pool, queries, payload)
		if err != nil {
			fmt.Printf("Signup error: %s", err)
			if errors.Is(err, ErrEmailAlreadyExists) {
				writeError(w, ErrEmailAlreadyExists)
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "User registered %s!", payload.Name)
	})

	mux.Handle("POST /logout", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries := db.New(pool)
		claims, ok := r.Context().Value(CLAIMS_CONTEXT_KEY).(*UserJWTClaims)
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		id := claims.ID
		refreshCookie, err := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		success, err := deleteRefreshToken(context.Background(), queries, id, refreshCookie.Value)
		if err != nil || !success {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	})))

	mux.Handle("GET /profile", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		queries := db.New(pool)
		claims, _ := r.Context().Value(CLAIMS_CONTEXT_KEY).(*UserJWTClaims)
		response, err := getUserFromId(context.Background(), queries, claims.ID)
		if err != nil {
			if errors.Is(err, ErrUserIdDoesNotExist) {
				http.Error(w, "Invalid User ID", http.StatusBadRequest)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			fmt.Println("Error while fetching profile: ", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	})))

	mux.Handle("GET /", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := r.Context().Value(CLAIMS_CONTEXT_KEY).(*UserJWTClaims)
		fmt.Println("User Claims:", *claims)
		fmt.Fprintf(w, "Hello, World")
	})))

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{os.Getenv("CLIENT_URL")},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	})

	srv := &http.Server{
		Addr:         ":8000",
		Handler:      c.Handler(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("Listening on port:", 8000)
	log.Fatal(srv.ListenAndServe())
}
