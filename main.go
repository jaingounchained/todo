package main

import (
	"context"
	"errors"
	"log"
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
	"go.uber.org/zap"
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
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Logger setup")

	// Load config
	config, err := util.LoadConfig(".")
	if err != nil {
		logger.Fatal("Failed to load config: ", zap.Error(err))
	}

	// Setup DB connection
	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		logger.Fatal("Failed to connect to the db: ", zap.Error(err))
	}

	if err := connPool.Ping(context.Background()); err != nil {
		logger.Fatal("Failed to ping the db: ", zap.Error(err))
	}

	// run db migration
	runDBMigration(logger, config.MigrationURL, config.DBSource)

	store := db.NewStore(connPool)

	// Setup file storage
	cwd, err := os.Getwd()
	if err != nil {
		logger.Fatal("Failed to calculate current working directory: ", zap.Error(err))
	}

	var storage storage.Storage
	switch config.StorageType {
	case "LOCAL":
		localStoragePath := filepath.Join(cwd, config.LocalStorageDirectory)

		storage, err = localStorage.New(logger, localStoragePath)
		if err != nil {
			logger.Fatal("cannot setup file storage for the local storageType: ", zap.Error(err))
		}
	case "S3":
		logger.Fatal("S3 file storage unimplemented")
	default:
		logger.Fatal("Invalid file storage type chosen")
	}

	// Initializing the http server
	server, err := api.NewGinHandler(config, store, storage, logger)
	if err != nil {
		logger.Fatal("Unable to create server", zap.Error(err))
	}

	httpServer := server.HttpServer(config.ServerAddress)

	go startHTTPServer(logger, httpServer)

	applicationShutdown(logger, done, httpServer, connPool, storage)
}

func runDBMigration(logger *zap.Logger, migrationURL, dbSource string) {
	migration, err := migrate.New(migrationURL, dbSource)
	if err != nil {
		logger.Fatal("cannot create new migrate instance", zap.Error(err))
	}

	if err := migration.Up(); err != nil && err != migrate.ErrNoChange {
		logger.Fatal("failed to run migrate up with error", zap.Error(err))
	}

	logger.Info("db migrated successfully")
}

func applicationShutdown(logger *zap.Logger, done <-chan os.Signal, httpServer *http.Server, connPool *pgxpool.Pool, storage storage.Storage) {
	<-done
	logger.Info("Shutting down server...")

	// Server has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all database connections
	go connPool.Close()

	// Close storage connections
	go storage.CloseConnection(context.Background())

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: ", zap.Error(err))
	}

	logger.Info("Server exiting...")
}

func startHTTPServer(logger *zap.Logger, httpServer *http.Server) {
	logger.Info("Starting HTTP server...")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logger.Error("", zap.Error(err))
	}
}
