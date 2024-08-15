package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/token"
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
//	@Param			todoId	path		int	true	"Todo ID"	minimum(1)
//	@Success		200		{object}	db.Todo
//	@Failure		400
//	@Failure		401
//	@Failure		404
//	@Failure		500
//	@Security		AccessTokenAuth
//	@Router			/todos/{todoId} [get]
func (server *Server) getTodo(ctx *gin.Context) {
	var req getTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, todoIDInvalidError)
		return
	}

	todo := server.fetchTodoAndHandleErrors(ctx, req.TodoID)
	if todo == nil {
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
//	@Param			todo	body		createTodoRequest	true	"Todo title"
//	@Success		200		{object}	db.Todo
//	@Failure		401
//	@Failure		400
//	@Failure		500
//	@Security		AccessTokenAuth
//	@Router			/todos [post]
func (server *Server) createTodo(ctx *gin.Context) {
	var req createTodoRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	result, err := server.store.CreateTodoTx(ctx, db.CreateTodoTxParams{
		TodoOwner: authPayload.Username,
		TodoTitle: req.Title,
		Storage:   server.storage,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
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
//	@Produce		json
//
//	@Param			pageId		query	int	true	"page ID"	minimum(1)
//	@Param			pageSize	query	int	true	"page size"	minimum(5)	maximum(10)
//
//	@Success		200			{array}	[]db.Todo
//	@Failure		400
//	@Failure		401
//	@Failure		500
//	@Security		AccessTokenAuth
//	@Router			/todos [get]
func (server *Server) listTodo(ctx *gin.Context) {
	var req listTodoRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		// TODO: Improve error message
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)

	todos, err := server.store.ListTodos(ctx, db.ListTodosParams{
		Owner:  authPayload.Username,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, todos)
}

type updateTodoRequestURIParams struct {
	getTodoRequest
}

type updateTodoRequestBody struct {
	Title  *string `json:"title" binding:"omitempty,max=255"`
	Status *string `json:"status" binding:"omitempty,todoStatus"`
}

// updateTodoTitleStatus godoc
//
//	@Summary		Updated the todo title/status
//	@Description	Updates the todo title/status
//	@Tags			todos
//	@Accept			json
//	@Produce		json
//	@Param			todoId	path		int						true	"Todo ID"	minimum(1)
//	@Param			todo	body		updateTodoRequestBody	true	"Todo title/status"
//	@Success		200		{object}	db.Todo
//	@Failure		400
//	@Failure		404
//	@Failure		401
//	@Failure		500
//	@Security		AccessTokenAuth
//	@Router			/todos/{todoId} [patch]
func (server *Server) updateTodoTitleStatus(ctx *gin.Context) {
	// Bind ID
	var reqURIParams updateTodoRequestURIParams
	if err := ctx.ShouldBindUri(&reqURIParams); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, todoIDInvalidError)
		return
	}

	todo := server.fetchTodoAndHandleErrors(ctx, reqURIParams.TodoID)
	if todo == nil {
		return
	}

	// Bind Title and Status
	var reqBody updateTodoRequestBody
	if err := ctx.ShouldBindJSON(&reqBody); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	// Update at least one of title or status
	if reqBody.Title == nil && reqBody.Status == nil {
		NewHTTPError(ctx, http.StatusBadRequest, updateTodoTitleStatusInvalidBodyError)
		return
	}

	updatedTodo, err := server.store.UpdateTodoTitleStatus(ctx, db.UpdateTodoTitleStatusParams{
		ID:     reqURIParams.TodoID,
		Title:  reqBody.Title,
		Status: reqBody.Status,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, updatedTodo)
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
//	@Param			todoId	path	int	true	"Todo ID"	minimum(1)
//	@Success		200
//	@Failure		400
//	@Failure		404
//	@Failure		401
//	@Failure		500
//	@Security		AccessTokenAuth
//	@Router			/todos/{todoId} [delete]
func (server *Server) deleteTodo(ctx *gin.Context) {
	var req deleteTodoRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, todoIDInvalidError)
		return
	}

	todo := server.fetchTodoAndHandleErrors(ctx, req.TodoID)
	if todo == nil {
		return
	}

	err := server.store.DeleteTodoTx(ctx, db.DeleteTodoTxParams{
		TodoID:  req.TodoID,
		Storage: server.storage,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}

func (server *Server) fetchTodoAndHandleErrors(ctx *gin.Context, todoID int64) *db.Todo {
	todo, err := server.store.GetTodo(ctx, todoID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewHTTPError(ctx, http.StatusNotFound, &ResourceNotFoundError{
				resourceType: "todo",
				id:           todoID,
			})
			return nil
		}

		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return nil
	}

	authPayload := ctx.MustGet(authorizationPayloadKey).(*token.Payload)
	if todo.Owner != authPayload.Username {
		err := errors.New("todo doesn't belog to the authenticated user")
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return nil
	}

	return &todo
}
