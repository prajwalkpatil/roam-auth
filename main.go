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

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
)

var ErrEmailAlreadyExists error = errors.New("Email already exists")
var RefreshTokenNotFound error = errors.New("Refresh Token Not Found")
var REFRESH_TOKEN_EXPIRY_DAYS int = 15

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
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

func loginUser(ctx context.Context, conn *pgx.Conn, queries *db.Queries, payload LoginRequest) (bool, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return false, err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)
	passwordResult, err := qtx.GetUserPasswordFromEmail(ctx, payload.Email)
	if err != nil {
		return false, err
	}
	fmt.Println("passwordResult: ", passwordResult)

	isValid := isValidPassword(passwordResult.HashedPassword, payload.Password)
	if !isValid {
		return false, nil
	}
	fmt.Println("isValidPassword: ", isValid)
	token, err := createRefreshToken()
	if err != nil {
		return false, nil
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
		return false, err
	}
	fmt.Println("Refresh token:", refreshResult)
	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
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
	uid, err := uuid.Parse(id)
	if err != nil {
		return false, err
	}
	tokenResult, err := queries.GetRefreshToken(ctx, db.GetRefreshTokenParams{
		ID:           uid,
		RefreshToken: token,
	})
	if err != nil {
		return false, err
	}
	return tokenResult.RefreshToken == token, nil
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
		return result, RefreshTokenNotFound
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

func handleLogin(w http.ResponseWriter, _ *http.Request) {
	fmt.Println("/login called")
	fmt.Fprintf(w, "Hello, World")
}

func main() {
	godotenv.Load()
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	queries := db.New(conn)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /login", func(w http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(w, "Hello, World")
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
		success, err := loginUser(context.Background(), conn, queries, payload)
		if err != nil {
			fmt.Println("Login error: ", err)
			http.Error(w, "Unable to login", http.StatusInternalServerError)
			return
		}
		if success {
			fmt.Fprintf(w, "Hello %s!", payload.Email)
		} else {
			http.Error(w, "Invalid Password", http.StatusBadRequest)
		}
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
