package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	db "roam-auth/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/joho/godotenv"
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

func createUserPassword(ctx context.Context, conn *pgx.Conn, queries *db.Queries, id string, encryptedPassword string) (string, error) {
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
		ID:                userId,
		EncryptedPassword: encryptedPassword,
	})

	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return userId.String(), nil
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
	id, err := createUser(context.Background(), conn, queries, "Prajwal 5", "prajwalpatil5@gmail.com")
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			fmt.Println("DUPLICATE USER: ", err)
		} else {
			fmt.Println(err)
		}
		os.Exit(1)
	}
	printUsers(queries)
	id, err = createUserPassword(context.Background(), conn, queries, id, "Thisisatestpassword")
	if err != nil {
		fmt.Println("Couldn't create password: ", err)
	}
	userId, err := uuid.Parse(id)
	password, err := queries.GetUserPassword(context.Background(), userId)
	if err != nil {
		fmt.Println("Error getting password: ", err)
	}
	fmt.Println("Password: ", password)

}
