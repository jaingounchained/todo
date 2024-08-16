package gapi

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	mockdb "github.com/jaingounchained/todo/db/mock"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/pb"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func TestUpdateUserAPI(t *testing.T) {
	user, _ := randomUser(t)

	newName := util.RandomOwner()
	newEmail := util.RandomEmail()

	testCases := []struct {
		name          string
		body          *pb.UpdateUserRequest
		buildStubs    func(store *mockdb.MockStore)
		buildContext  func(t *testing.T, tokenMaker token.Maker) context.Context
		checkResponse func(t *testing.T, res *pb.UpdateUserResponse, err error)
	}{
		{
			name: "OK",
			body: &pb.UpdateUserRequest{
				Username: user.Username,
				FullName: &newName,
				Email:    &newEmail,
			},
			buildStubs: func(store *mockdb.MockStore) {
				arg := db.UpdateUserParams{
					Username: user.Username,
					FullName: &newName,
					Email:    &newEmail,
				}

				updatedUser := db.User{
					Username:          user.Username,
					HashedPassword:    user.HashedPassword,
					FullName:          newName,
					Email:             newEmail,
					PasswordChangedAt: user.PasswordChangedAt,
					CreatedAt:         user.CreatedAt,
					IsEmailVerified:   user.IsEmailVerified,
				}
				store.EXPECT().
					UpdateUser(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(updatedUser, nil)
			},
			buildContext: func(t *testing.T, tokenMaker token.Maker) context.Context {
				return newContextWithBearerToken(t, tokenMaker, user.Username, time.Minute)
			},
			checkResponse: func(t *testing.T, res *pb.UpdateUserResponse, err error) {
				require.NoError(t, err)
				require.NotNil(t, res)
				updatedUser := res.GetUser()
				require.Equal(t, user.Username, updatedUser.Username)
				require.Equal(t, newName, updatedUser.FullName)
				require.Equal(t, newEmail, updatedUser.Email)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			storeCtrl := gomock.NewController(t)
			defer storeCtrl.Finish()
			store := mockdb.NewMockStore(storeCtrl)

			tc.buildStubs(store)

			server := newTestServer(t, store, nil)
			ctx := tc.buildContext(t, server.tokenMaker)
			res, err := server.UpdateUser(ctx, tc.body)
			tc.checkResponse(t, res, err)
		})
	}
}
