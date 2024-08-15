package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/util"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum" example:"jaingounchained"`
	Password string `json:"password" binding:"required,min=6" example:"weak_password"`
	FullName string `json:"fullName" binding:"required" example:"Jain Bhavya"`
	Email    string `json:"email" binding:"required,email" example:"jain@jaingounchained.com"`
}

type userResponse struct {
	Username string `json:"username" example:"jaingounchained"`
	FullName string `json:"fullName" example:"Jain Bhavya"`
	Email    string `json:"email" example:"jain@jaingounchained.com"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	}
}

// createUser godoc
//
//	@Summary		Creates a User
//	@Description	Creates a User
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		createUserRequest	true	"User details"
//	@Success		200		{object}	userResponse
//	@Failure		400
//	@Failure		403
//	@Failure		500
//	@Router			/users [post]
func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	user, err := server.store.CreateUser(ctx, db.CreateUserParams{
		Username:       req.Username,
		HashedPassword: hashedPassword,
		FullName:       req.FullName,
		Email:          req.Email,
	})
	if err != nil {
		// TODO: Differenciate error between unique username and email violation
		if db.ErrorCode(err) == db.UniqueViolation {
			NewHTTPError(ctx, http.StatusForbidden, err)
			return
		}
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, newUserResponse(user))
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum" example:"jaingounchained"`
	Password string `json:"password" binding:"required,min=6" example:"weak_password"`
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"sessionId"`
	AccessToken           string       `json:"accessToken"`
	AccessTokenExpiresAt  time.Time    `json:"accessTokenExpiredAt"`
	RefreshToken          string       `json:"refreshToken"`
	RefreshTokenExpiresAt time.Time    `json:"refreshTokenExpiredAt"`
	User                  userResponse `json:"user"`
}

// loginUser godoc
//
//	@Summary		User login
//	@Description	Returns an access token for accessing user resources
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			loginCredentials	body		loginUserRequest	true	"User login credentials"
//	@Success		200					{object}	loginUserResponse
//	@Failure		400
//	@Failure		401
//	@Failure		403
//	@Failure		404
//	@Failure		500
//	@Router			/users/login [post]
func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		NewHTTPError(ctx, http.StatusBadRequest, err)
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if errors.Is(err, db.ErrRecordNotFound) {
			NewHTTPError(ctx, http.StatusNotFound, err)
			return
		}
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		NewHTTPError(ctx, http.StatusUnauthorized, err)
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		NewHTTPError(ctx, http.StatusInternalServerError, err)
		return
	}

	ctx.JSON(http.StatusOK, loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	})
}
