package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/jaingounchained/todo/db/mock"
	db "github.com/jaingounchained/todo/db/sqlc"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/assert"
)

func randomTodo(owner string) db.Todo {
	return db.Todo{
		ID:        util.RandomInt(1, 1000),
		Owner:     owner,
		Title:     util.RandomString(10),
		Status:    util.RandomStatus(),
		FileCount: 0,
	}
}

func assertBodyMatchTodo(t *testing.T, body *bytes.Buffer, todo db.Todo) {
	data, err := io.ReadAll(body)
	assert.NoError(t, err)

	var gotTodo db.Todo
	err = json.Unmarshal(data, &gotTodo)
	assert.NoError(t, err)
	assert.Equal(t, gotTodo, todo)
}

func assertBodyMatchTodos(t *testing.T, body *bytes.Buffer, todos []db.Todo) {
	data, err := io.ReadAll(body)
	assert.NoError(t, err)

	var gotTodos []db.Todo
	err = json.Unmarshal(data, &gotTodos)
	assert.NoError(t, err)
	assert.Equal(t, gotTodos, todos)
}

func assertBodyMatchError(t *testing.T, body *bytes.Buffer, expectedError error) {
	data, err := io.ReadAll(body)
	assert.NoError(t, err)

	var gotHTTPError HTTPError
	err = json.Unmarshal(data, &gotHTTPError)

	assert.NoError(t, err)
	assert.NotNil(t, gotHTTPError)
	assert.Equal(t, gotHTTPError.Message, expectedError.Error())
}

func TestGetTodoAPI(t *testing.T) {
	user, _ := randomUser(t)
	role := util.UserRole
	todo := randomTodo(user.Username)

	tcs := []struct {
		name               string
		todoID             int64
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "OK",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(todo, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "UnauthorizedUser",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, util.RandomString(10), role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(todo, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:      "NoAuthorization",
			todoID:    todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusUnauthorized, recorder.Code)
			},
		},
		{
			name:   "NotFound",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(db.Todo{}, db.ErrRecordNotFound)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "todo",
				id:           todo.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, expectedError)
			},
		},
		{
			name:   "InternalError",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(db.Todo{}, sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, expectedError)
			},
		},
		{
			name:   "InvalidID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Any()).
					Times(0)
			},
			errorExpected: true,
			expectedError: todoIDInvalidError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, expectedError)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store)

			// start test server and send request
			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d", tc.todoID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.Router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}
		})
	}
}

func TestCreateTodoAPI(t *testing.T) {
	user, _ := randomUser(t)
	role := util.UserRole
	todo := randomTodo(user.Username)

	tcs := []struct {
		name               string
		body               gin.H
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name: "OK",
			body: gin.H{
				"title": todo.Title,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				arg := db.CreateTodoTxParams{
					TodoOwner: user.Username,
					TodoTitle: todo.Title,
					Storage:   mockStorage,
				}
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(db.CreateTodoTxResult{Todo: todo}, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name: "InvalidRequestTitleTooLong",
			body: gin.H{
				"title": util.RandomString(256),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidRequestTitleAbsent",
			body: nil,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalServerError",
			body: gin.H{
				"title": todo.Title,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateTodoTxResult{}, sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, expectedError error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, expectedError)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mockStorage.NewMockStorage(ctrl)

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store, mockStorage)

			// start test server and send request
			server := newTestServer(t, store, mockStorage)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			assert.NoError(t, err)

			url := "/todos"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.Router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}
		})
	}
}

