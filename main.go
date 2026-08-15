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

func printUsers(queries *db.Queries) {
	res, err := queries.GetUsers(context.Background(), db.GetUsersParams{
		Limit:  10,
		Offset: 0,
	})
	if err != nil {
		fmt.Println("Failed to execute the query: ", err)
	}
	fmt.Printf("Query result: %v", res)
}

func createUser(ctx context.Context, conn *pgx.Conn, queries *db.Queries, name string, email string) (string, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)

	qtx := queries.WithTx(tx)

	id, err := qtx.CreateAuthUser(ctx, email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return "", fmt.Errorf("%w: %v", ErrEmailAlreadyExists, err)
		}
		return "", err
	}
	id, err = qtx.CreatePublicUser(ctx, db.CreatePublicUserParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return id.String(), nil
}

func createUserPassword(ctx context.Context, conn *pgx.Conn, queries *db.Queries, id string, hashedPassword string) (string, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	qtx := queries.WithTx(tx)

	userId, err := uuid.Parse(id)
	if err != nil {
		return "", err
	}
	userId, err = qtx.CreateUserPassword(ctx, db.CreateUserPasswordParams{
		ID:             userId,
		HashedPassword: hashedPassword,
	})
	if err != nil {
		return "", fmt.Errorf("Couldn't create a password: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userId.String(), nil
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

func loginUser(ctx context.Context, queries *db.Queries, payload LoginRequest) (bool, error) {
	result, err := queries.GetUserPasswordFromEmail(ctx, payload.Email)
	if err != nil {
		return false, err
	}
	return isValidPassword(result.HashedPassword, payload.Password), nil
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

func generateRandomEmail() (string, error) {
	b := make([]byte, 10)
	rand.Read(b)
	return hex.EncodeToString(b) + "@gmail.com", nil
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
			Time: time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
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
			Time: time.Now().AddDate(0, 0, REFRESH_TOKEN_EXPIRY_DAYS),
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

func handleLogin(w http.ResponseWriter, req *http.Request) {
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
	randomEmail, _ := generateRandomEmail()
	id, err := createUser(context.Background(), conn, queries, "Prajwal", randomEmail)
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			fmt.Println("DUPLICATE USER: ", err)
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	printUsers(queries)

	testPwd := "thisisatestpassword"
	hashed, err := hashPassword(testPwd)
	if err != nil {
		fmt.Println("Couldn't hash the password: ", err)
	}
	id, err = createUserPassword(context.Background(), conn, queries, id, hashed)
	if err != nil {
		fmt.Println(err)
	}

	userId, err := uuid.Parse(id)
	hashedPass, err := queries.GetUserPassword(context.Background(), userId)
	if err != nil {
		fmt.Println("Error getting password: ", err)
	}
	fmt.Println("Encrypted Password: ", hashedPass)
	fmt.Println("Is Valid Password: ", isValidPassword(hashedPass, testPwd))

	token, _ := createRefreshToken()
	fmt.Println("Refresh Token: ", token)
	addRefreshResult, err := queries.AddRefreshToken(context.Background(), db.AddRefreshTokenParams{
		ID:           userId,
		RefreshToken: token,
		ExpiresAt:    pgtype.Timestamptz{Time: time.Now().AddDate(0, 0, 15), Valid: true},
	})
	if err != nil {
		fmt.Println("Error adding refresh token: ", err)
		os.Exit(1)
	}

	fmt.Println("Add refresh result: ", addRefreshResult)

	tokenResult, err := queries.GetRefreshToken(context.Background(), db.GetRefreshTokenParams{
		ID:           userId,
		RefreshToken: token,
	})
	if err != nil {
		fmt.Println("Error finding refresh token: ", err)
		os.Exit(1)
	}

	fmt.Println("Token Result: ", tokenResult)

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
		success, err := loginUser(context.Background(), queries, payload)
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

	log.Fatal(srv.ListenAndServe())

}
