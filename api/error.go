package api

import "github.com/gin-gonic/gin"

const (
	errorJSONKey = "error"
)

func errorResponse(err error) gin.H {
	return gin.H{
		errorJSONKey: err.Error(),
	}
}
