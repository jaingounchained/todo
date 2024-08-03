package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	"github.com/jaingounchained/todo/api"
	db "github.com/jaingounchained/todo/db/sqlc"
	storage "github.com/jaingounchained/todo/storage/localDirectory"
	"github.com/jaingounchained/todo/util"
	_ "github.com/lib/pq"
)

func main() {
	// Load config
	config, err := util.LoadConfig(".")
	if err != nil {
		log.Fatal("cannot load config:", err)
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

	localStoragePath := filepath.Join(cwd, "uploads")
	localStorage, err := storage.New(localStoragePath)
	if err != nil {
		log.Fatal("cannot setup storage:", err)
	}

	store := db.NewStore(conn)

	// TODO: configure server to not limit the number of incoming bytes to ~10 MB: Give max memory to the gin engine
	server := api.NewServer(store, localStorage)

	err = server.Start(config.ServerAddress)
	if err != nil {
		log.Fatal("cannot start server")
	}
}
