package api

import (
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func newTestServer(t *testing.T, store db.Store, storage storage.Storage) *Server {
	config := util.Config{
		TokenSymmetricKey:   util.RandomString(32),
		AccessTokenDuration: time.Minute,
	}

	server, err := NewGinHandler(config, store, storage)
	require.NoError(t, err)

	return server
}

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	os.Exit(m.Run())
}