func TestListTodoAPI(t *testing.T) {
	user, _ := randomUser(t)
	role := util.UserRole

	n := 5
	todos := make([]db.Todo, 0)
	for i := 0; i < n; i++ {
		todos = append(todos, randomTodo(user.Username))
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	tcs := []struct {
		name               string
		todoID             int64
		query              Query
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name: "OK",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				arg := db.ListTodosParams{
					Owner:  user.Username,
					Limit:  int32(n),
					Offset: 0,
				}

				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(todos, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodos(t, recorder.Body, todos)
			},
		},
		{
			name: "InternalError",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Todo{}, sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				pageID:   -1,
				pageSize: n,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidPageSize",
			query: Query{
				pageID:   1,
				pageSize: 100,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store)

			// start test server and send request
			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			url := "/todos"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			assert.NoError(t, err)

			// Add query parameters to response URL
			q := request.URL.Query()
			q.Add("pageId", fmt.Sprintf("%d", tc.query.pageID))
			q.Add("pageSize", fmt.Sprintf("%d", tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			tc.setupAuth(t, request, server.tokenMaker)
			server.Router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}
		})
	}
}

func TestUpdateTodoAPI(t *testing.T) {
	user, _ := randomUser(t)
	role := util.UserRole
	todo := randomTodo(user.Username)

	updatedTitle := util.RandomString(10)
	updatedStatus := util.RandomStatus()

	tcs := []struct {
		name               string
		todoID             int64
		body               gin.H
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "InvalidID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: todoIDInvalidError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "InvalidTitle",
			todoID: todo.ID,
			body: gin.H{
				"title": util.RandomString(256),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "InvalidStatusString",
			todoID: todo.ID,
			body: gin.H{
				"status": util.RandomString(10),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "InvalidStatusNumber",
			todoID: todo.ID,
			body: gin.H{
				"status": util.RandomInt(1, 1000),
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "TitleAndStatusNotPresent",
			todoID: todo.ID,
			body:   gin.H{},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: updateTodoTitleStatusInvalidBodyError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "TodoNotFound",
			todoID: todo.ID,
			body: gin.H{
				"title":  updatedTitle,
				"status": updatedStatus,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "todo",
				id:           todo.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "InternalError",
			todoID: todo.ID,
			body: gin.H{
				"title":  updatedTitle,
				"status": updatedStatus,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Any()).Times(1).Return(db.Todo{}, sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "OKOnlyTitleUpdate",
			todoID: todo.ID,
			body: gin.H{
				"title": updatedTitle,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.UpdateTodoTitleStatusParams{
					ID:    todo.ID,
					Title: &updatedTitle,
				}
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Eq(arg)).Times(1).Return(todo, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "OKOnlyStatusUpdate",
			todoID: todo.ID,
			body: gin.H{
				"status": updatedStatus,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.UpdateTodoTitleStatusParams{
					ID:     todo.ID,
					Status: &updatedStatus,
				}
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Eq(arg)).Times(1).Return(todo, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "OKTitleAndStatusUpdate",
			todoID: todo.ID,
			body: gin.H{
				"title":  updatedTitle,
				"status": updatedStatus,
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.UpdateTodoTitleStatusParams{
					ID:     todo.ID,
					Title:  &updatedTitle,
					Status: &updatedStatus,
				}
				store.EXPECT().UpdateTodoTitleStatus(gomock.Any(), gomock.Eq(arg)).Times(1).Return(todo, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchTodo(t, recorder.Body, todo)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store)

			// start test server and send request
			server := newTestServer(t, store, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			assert.NoError(t, err)

			url := fmt.Sprintf("/todos/%d", tc.todoID)
			request, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(data))
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.Router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}
		})
	}
}

func TestDeleteTodoAPI(t *testing.T) {
	user, _ := randomUser(t)
	role := util.UserRole
	todo := randomTodo(user.Username)

	tcs := []struct {
		name               string
		todoID             int64
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "InvalidID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: todoIDInvalidError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "NotFound",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "todo",
				id:           todo.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "InternalError",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.DeleteTodoTxParams{
					TodoID:  todo.ID,
					Storage: mockStorage,
				}
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "OK",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, role, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.DeleteTodoTxParams{
					TodoID:  todo.ID,
					Storage: mockStorage,
				}
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockStorage := mockStorage.NewMockStorage(ctrl)

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store, mockStorage)

			// start test server and send request
			server := newTestServer(t, store, mockStorage)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d", tc.todoID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.Router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}

		})
	}
}
