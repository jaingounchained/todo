package db

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

var testStore Store

func TestMain(m *testing.M) {
	connURI := CreateTestPostgresContainer()

	connPool, err := pgxpool.New(context.Background(), connURI)
	if err != nil {
		log.Fatal("cannot connect to db: ", err)
	}

	// Run db migrations
	runDBMigration("file://./../migration/", connURI)

	testStore = NewStore(connPool)

	code := m.Run()
	m.Run()
	os.Exit(code)
}

func CreateTestPostgresContainer() string {
	ctx := context.Background()
	container, err := postgres.Run(
		ctx,
		"postgres:14",
		postgres.WithDatabase("todos"),
		postgres.WithUsername("root"),
		postgres.WithPassword("secret"),
		testcontainers.
			WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Fatal("Cannot setup test postgres container with error: ", err)
	}

	connURI, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Fatal("Cannot setup test postgres container with error: ", err)
	}

	return connURI
}

func runDBMigration(migrationURL string, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal("cannot create new migrate instance with error: ", err)
	}

	if err = migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal("failed to run migrate up with error: ", err)
	}
}
