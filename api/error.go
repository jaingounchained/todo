package api

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

var (
	todoIDInvalidError                         = errors.New("Invalid todoId; todoId must be a valid integer > 0")
	todoStatusInvalidError                     = errors.New("Invalid todoTitle; todoTitle must be either 'complete' or 'incomplete'")
	uploadAttachmentAPIContentLengthLimitError = fmt.Errorf("Upload attachment API request content lenght must be less than %d Mibs", MaxContentLength/1024/1024)
	invalidHeaderContentTypeError              = fmt.Errorf("Request %s isn't %s", ContentType, MultipartFormDataHeader)
	attachmentKeyEmptyError                    = fmt.Errorf("No files present in '%s' key", UploadAttachmentFormFileKey)
	noAttachmentsPresentForTheTodo             = errors.New("No attachments present for the todo")
	updateTodoTitleStatusInvalidBodyError      = errors.New("At least one of 'title' or 'status' must be provided for update")
	// todoTitleInvalidError                      = errors.New("Invalid todoTitle; todoTitle must be a string of length < 256")
	// pageIDInvalidError                         = errors.New("Invalid pageId; pageId must be a valid integer > 0")
	// pageSizeInvalidError                       = errors.New("Invalid pageSize; pageSize must be a valid integer >= 5 & <= 10")
	// attachmentIDInvalidError                   = errors.New("Invalid attachmentId; attachmentId must be a valid integer > 0")
)

type todoAttachmentLimitReachedError error

func newTodoAttachmentLimitReachedError(attachments int) error {
	return fmt.Errorf("%d attachments per todo allowed; %d files already present for the todo", TodoAttachmentLimit, attachments)
}

type ResourceNotFoundError struct {
	resourceType string
	id           int64
}

func (e *ResourceNotFoundError) Error() string {
	return fmt.Sprintf("Resource Type: %s not found with ID: %d within the system", e.resourceType, e.id)
}

type attachmentNotAssociatedWithTodoError error

func newAttachmentNotAssociatedWithTodoError(todoID, attachmentID int64) attachmentNotAssociatedWithTodoError {
	return fmt.Errorf("attachment %d is not associated with the todo %d", attachmentID, todoID)
}

type invalidMimeTypeError error

func newInvalidMimeTypeError(filename, mimeType string) invalidMimeTypeError {
	return fmt.Errorf("%s file of invalid mime type: %s", filename, mimeType)
}

type fileSizeTooLargeError error

func newFileSizeTooLargeError(filename string, fileSizeLimit int64) fileSizeTooLargeError {
	return fmt.Errorf("%s file size too large, must e < %d MB", filename, fileSizeLimit/1024/1024)
}

func NewHTTPError(ctx *gin.Context, status int, err error) {
	er := &HTTPError{
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}

type HTTPError struct {
	Message string `json:"message" example:"generic error"`
}

func (e *HTTPError) Error() string {
	return e.Error()
}
