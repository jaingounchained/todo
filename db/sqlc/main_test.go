package db

import (
	"context"
	"log"
	"os"
	"testing"

	_ "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jaingounchained/todo/util"
)

var testStore Store

func TestMain(m *testing.M) {
	config, err := util.LoadConfig("../..")
	if err != nil {
		log.Fatal("cannot load config: ", err)
	}

	connPool, err := pgxpool.New(context.Background(), config.DBSource)
	if err != nil {
		log.Fatal("cannot connect to db: ", err)
	}

	if err := connPool.Ping(context.Background()); err != nil {
		log.Fatal("cannot ping the db: ", err)
	}

	testStore = NewStore(connPool)

	code := m.Run()
	m.Run()
	os.Exit(code)
}
