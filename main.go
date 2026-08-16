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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyExists error = errors.New("Email already exists")
var ErrRefreshTokenNotFound error = errors.New("Refresh Token Not Found")
var ErrInvalidJWT error = errors.New("Invalid JWT")
var ErrExpiredJWT error = errors.New("Expired JWT")
var ErrInvalidPassword error = errors.New("Invalid Password")

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

func handleRefreshTokenOnSuccessfulLogin(ctx context.Context, queries *db.Queries, uid uuid.UUID, refreshCookie *http.Cookie) (string, error) {
	newRefreshToken, err := createRefreshToken()
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

func loginUser(ctx context.Context, conn *pgx.Conn, queries *db.Queries, payload LoginRequest, refreshCookie *http.Cookie) (*LoginResponse, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	passwordResult, err := qtx.GetUserPasswordFromEmail(ctx, payload.Email)
	if err != nil {
		return nil, err
	}
	fmt.Println("passwordResult: ", passwordResult)

	isValid := isValidPassword(passwordResult.HashedPassword, payload.Password)
	if !isValid {
		return nil, ErrInvalidPassword
	}
	fmt.Println("isValidPassword: ", isValid)

	refreshToken, err := handleRefreshTokenOnSuccessfulLogin(ctx, qtx, passwordResult.ID, refreshCookie)
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

func signupUser(ctx context.Context, conn *pgx.Conn, queries *db.Queries, payload SignupRequest) (bool, error) {
	tx, err := conn.Begin(ctx)
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

func createRefreshToken() (string, error) {
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
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		authItems := strings.Split(authHeader, " ")
		if len(authItems) != 2 {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		jwtString := authItems[1]
		claims, err := parseJWTClaims(jwtString, jwtSigningKey)
		if errors.Is(err, ErrExpiredJWT) {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), CLAIMS_CONTEXT_KEY, claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func main() {
	godotenv.Load()
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())

	jwtSigningKey = []byte(os.Getenv("JWT_SIGNING_KEY"))
	queries := db.New(conn)

	mux := http.NewServeMux()

	mux.HandleFunc("POST /refresh", func(w http.ResponseWriter, r *http.Request) {
		refreshToken, err := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		user, err := getUserFromRefreshToken(context.Background(), queries, refreshToken.Value)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		jwtString, err := createJWTString(user.ID, user.Email)
		if err != nil {
			http.Error(w, "Unexpected error occured", http.StatusInternalServerError)
			return
		}
		user.Token = jwtString
		json.NewEncoder(w).Encode(user)
	})

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var payload LoginRequest
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			fmt.Println("JSON decode error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		existingRefreshCookie, _ := r.Cookie(REFRESH_TOKEN_COOKIE_NAME)
		loginResponse, err := loginUser(context.Background(), conn, queries, payload, existingRefreshCookie)
		if err != nil {
			if errors.Is(err, ErrInvalidPassword) {
				http.Error(w, "Invalid Password", http.StatusBadRequest)
				return
			}
			fmt.Println("Login error: ", err)
			http.Error(w, "Unable to login", http.StatusInternalServerError)
			return
		}
		refreshCookie := createRefreshCookie(loginResponse.RefreshToken)
		http.SetCookie(w, refreshCookie)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(loginResponse)
	})

	mux.HandleFunc("POST /signup", func(w http.ResponseWriter, r *http.Request) {
		var payload SignupRequest
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			fmt.Println("JSON decode error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		_, err = signupUser(context.Background(), conn, queries, payload)
		if err != nil {
			if errors.Is(err, ErrEmailAlreadyExists) {
				http.Error(w, "Email already exists", http.StatusBadRequest)
			} else {
				http.Error(w, "Couldn't create the account", http.StatusInternalServerError)
			}
			fmt.Printf("Signup error: %s", err)
			return
		}
		fmt.Fprintf(w, "User registered %s!", payload.Name)
	})

	mux.Handle("GET /", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := r.Context().Value(CLAIMS_CONTEXT_KEY).(*UserJWTClaims)
		fmt.Println("User Claims:", *claims)
		fmt.Fprintf(w, "Hello, World")
	})))

	srv := &http.Server{
		Addr:         ":8000",
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("Listening on port:", 8000)
	log.Fatal(srv.ListenAndServe())
}
