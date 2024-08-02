package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/jaingounchained/todo/db/sqlc"
)

// Server serves HTTP requests for todo service
type Server struct {
	store  db.Store
	router *gin.Engine
}

// NewServer creates a new HTTP server and setup routing
func NewServer(store db.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("todoStatus", validTodoStatus)
	}

	router.GET("/health", server.health)

	router.POST("/todos", server.createTodo)
	router.POST("/todos/:id/title", server.updateTodoTitle)
	router.POST("/todos/:id/status", server.updateTodoStatus)
	router.GET("/todos/:id", server.getTodo)
	router.GET("/todos", server.listTodo)
	router.DELETE("/todos/:id", server.deleteTodo)

	server.router = router
	return server
}

// Start runs the HTTP server on a specific address
func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

func errorResponse(err error) gin.H {
	return gin.H{
		"error": err.Error(),
	}
}
