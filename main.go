package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaingounchained/todo/api"
	db "github.com/jaingounchained/todo/db/sqlc"
	_ "github.com/jaingounchained/todo/docs"
	storage "github.com/jaingounchained/todo/storage"
	localStorage "github.com/jaingounchained/todo/storage/local_directory"
	"github.com/jaingounchained/todo/util"
	"github.com/rs/zerolog/log"
)

//	@title			Todo API
//	@version		1.0
//	@description	A todo management service API in go which supports attachments

//	@contact.name	Bhavya Jain

//	@host		localhost:8080
//	@BasePath	/

// @securityDefinitions.apikey	AccessTokenAuth
// @in							header
// @name						Authorization
// @description				To access todos and attachments
func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Logging setup
	log.Info().Msg("logger setup")

	// Load config
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config")
	}

	// Setup DB connection
	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to the postgreSQL db")
	}

	if err := connPool.Ping(context.Background()); err != nil {
		log.Fatal().Err(err).Msg("failed to ping the db")
	}

	// run db migration
	runDBMigration(config.MigrationURL, config.DBSource)

	store := db.NewStore(connPool)

	// Setup file storage
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to calculate current working directory")
	}

	var storage storage.Storage
	switch config.StorageType {
	case "LOCAL":
		localStoragePath := filepath.Join(cwd, config.LocalStorageDirectory)

		storage, err = localStorage.New(localStoragePath)
		if err != nil {
			log.Fatal().Err(err).Msg("cannot setup file storage for the local storageType")
		}
	case "S3":
		log.Fatal().Msg("s3 file storage unimplemented")
	default:
		log.Fatal().Msg("invalid file storage type chosen")
	}

	// Initializing the http server
	server, err := api.NewGinHandler(config, store, storage)
	if err != nil {
		log.Fatal().Err(err).Msg("unable to create server")
	}

	httpServer := server.HttpServer(config.ServerAddress)

	go startHTTPServer(httpServer)

	applicationShutdown(done, httpServer, connPool, storage)
}

func runDBMigration(migrationURL, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot create new migrate instance")
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal().Err(err).Msg("failed to run migrate up")
	}

	log.Info().Msg("db migrated successfully")
}

func applicationShutdown(done <-chan os.Signal, httpServer *http.Server, connPool *pgxpool.Pool, storage storage.Storage) {
	<-done
	log.Info().Msg("shutting down server...")

	// Server has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all database connections
	go connPool.Close()

	// Close storage connections
	go storage.CloseConnection(context.Background())

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Error().Err(err).Msg("server forced to shutdown")
	}

	log.Info().Msg("server exiting...")
}

func startHTTPServer(httpServer *http.Server) {
	log.Info().Msg("starting HTTP server...")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Error().Err(err).Msg("failed to start HTTP server")
	}
}
