// cmd/seed-super-admin/main.go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/alexedwards/argon2id"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

func main() {
	email := flag.String("email", "", "email do super admin")
	password := flag.String("password", "", "senha do super admin")
	name := flag.String("name", "", "nome do super admin")
	flag.Parse()

	if *email == "" || *password == "" || *name == "" {
		log.Fatal("uso: --email --password --name são obrigatórios")
	}

	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL not set")
	}

	pepper := os.Getenv("SUPER_ADMIN_PEPPER")
	if pepper == "" {
		log.Fatal("SUPER_ADMIN_PEPPER not set")
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	hash, err := argon2id.CreateHash(*password+pepper, argon2id.DefaultParams)
	if err != nil {
		log.Fatal(err)
	}

	_, err = pool.Exec(
		context.Background(),
		`INSERT INTO platform_admins (name, email, password_hash) VALUES ($1, $2, $3)`,
		*name, *email, hash,
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("super admin criado com sucesso")
}
