package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/util"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum" example:"jaingounchained"`
	Password string `json:"password" binding:"required,min=6" example:"weak_password"`
	FullName string `json:"fullName" binding:"required" example:"Jain Bhavya"`
	Email    string `json:"email" binding:"required,email" example:"jain@jaingounchained.com"`
}

type createUserResponse struct {
	Username string `json:"username" example:"jaingounchained"`
	FullName string `json:"fullName" example:"Jain Bhavya"`
	Email    string `json:"email" example:"jain@jaingounchained.com"`
}

// createUser godoc
//
//	@Summary		Creates a User
//	@Description	Creates a User
//	@Tags			users
//	@Accept			json
//	@Produce		json
//	@Param			user	body		createUserRequest	true	"User details"
//	@Success		200		{object}	createUserResponse
//	@Failure		400
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

	ctx.JSON(http.StatusOK, createUserResponse{
		Username: user.Username,
		FullName: user.FullName,
		Email:    user.Email,
	})
}
