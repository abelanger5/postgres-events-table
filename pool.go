package main

import (
	"context"
	"log"
	"os"

	"github.com/abelanger5/postgres-events-table/internal/dbsqlc"
	"github.com/jackc/pgx/v5/pgxpool"
)

var pool *pgxpool.Pool
var queries *dbsqlc.Queries

func init() {
	dbUrl := os.Getenv("DATABASE_URL")

	if dbUrl == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	config, err := pgxpool.ParseConfig(dbUrl)

	if err != nil {
		log.Fatal("could not parse DATABASE_URL: %w", err)
	}

	config.MaxConns = 5

	pool, err = pgxpool.NewWithConfig(context.Background(), config)

	if err != nil {
		log.Fatal("could not create connection pool: %w", err)
	}

	queries = dbsqlc.New()
}
