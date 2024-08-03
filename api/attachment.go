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

type uploadTodoAttachmentsRequest struct {
	getTodoRequest
}

func (server *Server) uploadTodoAttachments(ctx *gin.Context) {
	// Extract todo
	var req uploadTodoAttachmentsRequest
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

	// Return error if already number of attachments capped
	if todo.FileCount >= 5 {
		// cannot upload more attachments
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("Cannot upload more attachments")))
		return
	}

	// Check form data is less than maximum specified bytes
	form, err := ctx.MultipartForm()
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	files, ok := form.File["attachments"]
	if !ok {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("no file present in 'attachments' key")))
		return
	}

	// Validate length of files; should be less than 5 - todo's already uploaded items
	if len(files)+int(todo.FileCount) > 5 {
		ctx.JSON(http.StatusRequestEntityTooLarge, errorResponse(errors.New("Not allowed to upload more than 5 files, already present x files")))
		return
	}

	// validate individual file type
	for _, file := range files {
		if err := validateMimeType(file.Header); err != nil {
			ctx.JSON(http.StatusUnsupportedMediaType, errorResponse(err))
			return
		}

		if err := validateFileSize(file.Size); err != nil {
			ctx.JSON(http.StatusRequestEntityTooLarge, errorResponse(errors.New("Large file")))
			return
		}
	}

	fileContents := make(map[string][]byte)
	for _, file := range files {
		multiPartFile, err := file.Open()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
			return
		}

		b := make([]byte, file.Size)
		if _, err = multiPartFile.Read(b); err != nil {
			ctx.JSON(http.StatusInternalServerError, errorResponse(err))
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
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, nil)
}

type getTodoAttachmentRequest struct {
	getTodoRequest
	AttachmentID int64 `uri:"attachmentId" binding:"required,min=1"`
}

func (server *Server) getTodoAttachment(ctx *gin.Context) {
	var req getTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// Query attachment metadata
	attachment, err := server.store.GetAttachment(ctx, req.AttachmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if attachment.TodoID != req.TodoID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("attachment doesn't belong to todo")))
		return
	}

	b, err := server.storage.GetFileContents(ctx, req.TodoID, attachment.StorageFilename)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
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

func (server *Server) getTodoAttachmentMetadata(ctx *gin.Context) {
	var req getTodoAttachmentMetadataRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	_, err := server.store.GetTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	// Query attachment metadata
	attachments, err := server.store.ListAttachmentOfTodo(ctx, req.TodoID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
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

func (server *Server) deleteTodoAttachment(ctx *gin.Context) {
	var req deleteTodoAttachmentRequest
	if err := ctx.ShouldBindUri(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	attachment, err := server.store.GetAttachment(ctx, req.AttachmentID)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}

		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	if attachment.TodoID != req.TodoID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("attachment doesn't belong to todo")))
		return
	}

	err = server.store.DeleteAttachmentTx(ctx, db.DeleteAttachmentTxParams{
		TodoID:     req.TodoID,
		Attachment: attachment,
		Storage:    server.storage,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, nil)
}
