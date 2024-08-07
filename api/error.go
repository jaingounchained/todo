package api

import (
	"errors"
	"fmt"

	"github.com/gin-gonic/gin"
)

var (
	todoIDInvalidError                         = errors.New("Invalid todoId; todoId must be a valid integer > 0")
	todoTitleInvalidError                      = errors.New("Invalid todoTitle; todoTitle must be a string of length < 256")
	todoStatusInvalidError                     = errors.New("Invalid todoTitle; todoTitle must be either 'complete' or 'incomplete'")
	pageIDInvalidError                         = errors.New("Invalid pageId; pageId must be a valid integer > 0")
	pageSizeInvalidError                       = errors.New("Invalid pageSize; pageSize must be a valid integer >= 5 & <= 10")
	attachmentIDInvalidError                   = errors.New("Invalid attachmentId; attachmentId must be a valid integer > 0")
	uploadAttachmentAPIContentLengthLimitError = fmt.Errorf("Upload attachment API request content lenght must be less than %d Mibs", MaxContentLength/1024/1024)
	invalidHeaderContentTypeError              = fmt.Errorf("Request %s isn't %s", ContentType, MultipartFormDataHeader)
	attachmentKeyEmptyError                    = fmt.Errorf("No files present in '%s' key", UploadAttachmentFormFileKey)
	noAttachmentsPresentForTheTodo             = errors.New("No attachments present for the todo")
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

func NewHTTPError(ctx *gin.Context, status int, err error) {
	er := HTTPError{
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}

type HTTPError struct {
	Message string `json:"message" example:"generic error"`
}

// Error implements the error interface for ErrInvalidInput.
func (e *HTTPError) Error() string {
	return e.Error()
}
