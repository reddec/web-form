package storage_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/reddec/web-form/internal/storage"

	"github.com/jackc/pgx/v5"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"
)

var dbURL string

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	// uses pool to try to connect to Docker
	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.Run("postgres", "14", []string{"POSTGRES_PASSWORD=password"})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	if err := pool.Retry(func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		dbURL = fmt.Sprintf("postgres://postgres:password@localhost:%s/postgres?sslmode=disable", resource.GetPort("5432/tcp"))
		db, err := pgx.Connect(ctx, dbURL)
		if err != nil {
			return err
		}
		defer db.Close(ctx)
		return db.Ping(ctx)
	}); err != nil {
		log.Fatalf("Could not connect to database: %s", err)
	}

	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestNewDB_pg(t *testing.T) {
	// NOTE: postgres by default converts name text case of column names from CREATE TABLE statement to lower.
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s, err := storage.NewDB(ctx, "postgres", dbURL)
	require.NoError(t, err)
	defer s.Close()

	err = s.Exec(ctx, `CREATE TABLE pizza (
    ID BIGSERIAL NOT NULL PRIMARY KEY,
    CUSTOMER TEXT NOT NULL,
    ADDRESS TEXT[] NOT NULL,
    QTY BIGINT NOT NULL,
    PRICE DOUBLE PRECISION NOT NULL , -- do not real DOUBLE in production
    DELIVERED BOOLEAN NOT NULL DEFAULT FALSE,
    "WEIRD COLUMN" TEXT NOT NULL DEFAULT 'hello world'
)`)
	require.NoError(t, err)

	res, err := s.Store(ctx, "pizza", map[string]any{
		"customer": "demo",
		"address":  []string{"Little Village", "New York"},
		"qty":      2,
		"price":    123.456,
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"id":           int64(1),
		"customer":     "demo",
		"address":      []any{"Little Village", "New York"},
		"qty":          int64(2),
		"price":        123.456,
		"delivered":    false,
		"WEIRD COLUMN": "hello world",
	}, res)
}

func TestNewDB_sqlite(t *testing.T) {
	// NOTE: sqlite keep case of column names from CREATE TABLE statement.

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	s, err := storage.NewDB(ctx, "sqlite", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer s.Close()

	err = s.Exec(ctx, `CREATE TABLE pizza (
    id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    customer TEXT NOT NULL,
    address TEXT NOT NULL,
    qty INTEGER NOT NULL,
    price DOUBLE PRECISION NOT NULL , -- do not real DOUBLE in production
    delivered BOOLEAN NOT NULL DEFAULT FALSE,
    "WEIRD COLUMN" TEXT NOT NULL DEFAULT 'hello world'
)`)
	require.NoError(t, err)

	res, err := s.Store(ctx, "pizza", map[string]any{
		"customer": "demo",
		"address":  []string{"Little Village", "New York"},
		"qty":      2,
		"price":    123.456,
	})
	require.NoError(t, err)

	assert.Equal(t, map[string]any{
		"id":           int64(1),
		"customer":     "demo",
		"address":      "Little Village,New York",
		"qty":          int64(2),
		"price":        123.456,
		"delivered":    int64(0),
		"WEIRD COLUMN": "hello world",
	}, res)
}
