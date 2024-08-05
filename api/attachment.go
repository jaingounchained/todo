package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
)

const (
	MaxContentLength          = 10 << 20
	UploadAttachmentFieldName = "attachments"
)

type uploadTodoAttachmentsRequest struct {
	getTodoRequest
}

// uploadTodoAttachments godoc
//
//	@Summary		Upload attachments
//	@Description	Upload attachments for the corresponding todo
//	@Tags			attachments
//	@Accept			multipart/form-data
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Param			attachments	formData	[]file	true	"attachments"
//	@Success		200		{object}	nil
//	@Failure		403		{object}	HTTPError "< 5 attachments allowed per todo"
//	@Failure		413		{object}	HTTPError "< 2 MB per file"
//	@Failure		415		{object}	HTTPError "unsuppoerted attachment"
//	@Failure		404		{object}	HTTPError
//	@Failure		400		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/todos/{id}/attachments [post]
func (server *Server) uploadTodoAttachments(ctx *gin.Context) {
	// Extract todo
	var req uploadTodoAttachmentsRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	todo, err := server.store.GetTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Return error if already number of attachments capped
	if todo.FileCount >= 5 {
		// cannot upload more attachments
		NewError(ctx, http.StatusForbidden, errors.New("Cannot upload more attachments"))
		return
	}

	// Check form data is less than maximum specified bytes
	if ctx.Request.ContentLength > MaxContentLength {
		NewError(ctx, http.StatusRequestEntityTooLarge, errors.New("Request body more than 10 MBs"))
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	files, ok := form.File[UploadAttachmentFieldName]
	if !ok {
		NewError(ctx, http.StatusBadRequest, errors.New("no file present in 'attachments' key"))
		return
	}

	// Validate length of files; should be less than 5 - todo's already uploaded items
	if len(files)+int(todo.FileCount) > 5 {
		NewError(ctx, http.StatusRequestEntityTooLarge, errors.New("Not allowed to upload more than 5 files, already present x files"))
		return
	}

	// validate individual file type
	for _, file := range files {
		if err := validateMimeType(file.Header); err != nil {
			NewError(ctx, http.StatusUnsupportedMediaType, err)
			return
		}

		if err := validateFileSize(file.Size); err != nil {
			NewError(ctx, http.StatusRequestEntityTooLarge, errors.New("Large file"))
			return
		}
	}

	fileContents := make(map[string][]byte)
	for _, file := range files {
		multiPartFile, err := file.Open()
		if err != nil {
			NewError(ctx, http.StatusInternalServerError, err)
			return
		}

		b := make([]byte, file.Size)
		if _, err = multiPartFile.Read(b); err != nil {
			NewError(ctx, http.StatusInternalServerError, err)
			return
		}

		fileContents[filepath.Base(file.Filename)] = b
	}

	err = server.store.UploadAttachmentTx(ctx, db.UploadAttachmentTxParams{
		Todo:         todo,
		FileContents: fileContents,
		Storage:      server.storage,
	})
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}

type getTodoAttachmentRequest struct {
	getTodoRequest
	AttachmentID int64 `uri:"attachmentId" binding:"required,min=1"`
}

// getTodoAttachment godoc
//
//	@Summary		Get attachments
//	@Description	Get attachment for the corresponding todo
//	@Tags			attachments
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Param			attachmentId	path		int	true	"attachment ID"          minimum(1)
//	@Success		200		{object}	nil
//	@Failure		403		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		400		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/todos/{id}/attachments/{attachmentId} [get]
func (server *Server) getTodoAttachment(ctx *gin.Context) {
	var req getTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	// Query attachment metadata
	attachment, err := server.store.GetAttachment(ctx, req.AttachmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	if attachment.TodoID != req.TodoID {
		NewError(ctx, http.StatusForbidden, errors.New("attachment doesn't belong to todo"))
		return
	}

	b, err := server.storage.GetFileContents(ctx, req.TodoID, attachment.StorageFilename)
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Set the Content-Type header for the response
	ctx.Header("Content-Type", "application/octet-stream")
	// Set the Content-Disposition header to instruct the client to treat the response as a file to be downloaded
	ctx.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", attachment.OriginalFilename))
	// Write the content to the response body and set the response Content-Type
	ctx.Data(http.StatusOK, "application/octet-stream", b)
}

type getTodoAttachmentMetadataRequest struct {
	getTodoRequest
}

type getTodoAttachmentMetadataResponse struct {
	ID       int64  `json:"id"`
	TodoID   int64  `json:"todo_id"`
	Filename string `json:"filename"`
}

// getTodoAttachmentMetadata godoc
//
//	@Summary		Get attachments metadata
//	@Description	Get attachment metadata for the corresponding todo
//	@Tags			attachments
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Success		200		{object}	getTodoAttachmentMetadataResponse
//	@Failure		403		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		400		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/todos/{id}/attachments [get]
func (server *Server) getTodoAttachmentMetadata(ctx *gin.Context) {
	var req getTodoAttachmentMetadataRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	_, err := server.store.GetTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	// Query attachment metadata
	attachments, err := server.store.ListAttachmentOfTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	resp := make([]getTodoAttachmentMetadataResponse, 0)
	for _, attachment := range attachments {
		resp = append(resp, getTodoAttachmentMetadataResponse{
			ID:       attachment.ID,
			TodoID:   attachment.TodoID,
			Filename: attachment.OriginalFilename,
		})
	}

	ctx.JSON(http.StatusOK, resp)
}

type deleteTodoAttachmentRequest struct {
	getTodoRequest
	AttachmentID int64 `uri:"attachmentId" binding:"required,min=1"`
}

// deleteTodoAttachment godoc
//
//	@Summary		Delete attachment
//	@Description	Delete attachment for the corresponding todo
//	@Tags			attachments
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"          minimum(1)
//	@Param			attachmentId	path		int	true	"attachment ID"          minimum(1)
//	@Success		200		{object}	nil
//	@Failure		413		{object}	HTTPError
//	@Failure		404		{object}	HTTPError
//	@Failure		400		{object}	HTTPError
//	@Failure		500		{object}	HTTPError
//	@Router			/todos/{id}/attachments/{attachmentId} [delete]
func (server *Server) deleteTodoAttachment(ctx *gin.Context) {
	var req deleteTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewError(ctx, http.StatusBadRequest, err)
		return
	}

	attachment, err := server.store.GetAttachment(ctx, req.AttachmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			NewError(ctx, http.StatusNotFound, err)
			return
		}

		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	if attachment.TodoID != req.TodoID {
		NewError(ctx, http.StatusForbidden, errors.New("attachment doesn't belong to todo"))
		return
	}

	err = server.store.DeleteAttachmentTx(ctx, db.DeleteAttachmentTxParams{
		TodoID:     req.TodoID,
		Attachment: attachment,
		Storage:    server.storage,
	})
	if err != nil {
		NewError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}
