package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/jaingounchained/todo/db/sqlc"
	storage "github.com/jaingounchained/todo/storage"
)

// Server serves HTTP requests for todo service
type Server struct {
	store   db.Store
	storage storage.Storage
	router  *gin.Engine
}

// NewServer creates a new HTTP server and setup routing
func NewServer(store db.Store, storage storage.Storage) *Server {
	server := &Server{
		store:   store,
		storage: storage,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("todoStatus", validTodoStatus)
	}

	server.setupRouter()
	return server
}

func (server *Server) setupRouter() {
	router := gin.Default()

	// health check router
	router.GET("/health", server.health)

	server.setupGetResourceRouters(router)
	server.setupCreateResourceRouters(router)
	server.setupUpdateResourceRouters(router)
	server.setupDeleteResourceRouters(router)

	server.router = router
}

func (server *Server) setupGetResourceRouters(router *gin.Engine) {
	// Get todo
	router.GET("/todos", server.listTodo)
	router.GET("/todos/:todoId", server.getTodo)

	// TODO: Get todo attachment metadata
	router.GET("/todos/:todoId/attachments", server.getTodoAttachmentMetadata)

	// TODO: Get todo attachment
	router.GET("/todos/:todoId/attachments/:attachmentId", server.getTodoAttachment)
}

func (server *Server) setupCreateResourceRouters(router *gin.Engine) {
	// Create todo
	router.POST("/todos", server.createTodo)

	// TODO: Create attachments
	router.POST("/todos/:todoId/attachments", server.uploadTodoAttachments)
}

func (server *Server) setupUpdateResourceRouters(router *gin.Engine) {
	// Update todo title
	router.POST("/todos/:todoId/title", server.updateTodoTitle)

	// Update todo status
	router.POST("/todos/:todoId/status", server.updateTodoStatus)
}

func (server *Server) setupDeleteResourceRouters(router *gin.Engine) {
	// TODO: Delete todo attachment
	router.DELETE("/todos/:todoId/attachments/:attachmentId", server.deleteTodoAttachment)

	// Delete todo
	router.DELETE("/todos/:todoId", server.deleteTodo)
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
