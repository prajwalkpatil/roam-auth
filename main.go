package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	db "roam-auth/db/sqlc"

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
	fmt.Println("Query result: ", res)
}

func createUser(queries *db.Queries, name string, email string) (string, error) {
	id, err := queries.CreateAuthUser(context.Background(), email)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return "", ErrEmailAlreadyExists
			}
		}
	}
	_, err = queries.CreatePublicUser(context.Background(), db.CreatePublicUserParams{
		ID:   id,
		Name: name,
	})
	if err != nil {
		return "", err
	}
	return id.String(), nil
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
	_, err = createUser(queries, "Prajwal Patil", "prajwalpatilk@gmail.com")
	if err != nil {
		if errors.Is(err, ErrEmailAlreadyExists) {
			fmt.Println("DUPLICATE USER: ", err)
		} else {
			fmt.Println(err)
		}
	}
	printUsers(queries)
}
