package api

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
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
//	@Param			id			path		int		true	"Todo ID"	minimum(1)
//	@Param			attachments	formData	[]file	true	"attachments"
//	@Success		200
//	@Failure		403
//	@Failure		413
//	@Failure		415
//	@Failure		404
//	@Failure		400
//	@Failure		500
//	@Router			/todos/{id}/attachments [post]
func (server *Server) uploadTodoAttachments(ctx *gin.Context) {
	// Validate Content-Type header
	if strings.TrimSpace(ctx.ContentType()) != MultipartFormDataHeader {
		NewHTTPError(ctx, http.StatusBadRequest, invalidHeaderContentTypeError)
		return
	}

	// Extract todo
	var req uploadTodoAttachmentsRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, todoIDInvalidError)
		return
	}

	todo := server.fetchTodoAndHandleErrors(ctx, req.TodoID)
	if todo == nil {
		return
	}

	// Return error if already number of attachments capped
	if todo.FileCount >= TodoAttachmentLimit {
		NewHTTPError(ctx, http.StatusForbidden, newTodoAttachmentLimitReachedError(TodoAttachmentLimit))
		return
	}

	// Check form data is less than maximum specified bytes
	if ctx.Request.ContentLength > MaxContentLength {
		NewHTTPError(ctx, http.StatusRequestEntityTooLarge, uploadAttachmentAPIContentLengthLimitError)
		return
	}

	form, err := ctx.MultipartForm()
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	files, ok := form.File[UploadAttachmentFormFileKey]
	if !ok {
		NewHTTPError(ctx, http.StatusBadRequest, attachmentKeyEmptyError)
		return
	}

	// Validate length of files; should be less than 5 - todo's already uploaded items
	l := len(files)
	if l+int(todo.FileCount) > 5 {
		NewHTTPError(ctx, http.StatusRequestEntityTooLarge, newTodoAttachmentLimitReachedError(l))
		return
	}

	// validate individual file type
	for _, file := range files {
		if err := validateMimeType(file.Filename, file.Header); err != nil {
			NewHTTPError(ctx, http.StatusUnsupportedMediaType, err)
			return
		}

		if err := validateFileSize(file.Filename, file.Size); err != nil {
			NewHTTPError(ctx, http.StatusRequestEntityTooLarge, err)
			return
		}
	}

	fileContents := make(map[string][]byte)
	for _, file := range files {
		multiPartFile, err := file.Open()
		if err != nil {
			NewHTTPError(ctx, http.StatusInternalServerError, err)
			return
		}

		b := make([]byte, file.Size)
		if _, err = multiPartFile.Read(b); err != nil {
			NewHTTPError(ctx, http.StatusInternalServerError, err)
			return
		}

		fileContents[filepath.Base(file.Filename)] = b
	}

	err = server.store.UploadAttachmentTx(ctx, db.UploadAttachmentTxParams{
		Todo:         *todo,
		FileContents: fileContents,
		Storage:      server.storage,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
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
//	@Param			todoId			path	int	true	"Todo ID"		minimum(1)
//	@Param			attachmentId	path	int	true	"attachment ID"	minimum(1)
//	@Success		200
//	@Failure		403
//	@Failure		404
//	@Failure		400
//	@Failure		500
//	@Router			/todos/{todoId}/attachments/{attachmentId} [get]
func (server *Server) getTodoAttachment(ctx *gin.Context) {
	var req getTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	attachment := server.fetchAttachmentAndHandleErrors(ctx, req.AttachmentID)
	if attachment == nil {
		return
	}

	if attachment.TodoID != req.TodoID {
		NewHTTPError(ctx, http.StatusForbidden, newAttachmentNotAssociatedWithTodoError(req.TodoID, req.AttachmentID))
		return
	}

	b, err := server.storage.GetFileContents(ctx, req.TodoID, attachment.StorageFilename)
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
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
	ID       int64  `json:"attachmentId"`
	TodoID   int64  `json:"todoId"`
	Filename string `json:"filename"`
}

// getTodoAttachmentMetadata godoc
//
//	@Summary		Get attachments metadata
//	@Description	Get attachment metadata for the corresponding todo
//	@Tags			attachments
//	@Accept			json
//	@Produce		json
//	@Param			id	path		int	true	"Todo ID"	minimum(1)
//	@Success		200	{object}	getTodoAttachmentMetadataResponse
//	@Failure		403
//	@Failure		404
//	@Failure		400
//	@Failure		500
//	@Router			/todos/{id}/attachments [get]
func (server *Server) getTodoAttachmentMetadata(ctx *gin.Context) {
	var req getTodoAttachmentMetadataRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, todoIDInvalidError)
		return
	}

	todo := server.fetchTodoAndHandleErrors(ctx, req.TodoID)
	if todo == nil {
		return
	}

	// Query attachment metadata
	attachments, err := server.store.ListAttachmentOfTodo(ctx, req.TodoID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewHTTPError(ctx, http.StatusNotFound, &ResourceNotFoundError{
				resourceType: "attachment",
				id:           req.TodoID,
			})
			return
		}

		NewHTTPError(ctx, http.StatusInternalServerError, err)
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
//	@Param			id				path	int	true	"Todo ID"		minimum(1)
//	@Param			attachmentId	path	int	true	"attachment ID"	minimum(1)
//	@Success		200
//	@Failure		413
//	@Failure		404
//	@Failure		400
//	@Failure		500
//	@Router			/todos/{id}/attachments/{attachmentId} [delete]
func (server *Server) deleteTodoAttachment(ctx *gin.Context) {
	var req deleteTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	attachment := server.fetchAttachmentAndHandleErrors(ctx, req.AttachmentID)
	if attachment == nil {
		return
	}

	if attachment.TodoID != req.TodoID {
		NewHTTPError(ctx, http.StatusForbidden, newAttachmentNotAssociatedWithTodoError(req.TodoID, req.AttachmentID))
		return
	}

	err := server.store.DeleteAttachmentTx(ctx, db.DeleteAttachmentTxParams{
		TodoID:     req.TodoID,
		Attachment: *attachment,
		Storage:    server.storage,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, nil)
}

func (server *Server) fetchAttachmentAndHandleErrors(ctx *gin.Context, attachmentID int64) *db.Attachment {
	attachment, err := server.store.GetAttachment(ctx, attachmentID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewHTTPError(ctx, http.StatusNotFound, &ResourceNotFoundError{
				resourceType: "attachment",
				id:           attachmentID,
			})
			return nil
		}

		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return nil
	}

	return &attachment
}
