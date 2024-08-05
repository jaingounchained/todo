package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func logger(logger *zap.Logger) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		logger.Info("Incoming request",
			zap.Int("status", ctx.Writer.Status()),
			zap.Int64("content_length", ctx.Request.ContentLength),
			zap.String("method", ctx.Request.Method),
			zap.String("query", ctx.Request.URL.RawQuery),
			zap.String("path", ctx.Request.URL.Path),
			zap.String("ip", ctx.ClientIP()),
			zap.String("user-agent", ctx.Request.UserAgent()),
			zap.String("errors", ctx.Errors.ByType(gin.ErrorTypePrivate).String()),
			zap.Duration("elapsed", time.Since(start)),
		)
	}
}
