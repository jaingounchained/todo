package gapi

import (
	"fmt"

	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/pb"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
)

// Server serves GRPC requests for todo service
type Server struct {
	pb.UnimplementedTodoServer
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
}

// NewGRPCServer creates a new GRPC server
func NewGRPCServer(config util.Config, store db.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	return &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}, nil
}
