package main

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/jaingounchained/todo/api"
	db "github.com/jaingounchained/todo/db/sqlc"
	_ "github.com/jaingounchained/todo/docs"
	storage "github.com/jaingounchained/todo/storage"
	localStorage "github.com/jaingounchained/todo/storage/local_directory"
	"github.com/jaingounchained/todo/util"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// @title     Todo API
// @version         1.0
// @description     A todo management service API in GO which supports attachments

// @contact.name   Bhavya Jain

// @host      localhost:8080
// @BasePath  /
func main() {
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
	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db:", err)
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
			log.Fatal("cannot setup storage:", err)
		}
	}

	store := db.NewStore(conn)

	// TODO: configure ginHandler to not limit the number of incoming bytes to ~10 MB: Give max memory to the gin engine
	ginHandler := api.NewGinHandler(store, storage, logger)
	httpServer := ginHandler.HttpServer(config.ServerAddress)

	// Initializing the http server
	go startHTTPServer(httpServer)

	done := make(chan os.Signal, 1)
	signal.Notify(done, syscall.SIGINT, syscall.SIGTERM)

	applicationShutdown(done, httpServer, conn)
}

func applicationShutdown(done <-chan os.Signal, httpServer *http.Server, dbConn *sql.DB) {
	<-done
	log.Println("Shutting down server...")

	// Close all database connection etc
	dbConn.Close()

	// Server has 5 seconds to finish
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	select {
	case <-ctx.Done():
		log.Println("timeout of 5 seconds.")
	}
	log.Println("Server exiting")
}

func startHTTPServer(httpServer *http.Server) {
	log.Println("Starting HTTP server...")
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("error: %s\n", err)
	}
}
