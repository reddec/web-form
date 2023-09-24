package storage

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/reddec/web-form/internal/utils"

	"github.com/jackc/pgx/v5/stdlib"
	migrate "github.com/rubenv/sql-migrate"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jmoiron/sqlx"
)

var ErrUnsupportedDatabase = errors.New("unsupported database schema")

// MySQL not supported due to: no RETURNING clause, different escape approaches and other similar uncommon behaviour.

type DBStore interface {
	ClosableStorage
	Exec(ctx context.Context, query string) error
	Migrate(ctx context.Context, sourceDir string) error
}

func NewDB(ctx context.Context, dialect string, dbURL string) (DBStore, error) {
	switch dialect {
	case "postgres", "postgresql", "pgx", "pg", "postgress":
		pool, err := pgxpool.New(ctx, dbURL)
		if err != nil {
			return nil, fmt.Errorf("create pool: %w", err)
		}
		return &pgStore{pool: pool}, nil
	case "sqlite", "sqlite3", "lite", "file", ":memory:", "":
		db, err := sqlx.Open("sqlite", dbURL)
		if err != nil {
			return nil, fmt.Errorf("open DB: %w", err)
		}
		return &liteStore{pool: db}, nil
	default:
		return nil, fmt.Errorf("%q: %w", dialect, ErrUnsupportedDatabase)
	}
}

type pgStore struct {
	pool *pgxpool.Pool
}

func (s *pgStore) Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error) {
	var query strings.Builder

	keys := utils.Keys(fields)

	query.WriteString("INSERT INTO ")
	utils.QuoteBuilder(&query, table, '"')
	query.WriteString(" (")

	// escape columns
	for i, name := range keys {
		if i > 0 {
			_, _ = query.WriteString(", ")
		}
		utils.QuoteBuilder(&query, name, '"')
	}
	query.WriteString(") VALUES (")

	// generate params
	var params = make([]any, 0, len(fields))
	for i := range keys {
		if i > 0 {
			_, _ = query.WriteString(", ")
		}
		query.WriteRune('$')
		query.WriteString(strconv.Itoa(i + 1))
		params = append(params, fields[keys[i]])
	}
	query.WriteString(") RETURNING *")

	var result = make(map[string]any)
	rows, err := s.pool.Query(ctx, query.String(), params...)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, fmt.Errorf("get row: %w", err)
	}

	descriptions := rows.FieldDescriptions()
	values, err := rows.Values()
	if err != nil {
		return nil, fmt.Errorf("get row values: %w", err)
	}

	for i, f := range descriptions {
		result[f.Name] = values[i]
	}
	return result, nil
}

func (s *pgStore) Exec(ctx context.Context, query string) error {
	_, err := s.pool.Exec(ctx, query)
	return err
}

func (s *pgStore) Migrate(ctx context.Context, sourceDir string) error {
	cfg := s.pool.Config().ConnConfig
	db := stdlib.OpenDB(*cfg)
	defer db.Close()
	_, err := migrate.ExecContext(ctx, db, "postgres", migrate.FileMigrationSource{Dir: sourceDir}, migrate.Up)
	return err
}

func (s *pgStore) Close() error {
	s.pool.Close()
	return nil
}

type liteStore struct {
	pool *sqlx.DB
}

func (s *liteStore) Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error) {
	var query strings.Builder

	keys := utils.Keys(fields)

	query.WriteString("INSERT INTO ")
	utils.QuoteBuilder(&query, table, '"')
	query.WriteString(" (")

	// escape columns
	for i, name := range keys {
		if i > 0 {
			_, _ = query.WriteString(", ")
		}
		utils.QuoteBuilder(&query, name, '"')
	}
	query.WriteString(") VALUES (")

	// generate params
	var params = make([]any, 0, len(fields))
	for i := range keys {
		if i > 0 {
			_, _ = query.WriteString(", ")
		}
		query.WriteRune('?') // difference between PG

		param := fields[keys[i]]

		if isArray(param) {
			// corner case for sqlite since it's not supporting native arrays.
			// it will concat values via comma without escaping.
			// it's slow and prone to errors if values have commas.
			param = joinIterable(param, ",")
		}

		params = append(params, param)
	}
	query.WriteString(") RETURNING *")

	var result = make(map[string]any)
	row := s.pool.QueryRowxContext(ctx, query.String(), params...)

	if err := row.MapScan(result); err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	return result, nil
}

func (s *liteStore) Exec(ctx context.Context, query string) error {
	_, err := s.pool.ExecContext(ctx, query)
	return err
}

func (s *liteStore) Close() error {
	return s.pool.Close()
}

func (s *liteStore) Migrate(ctx context.Context, sourceDir string) error {
	_, err := migrate.ExecContext(ctx, s.pool.DB, "sqlite3", migrate.FileMigrationSource{Dir: sourceDir}, migrate.Up)
	return err
}

func isArray(value any) bool {
	kind := reflect.TypeOf(value).Kind()
	return kind == reflect.Array || kind == reflect.Slice
}

func joinIterable(value any, sep string) string {
	v := reflect.ValueOf(value)
	n := v.Len()
	var out strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			out.WriteString(sep)
		}
		out.WriteString(fmt.Sprint(v.Index(i).Interface()))
	}
	return out.String()
}
