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

var REFRESH_TOKEN_EXPIRY_DAYS int = 15

var CLAIMS_CONTEXT_KEY = "claims"

var JWT_COOKIE_NAME string = "Token"
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
	RefreshToken string `json:"refresh_token"`
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

func loginUser(ctx context.Context, conn *pgx.Conn, queries *db.Queries, payload LoginRequest) (LoginResponse, error) {
	var response LoginResponse
	tx, err := conn.Begin(ctx)
	if err != nil {
		return response, err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	passwordResult, err := qtx.GetUserPasswordFromEmail(ctx, payload.Email)
	if err != nil {
		return response, err
	}
	fmt.Println("passwordResult: ", passwordResult)

	isValid := isValidPassword(passwordResult.HashedPassword, payload.Password)
	if !isValid {
		return response, nil
	}
	fmt.Println("isValidPassword: ", isValid)
	token, err := createRefreshToken()
	if err != nil {
		return response, nil
	}
	fmt.Println("Refresh token: ", token)
	refreshResult, err := qtx.AddRefreshToken(ctx, db.AddRefreshTokenParams{
		ID:           passwordResult.ID,
		RefreshToken: token,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
			Valid: true,
		},
	})
	if err != nil {
		fmt.Println("Error adding Refresh token: ", err)
		return response, err
	}
	fmt.Println("Refresh token:", refreshResult)
	if err := tx.Commit(ctx); err != nil {
		return response, err
	}
	response = LoginResponse{
		ID:           passwordResult.ID.String(),
		Email:        payload.Email,
		RefreshToken: refreshResult.RefreshToken,
		Valid:        true,
	}
	return response, nil
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

func createRefreshToken() (string, error) {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b), nil
}

func addNewRefreshToken(ctx context.Context, conn *pgx.Conn, queries *db.Queries, id string, token string) (db.AuthToken, error) {
	var result db.AuthToken
	tx, err := conn.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	uid, err := uuid.Parse(id)
	if err != nil {
		return result, err
	}
	result, err = qtx.AddRefreshToken(ctx, db.AddRefreshTokenParams{
		ID:           uid,
		RefreshToken: token,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
			Valid: true,
		},
	})
	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}

func isValidRefreshToken(ctx context.Context, queries *db.Queries, id string, token string) (bool, error) {
	tokenResult, err := findRefreshToken(ctx, queries, id, token)
	if err != nil {
		return false, err
	}
	expiryTime := tokenResult.ExpiresAt.Time
	return (tokenResult.RefreshToken == token) && expiryTime.After(time.Now()), nil
}

func findRefreshToken(ctx context.Context, queries *db.Queries, id string, token string) (db.AuthToken, error) {
	uid, err := uuid.Parse(id)
	if err != nil {
		return db.AuthToken{}, err
	}
	return queries.GetRefreshToken(ctx, db.GetRefreshTokenParams{
		ID:           uid,
		RefreshToken: token,
	})
}

func replaceRefreshToken(ctx context.Context, conn *pgx.Conn, queries *db.Queries, id string, oldToken string, newToken string) (db.AuthToken, error) {
	var result db.AuthToken
	tx, err := conn.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer tx.Rollback(ctx)

	uid, err := uuid.Parse(id)
	if err != nil {
		return result, err
	}

	qtx := queries.WithTx(tx)
	rowsAffected, err := qtx.DeleteRefreshToken(ctx, db.DeleteRefreshTokenParams{
		ID:           uid,
		RefreshToken: oldToken,
	})
	if rowsAffected == 0 {
		return result, ErrRefreshTokenNotFound
	}

	result, err = qtx.AddRefreshToken(ctx, db.AddRefreshTokenParams{
		ID:           uid,
		RefreshToken: newToken,
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
			Valid: true,
		},
	})

	if err != nil {
		return result, err
	}

	if err := tx.Commit(ctx); err != nil {
		return result, err
	}
	return result, nil
}

func createJWTString(id string, email string, signingKey []byte) (string, error) {
	claims := UserJWTClaims{
		ID:    id,
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(JWT_EXPIRY_MINUTES) * time.Minute)),
		},
	}
	token := jwt.NewWithClaims(JWT_SIGNING_ALGO, claims)
	tokenString, err := token.SignedString(signingKey)
	if err != nil {
		return "", err
	}
	return tokenString, err
}

func createJWTCookie(id string, email string, signingKey []byte) (*http.Cookie, error) {
	tokenString, err := createJWTString(id, email, signingKey)
	if err != nil {
		return nil, err
	}
	return &http.Cookie{
		Name:     JWT_COOKIE_NAME,
		Value:    tokenString,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   true,
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
		jwtCookie, err := r.Cookie(JWT_COOKIE_NAME)
		if errors.Is(err, http.ErrNoCookie) {
			//Use refresh token
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		if err != nil {
			http.Error(w, "Unauthenticated", http.StatusUnauthorized)
			return
		}
		claims, err := parseJWTClaims(jwtCookie.Value, jwtSigningKey)
		if errors.Is(err, ErrExpiredJWT) {
			// Use refresh token
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

	mux.HandleFunc("POST /login", func(w http.ResponseWriter, r *http.Request) {
		var payload LoginRequest
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err != nil {
			fmt.Println("JSON decode error", err)
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		loginResponse, err := loginUser(context.Background(), conn, queries, payload)
		if err != nil {
			fmt.Println("Login error: ", err)
			http.Error(w, "Unable to login", http.StatusInternalServerError)
			return
		}
		if !loginResponse.Valid {
			http.Error(w, "Invalid Password", http.StatusBadRequest)
			return
		}
		jwtCookie, err := createJWTCookie(loginResponse.ID, loginResponse.Email, jwtSigningKey)
		if err != nil {
			http.Error(w, "Unable to login", http.StatusInternalServerError)
			return
		}
		http.SetCookie(w, jwtCookie)
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
