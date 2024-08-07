package api

import "github.com/gin-gonic/gin"

const (
	errorJSONKey = "error"
)

func NewError(ctx *gin.Context, status int, err error) {
	er := HTTPError{
		Message: err.Error(),
	}
	ctx.JSON(status, er)
}

type HTTPError struct {
	Message string `json:"message" example:"status bad request"`
}

// Error implements the error interface for ErrInvalidInput.
func (e *HTTPError) Error() string {
	return e.Error()
}
