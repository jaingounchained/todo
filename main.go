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

// @title     Todo API
// @version         1.0
// @description     A todo management service API in GO which supports attachments
// @contact.name   Bhavya Jain
// @host      localhost:8080
// @BasePath  /
func main() {
	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	// Logging setup
	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer logger.Sync()

	logger.Info("Setup logger")

	// Load config
	config, err := util.LoadConfig(".")
	if err != nil {
		logger.Fatal("Failed tp load config", zap.Error(err))
	}

	// Setup DB connection
	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
	}

	if err := connPool.Ping(context.Background()); err != nil {
		log.Fatal("Cannot ping the db:", err)
		os.Exit(1)
	}

	// Setup file storage
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	var storage storage.Storage
	if config.StorageType == "LOCAL" {
		localStoragePath := filepath.Join(cwd, config.LocalStorageDirectory)
		storage, err = localStorage.New(localStoragePath)
		if err != nil {
			done <- syscall.SIGTERM
			logger.Error("cannot setup storage:", zap.Any("Error:", err))
		}
	}

	store := db.NewStore(connPool)

	// TODO: configure ginHandler to not limit the number of incoming bytes to ~10 MB: Give max memory to the gin engine
	ginHandler := api.NewGinHandler(store, storage, logger)
	httpServer := ginHandler.HttpServer(config.ServerAddress)

	// Initializing the http server
	go startHTTPServer(httpServer)

	applicationShutdown(done, httpServer, connPool)
}

func applicationShutdown(done <-chan os.Signal, httpServer *http.Server, connPool *pgxpool.Pool) {
	<-done
	log.Println("Shutting down server...")

	// Close all database connection etc
	connPool.Close()

	// Server has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}

func startHTTPServer(httpServer *http.Server) {
	log.Println("Starting HTTP server...")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("error: %s\n", err)
	}
}
