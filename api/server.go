package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	db "github.com/jaingounchained/todo/db/sqlc"
	storage "github.com/jaingounchained/todo/storage"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// Server serves HTTP requests for todo service
type Server struct {
	config     util.Config
	store      db.Store
	tokenMaker token.Maker
	storage    storage.Storage
	Router     *gin.Engine
}

// NewGinHandler creates a new HTTP server and setup routing
func NewGinHandler(config util.Config, store db.Store, storage storage.Storage) (*Server, error) {
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

	server.setupRouter()
	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.New()
	gin.Default()

	router.Use(loggerMiddleware())

	server.setupSwagger(router)

	// health check router
	router.GET("/health", server.health)

	server.setupUserRouters(router)

	authRouterGroup := router.Group("/").Use(authMiddleware(server.tokenMaker))

	server.setupGetResourceRouters(authRouterGroup)
	server.setupCreateResourceRouters(authRouterGroup)
	server.setupUpdateResourceRouters(authRouterGroup)
	server.setupDeleteResourceRouters(authRouterGroup)

	server.Router = router
}

func (server *Server) setupSwagger(router *gin.Engine) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}

func (server *Server) setupUserRouters(router *gin.Engine) {
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/tokens/renewAccess", server.renewAccessToken)
}

func (server *Server) setupGetResourceRouters(authRouterGroup gin.IRoutes) {
	authRouterGroup.GET("/todos", server.listTodo)
	authRouterGroup.GET("/todos/:todoId", server.getTodo)

	authRouterGroup.GET("/todos/:todoId/attachments", server.getTodoAttachmentMetadata)

	authRouterGroup.GET("/todos/:todoId/attachments/:attachmentId", server.getTodoAttachment)
}

func (server *Server) setupCreateResourceRouters(authRouterGroup gin.IRoutes) {
	authRouterGroup.POST("/todos", server.createTodo)

	authRouterGroup.POST("/todos/:todoId/attachments", server.uploadTodoAttachments)
}

func (server *Server) setupUpdateResourceRouters(authRouterGroup gin.IRoutes) {
	authRouterGroup.PATCH("/todos/:todoId", server.updateTodoTitleStatus)
}

func (server *Server) setupDeleteResourceRouters(authRouterGroup gin.IRoutes) {
	authRouterGroup.DELETE("/todos/:todoId/attachments/:attachmentId", server.deleteTodoAttachment)

	authRouterGroup.DELETE("/todos/:todoId", server.deleteTodo)
}
