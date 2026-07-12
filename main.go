package main

import (
	"context"
	"fmt"
	"os"
	db "roam-auth/db/sqlc"

	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	conn, err := pgx.Connect(context.Background(), os.Getenv("DATABASE_URL"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close(context.Background())
	params := db.GetUsersParams{
		Limit:  10,
		Offset: 0,
	}
	queries := db.New(conn)
	res, err := queries.GetUsers(context.Background(), params)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	fmt.Println("Query result: ", res)
}
