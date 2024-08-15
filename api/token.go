package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
)

type renewAccessTokenRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string    `json:"accessToken"`
	AccessTokenExpiresAt time.Time `json:"accessTokenExpiredAt"`
}

// TODO: Add swaggo comments; research how to add token authentication in other api docs
func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	refreshPayload, err := server.tokenMaker.VerifyToken(req.RefreshToken)
	if err != nil {
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	session, err := server.store.GetSession(ctx, refreshPayload.ID)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewHTTPError(ctx, http.StatusNotFound, err)
			return
		}
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	if session.IsBlocked {
		err := errors.New("blocked session")
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	if session.Username != refreshPayload.Username {
		err := errors.New("incorrect session user")
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	if session.RefreshToken != req.RefreshToken {
		err := errors.New("mismatched session token")
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	if time.Now().After(session.ExpiresAt) {
		err := errors.New("expired session")
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		refreshPayload.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, renewAccessTokenResponse{
		AccessToken:          accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,
	})
}
