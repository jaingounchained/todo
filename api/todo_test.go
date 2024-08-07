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

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/jaingounchained/todo/db/mock"
	db "github.com/jaingounchained/todo/db/sqlc"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func RandomTodo() db.Todo {
	return db.Todo{
		ID:        util.RandomInt(1, 1000),
		Title:     util.RandomString(10),
		Status:    util.RandomStatus(),
		FileCount: 0,
	}
}

func requireBodyMatchTodo(t *testing.T, body *bytes.Buffer, todo db.Todo) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTodo db.Todo
	err = json.Unmarshal(data, &gotTodo)
	require.NoError(t, err)
	require.Equal(t, todo, gotTodo)
}

func requireBodyMatchTodos(t *testing.T, body *bytes.Buffer, todos []db.Todo) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotTodos []db.Todo
	err = json.Unmarshal(data, &gotTodos)
	require.NoError(t, err)
	require.Equal(t, todos, gotTodos)
}

func TestGetTodoAPI(t *testing.T) {
	todo := RandomTodo()

	tcs := []struct {
		name          string
		todoID        int64
		buildDBStub   func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(todo, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "NotFound",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(db.Todo{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Eq(todo.ID)).
					Times(1).
					Return(db.Todo{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "InvalidID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					GetTodo(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
			server := NewGinHandler(store, nil, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d", tc.todoID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			// check response
			tc.checkResponse(recorder)
		})
	}
}

func TestCreateTodoAPI(t *testing.T) {
	todo := RandomTodo()

	tcs := []struct {
		name          string
		body          gin.H
		buildDBStub   func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"title": todo.Title,
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				arg := db.CreateTodoTxParams{
					TodoTitle: todo.Title,
					Storage:   mockStorage,
				}
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(db.CreateTodoTxResult{Todo: todo}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name: "InvalidRequestTitleTooLong",
			body: gin.H{
				"title": util.RandomString(256),
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidRequestTitleAbsent",
			body: nil,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InternalServerError",
			body: gin.H{
				"title": todo.Title,
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().
					CreateTodoTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.CreateTodoTxResult{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			server := NewGinHandler(store, mockStorage, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/todos"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestListTodoAPI(t *testing.T) {
	n := 5
	todos := make([]db.Todo, 0)
	for i := 0; i < n; i++ {
		todos = append(todos, RandomTodo())
	}

	type Query struct {
		pageID   int
		pageSize int
	}

	tcs := []struct {
		name          string
		todoID        int64
		query         Query
		buildDBStub   func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				arg := db.ListTodosParams{
					Limit:  int32(n),
					Offset: 0,
				}

				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(todos, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTodos(t, recorder.Body, todos)
			},
		},
		{
			name: "InternalError",
			query: Query{
				pageID:   1,
				pageSize: n,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(1).
					Return([]db.Todo{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name: "InvalidPageID",
			query: Query{
				pageID:   -1,
				pageSize: n,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "InvalidPageSize",
			query: Query{
				pageID:   1,
				pageSize: 100,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					ListTodos(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
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
			server := NewGinHandler(store, nil, nil)
			recorder := httptest.NewRecorder()

			url := "/todos"
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			// Add query parameters to response URL
			q := request.URL.Query()
			q.Add("pageId", fmt.Sprintf("%d", tc.query.pageID))
			q.Add("pageSize", fmt.Sprintf("%d", tc.query.pageSize))
			request.URL.RawQuery = q.Encode()

			server.router.ServeHTTP(recorder, request)
			// check response
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdateTodoTitleAPI(t *testing.T) {
	todo := RandomTodo()
	updatedTitle := util.RandomString(10)

	tcs := []struct {
		name          string
		todoID        int64
		body          gin.H
		buildDBStub   func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			todoID: todo.ID,
			body: gin.H{
				"title": updatedTitle,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				arg := db.UpdateTodoTitleParams{
					ID:    todo.ID,
					Title: updatedTitle,
				}
				store.EXPECT().
					UpdateTodoTitle(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(todo, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "InvalidID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoTitle(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "InvalidTitle",
			todoID: todo.ID,
			body: gin.H{
				"title": util.RandomString(256),
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoTitle(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "NotFound",
			body: gin.H{
				"title": updatedTitle,
			},
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoTitle(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Todo{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"title": updatedTitle,
			},
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoTitle(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Todo{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			server := NewGinHandler(store, nil, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/todos/%d/title", tc.todoID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			// check response
			tc.checkResponse(recorder)
		})
	}
}

func TestUpdateTodoStatusAPI(t *testing.T) {
	todo := RandomTodo()
	updatedStatus := util.RandomStatus()

	tcs := []struct {
		name          string
		todoID        int64
		body          gin.H
		buildDBStub   func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			todoID: todo.ID,
			body: gin.H{
				"status": updatedStatus,
			},
			buildDBStub: func(store *mockdb.MockStore) {
				arg := db.UpdateTodoStatusParams{
					ID:     todo.ID,
					Status: updatedStatus,
				}
				store.EXPECT().
					UpdateTodoStatus(gomock.Any(), gomock.Eq(arg)).
					Times(1).
					Return(todo, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchTodo(t, recorder.Body, todo)
			},
		},
		{
			name:   "InvalidID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoStatus(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "InvalidStatus",
			todoID: todo.ID,
			body: gin.H{
				"status": util.RandomString(10),
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoStatus(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name: "NotFound",
			body: gin.H{
				"status": updatedStatus,
			},
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoStatus(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Todo{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name: "InternalError",
			body: gin.H{
				"status": updatedStatus,
			},
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().
					UpdateTodoStatus(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.Todo{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
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
			server := NewGinHandler(store, nil, nil)
			recorder := httptest.NewRecorder()

			// Marshal body data to JSON
			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := fmt.Sprintf("/todos/%d/status", tc.todoID)
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			// check response
			tc.checkResponse(recorder)
		})
	}
}

func TestDeleteTodoAPI(t *testing.T) {
	todo := RandomTodo()

	tcs := []struct {
		name          string
		todoID        int64
		buildDBStub   func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "InvalidID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "NotFound",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "InternalError",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.DeleteTodoTxParams{
					TodoID:  todo.ID,
					Storage: mockStorage,
				}
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "OK",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.DeleteTodoTxParams{
					TodoID:  todo.ID,
					Storage: mockStorage,
				}
				store.EXPECT().DeleteTodoTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
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
			server := NewGinHandler(store, mockStorage, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d", tc.todoID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}
