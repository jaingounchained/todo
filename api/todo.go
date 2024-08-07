package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
)

type getTodoRequest struct {
	TodoID int64 `uri:"todoId" binding:"required,min=1"`
}

// GetTodo godoc
//
//	@Summary		Returns a Todo
//	@Description	Get todo by TodoID
//	@Tags			todos
//	@Produce		json
//	@Param			id path int true "Todo ID" minimum(1)
//	@Success		200	{object}	db.Todo
//	@Failure		400 {object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos/{id} [get]
func (server *Server) getTodo(ctx *gin.Context) {
	var req getTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	todo, err := server.store.GetTodo(ctx, req.TodoID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type createTodoRequest struct {
	Title string `json:"title" binding:"required,max=255"`
}

// createTodo godoc
//
//	@Summary		Creates a Todo
//	@Description	Creates a todo with the specified title
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//	@Param			todo	body	createTodoRequest true "Todo title"
//	@Success		200	{object}	db.Todo
//	@Failure		400 {object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos [post]
func (server *Server) createTodo(ctx *gin.Context) {
	var req createTodoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	result, err := server.store.CreateTodoTx(ctx, db.CreateTodoTxParams{
		TodoTitle: req.Title,
		Storage:   server.storage,
	})
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, result.Todo)
}

type listTodoRequest struct {
	PageID   int32 `form:"pageId" binding:"required,min=1"`
	PageSize int32 `form:"pageSize" binding:"required,min=5,max=10"`
}

// listTodo godoc
//
//	@Summary		List todos
//	@Description	List todos based on page ID and page size
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//
// @Param        pageId    query     int  true  "page ID"  minimum(1)
// @Param        pageSize    query     int  true  "page size"  minimum(5) maximum(10)
//
//	@Success		200	{array}		db.Todo
//	@Failure		400 {object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos [get]
func (server *Server) listTodo(ctx *gin.Context) {
	var req listTodoRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	todos, err := server.store.ListTodos(ctx, db.ListTodosParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
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

// updateTodoTitle godoc
//
//	@Summary		Updated the todo title
//	@Description	Updates the todo title
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Param			todo	body	updateTodoTitleRequestBody true "Todo title"
//	@Success		200	{object}	db.Todo
//	@Failure		400 {object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos/{id}/title [post]
func (server *Server) updateTodoTitle(ctx *gin.Context) {
	// Bind ID
	var reqURIParams updateTodoRequestURIParams
	if err := ctx.ShouldBindUri(&reqURIParams); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	// Bind Title
	var reqBody updateTodoTitleRequestBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	todo, err := server.store.UpdateTodoTitle(ctx, db.UpdateTodoTitleParams{
		ID:    reqURIParams.TodoID,
		Title: reqBody.Title,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type updateTodoStatusRequestBody struct {
	Status string `json:"status" binding:"required,todoStatus"`
}

// updateTodoStatus godoc
//
//	@Summary		Updated the todo status
//	@Description	Updates the todo status
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Param			todo	body	updateTodoStatusRequestBody true "Todo status"
//	@Success		200	{object}	db.Todo
//	@Failure		400 {object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos/{id}/status [post]
func (server *Server) updateTodoStatus(ctx *gin.Context) {
	// Bind ID
	var reqURIParams updateTodoRequestURIParams
	if err := ctx.ShouldBindUri(&reqURIParams); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	// Bind Title
	var reqBody updateTodoStatusRequestBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	todo, err := server.store.UpdateTodoStatus(ctx, db.UpdateTodoStatusParams{
		ID:     reqURIParams.TodoID,
		Status: reqBody.Status,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, todo)
}

type deleteTodoRequest struct {
	getTodoRequest
}

// deleteTodo godoc
//
//	@Summary		Deletes a Todo
//	@Description	Delete todo by TodoID
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Success		200	{object} nil
//	@Failure		400 {object}	HTTPError
//	@Failure		404	{object}	HTTPError
//	@Failure		500	{object}	HTTPError
//	@Router			/todos/{id} [delete]
func (server *Server) deleteTodo(ctx *gin.Context) {
	var req deleteTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	err := server.store.DeleteTodoTx(ctx, db.DeleteTodoTxParams{
		TodoID:  req.TodoID,
		Storage: server.storage,
	})
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}
