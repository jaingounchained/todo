package api

import "github.com/gin-gonic/gin"

const (
	errorJSONKey = "error"
)

func NewError(ctx *gin.Context, status int, err error) {
	er := HTTPError{
		Code:  status,
		Error: err.Error(),
	}
	ctx.JSON(status, er)
}

type HTTPError struct {
	Code  int    `json:"code" example:"400"`
	Error string `json:"error" example:"status bad request"`
}
