package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
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

func createRefreshToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func generateRandomEmail() (string, error) {
	b := make([]byte, 10)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(b) + "@gmail.com", nil
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

}
