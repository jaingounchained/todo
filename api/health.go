package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// health        godoc
//	@Description	Returns server health
//	@Tags			health
//	@Produce		text/plain
//	@Success		200
//	@Router			/health [get]
func (server *Server) health(ctx *gin.Context) {
	// check if db and other conns are working or not
	// singleton health function so all packages can use
	ctx.JSON(http.StatusOK, "OK")
}
