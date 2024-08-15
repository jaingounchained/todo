package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/jaingounchained/todo/db/sqlc"
	storage "github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// Server serves HTTP requests for todo service
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	storage    storage.Storage
	router     *gin.Engine
}

// NewGinHandler creates a new HTTP server and setup routing
func NewGinHandler(config util.Config, store db.Store, storage storage.Storage, l *zap.Logger) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		storage:    storage,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("todoStatus", validTodoStatus)
	}

	server.setupRouter(l)
	return server, nil
}

func (server *Server) setupRouter(l *zap.Logger) {
	router := gin.New()
	gin.Default()

	if l == nil {
		router.Use(gin.Logger())
	} else {
		router.Use(logger(l))
	}

	// health check router
	router.GET("/health", server.health)

	server.setupUserRouters(router)

	server.setupGetResourceRouters(router)
	server.setupCreateResourceRouters(router)
	server.setupUpdateResourceRouters(router)
	server.setupDeleteResourceRouters(router)

	server.setupSwagger(router)

	server.router = router
}

func (server *Server) setupSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (server *Server) setupUserRouters(router *gin.Engine) {
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
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
	// Update todo title or status
	router.PATCH("/todos/:todoId", server.updateTodoTitleStatus)
}

func (server *Server) setupDeleteResourceRouters(router *gin.Engine) {
	// TODO: Delete todo attachment
	router.DELETE("/todos/:todoId/attachments/:attachmentId", server.deleteTodoAttachment)

	// Delete todo
	router.DELETE("/todos/:todoId", server.deleteTodo)
}

// Start runs the HTTP server on a specific address
func (server *Server) HttpServer(address string) *http.Server {
	return &http.Server{
		Addr:    address,
		Handler: server.router,
	}
}
