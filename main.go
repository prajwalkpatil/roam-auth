package main

import (
	"context"
	"fmt"
	"os"
	db "roam-auth/db/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

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
		return "", err
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
	_, err = createUser(queries, "Freaking Insane", "freakinginsane@gmail.com")
	if err != nil {
		fmt.Println("Couldn't create user: ", err)
	}
	printUsers(queries)
}
