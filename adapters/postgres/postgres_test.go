package adapters

import (
	"context"
	"database/sql"
	_ "embed"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib" // Required for compatibility with pgtestdb.Config
	"github.com/peterldowns/pgtestdb"

	c "ledger/core"
	ct "ledger/core/test"
)

func TestPostgresDatabase(t *testing.T) {
	ct.RunEngineTests(t, func(t *testing.T) c.Storage {
		return setupTestDB(t)
	})
}

//go:embed migrations/000001_init_up.sql
var migrationScript string

// Custom pgtestdb migrator implementation

type migrator struct{}

func (migrator) Hash() (string, error) {
	return "000001_init_up", nil
}

func (migrator) Migrate(ctx context.Context, db *sql.DB, conf pgtestdb.Config) error {
	_, err := db.ExecContext(ctx, migrationScript)
	return err
}

func setupTestDB(t *testing.T) *PostgresDatabase {
	t.Helper()

	conf := pgtestdb.Config{
		DriverName: "pgx",
		User:       "postgres",
		Password:   "password123",
		Host:       "localhost",
		Port:       "5432",
		Options:    "sslmode=disable",
	}

	dbConfig := pgtestdb.Custom(t, conf, migrator{})
	ctx := context.TODO()
	db, err := New(ctx, dbConfig.URL())
	if err != nil {
		t.Fatalf("failed to initialize database pool: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}
