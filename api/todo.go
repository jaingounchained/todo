package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
)

type createTodoRequest struct {
	Title string `json:"title" binding:"required,max=255"`
}

func (server *Server) createTodo(ctx *gin.Context) {
	var req createTodoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	result, err := server.store.CreateTodoTx(ctx, db.CreateTodoTxParams{
		TodoTitle: req.Title,
		Storage:   server.storage,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, result.Todo)
}

type getTodoRequest struct {
	TodoID int64 `uri:"todoId" binding:"required,min=1"`
}

func (server *Server) getTodo(ctx *gin.Context) {
	var req getTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	todo, err := server.store.GetTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type listTodoRequest struct {
	PageID   int32 `form:"page_id" binding:"required,min=1"`
	PageSize int32 `form:"page_size" binding:"required,min=5,max=10"`
}

func (server *Server) listTodo(ctx *gin.Context) {
	var req listTodoRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	todos, err := server.store.ListTodos(ctx, db.ListTodosParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, todos)
}

type updateTodoRequestURIParams struct {
	getTodoRequest
}

type updateTodoTitleRequestBody struct {
	Title string `json:"title" binding:"required,max=255"`
}

func (server *Server) updateTodoTitle(ctx *gin.Context) {
	// Bind ID
	var reqURIParams updateTodoRequestURIParams
	if err := ctx.ShouldBindUri(&reqURIParams); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Bind Title
	var reqBody updateTodoTitleRequestBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	todo, err := server.store.UpdateTodoTitle(ctx, db.UpdateTodoTitleParams{
		ID:    reqURIParams.TodoID,
		Title: reqBody.Title,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type updateTodoStatusRequestBody struct {
	Status string `json:"status" binding:"required,todoStatus"`
}

func (server *Server) updateTodoStatus(ctx *gin.Context) {
	// Bind ID
	var reqURIParams updateTodoRequestURIParams
	if err := ctx.ShouldBindUri(&reqURIParams); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Bind Title
	var reqBody updateTodoStatusRequestBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	todo, err := server.store.UpdateTodoStatus(ctx, db.UpdateTodoStatusParams{
		ID:     reqURIParams.TodoID,
		Status: reqBody.Status,
	})
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type deleteTodoRequest struct {
	getTodoRequest
}

func (server *Server) deleteTodo(ctx *gin.Context) {
	var req deleteTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	err := server.store.DeleteTodoTx(ctx, db.DeleteTodoTxParams{
		TodoID:  req.TodoID,
		Storage: server.storage,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, nil)
}
