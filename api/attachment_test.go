package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/jaingounchained/todo/db/mock"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/storage"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/require"
)

func RandomAttachment() db.Attachment {
	return db.Attachment{
		ID:               util.RandomInt(1, 1000),
		TodoID:           util.RandomInt(1, 1000),
		StorageFilename:  util.RandomString(10),
		OriginalFilename: util.RandomString(10),
	}
}

func RandomAttachmentOfTodo(todo db.Todo) db.Attachment {
	return db.Attachment{
		ID:               util.RandomInt(1, 1000),
		TodoID:           todo.ID,
		StorageFilename:  util.RandomString(10),
		OriginalFilename: util.RandomString(10),
	}
}

func requireBodyMatchAttachment(t *testing.T, body *bytes.Buffer, attachment db.Attachment) {
	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAttachment db.Attachment
	err = json.Unmarshal(data, &gotAttachment)
	require.NoError(t, err)
	require.Equal(t, attachment, gotAttachment)
}

func requireBodyMatchAttachmentsMetadata(t *testing.T, body *bytes.Buffer, attachments []db.Attachment) {
	expectedAttachmentMetadataResponse := make([]getTodoAttachmentMetadataResponse, 0)
	for _, attachment := range attachments {
		expectedAttachmentMetadataResponse = append(expectedAttachmentMetadataResponse, getTodoAttachmentMetadataResponse{
			ID:       attachment.ID,
			TodoID:   attachment.TodoID,
			Filename: attachment.OriginalFilename,
		})
	}

	data, err := io.ReadAll(body)
	require.NoError(t, err)

	var gotAttachments []getTodoAttachmentMetadataResponse
	err = json.Unmarshal(data, &gotAttachments)
	require.NoError(t, err)
	require.Equal(t, expectedAttachmentMetadataResponse, gotAttachments)
}

func TestUploadTodoAttachmentsAPI(t *testing.T) {
	todo := RandomTodo()
	todo.FileCount = 1

	todoWithMaxFileCount := RandomTodo()
	todoWithMaxFileCount.FileCount = 5

	todoWithFileCount4 := RandomTodo()
	todoWithFileCount4.FileCount = 4

	type File struct {
		fileName     string
		fileMimeType string
		fileContents []byte
	}

	tcs := []struct {
		name          string
		todoID        int64
		fieldName     string
		files         []File
		buildDBStub   func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "InvalidID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "TodoAbsent",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "TodoQueryDBError",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, sql.ErrConnDone)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "MaxTodoFileCount",
			todoID: todoWithMaxFileCount.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todoWithMaxFileCount.ID)).Times(1).Return(todoWithMaxFileCount, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:      "LargeFile",
			todoID:    todo.ID,
			fieldName: util.RandomString(10),
			files: []File{
				{
					fileName:     util.RandomString(10),
					fileContents: []byte(util.RandomString(15 << 20)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
			},
		},
		{
			name:      "FileContentsPresentInWrongKey",
			todoID:    todo.ID,
			fieldName: util.RandomString(10),
			files: []File{
				{
					fileName:     util.RandomString(10),
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:      "TodoFiles+RequestFiles>5",
			todoID:    todoWithFileCount4.ID,
			fieldName: UploadAttachmentFormFileKey,
			files: []File{
				{
					fileName:     util.RandomString(10),
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
				{
					fileName:     util.RandomString(10),
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todoWithFileCount4.ID)).Times(1).Return(todoWithFileCount4, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
			},
		},
		{
			name:      "AttachmentFileTypeNotSupported",
			todoID:    todo.ID,
			fieldName: UploadAttachmentFormFileKey,
			files: []File{
				{
					fileName:     "example.txt",
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
				{
					fileName:     "example.zip",
					fileMimeType: util.RandomString(10),
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
			},
		},
		{
			name:      "AttachmentFileSizeTooLarge",
			todoID:    todo.ID,
			fieldName: UploadAttachmentFormFileKey,
			files: []File{
				{
					fileName:     "example.txt",
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(3 << 20)),
				},
				{
					fileName:     "example.jpg",
					fileMimeType: util.ImageJPG,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
			},
		},
		{
			name:      "UploadAttachmentTxInternalError",
			todoID:    todo.ID,
			fieldName: UploadAttachmentFormFileKey,
			files: []File{
				{
					fileName:     "example.txt",
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
				{
					fileName:     "example.jpg",
					fileMimeType: util.ImageJPG,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.UploadAttachmentTxParams{
					Todo:         todo,
					FileContents: fileContents,
					Storage:      mockStorage,
				}
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:      "OK",
			todoID:    todo.ID,
			fieldName: UploadAttachmentFormFileKey,
			files: []File{
				{
					fileName:     "example.txt",
					fileMimeType: util.TextPlain,
					fileContents: []byte(util.RandomString(100)),
				},
				{
					fileName:     "example.jpg",
					fileMimeType: util.ImageJPG,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				arg := db.UploadAttachmentTxParams{
					Todo:         todo,
					FileContents: fileContents,
					Storage:      mockStorage,
				}
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(nil)
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

			expectedFleContents := make(map[string][]byte)
			for _, file := range tc.files {
				expectedFleContents[file.fileName] = file.fileContents
			}
			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store, mockStorage, expectedFleContents)

			// start test server and send request
			server := NewGinHandler(store, mockStorage, nil)
			recorder := httptest.NewRecorder()

			// Create a buffer to hold the multipart form data
			var requestBody bytes.Buffer
			multipartWriter := multipart.NewWriter(&requestBody)

			for _, file := range tc.files {
				header := textproto.MIMEHeader{}
				header.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"%s\"; filename=\"%s\"", tc.fieldName, file.fileName))
				header.Set("Content-Type", file.fileMimeType)

				partWriter, err := multipartWriter.CreatePart(header)
				require.NoError(t, err)

				_, err = partWriter.Write(file.fileContents)
				require.NoError(t, err)
			}

			err := multipartWriter.Close()
			require.NoError(t, err)

			url := fmt.Sprintf("/todos/%d/attachments", tc.todoID)
			request, err := http.NewRequest(http.MethodPost, url, &requestBody)
			require.NoError(t, err)

			request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}

	// Request Content-Type is not multipart form data
	t.Run("ContentTypeNotMultipartFormData", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStorage := mockStorage.NewMockStorage(ctrl)

		store := mockdb.NewMockStore(ctrl)
		// Build stubs
		store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
		store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)

		// start test server and send request
		server := NewGinHandler(store, mockStorage, nil)
		recorder := httptest.NewRecorder()

		// Marshal body data to JSON
		data, err := json.Marshal(&gin.H{})
		require.NoError(t, err)

		url := fmt.Sprintf("/todos/%d/attachments", todo.ID)
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		require.NoError(t, err)

		request.Header.Set("Content-Type", util.RandomString(10))

		server.router.ServeHTTP(recorder, request)
		require.Equal(t, http.StatusBadRequest, recorder.Code)
	})
}

func TestGetTodoAttachmentAPI(t *testing.T) {
	todo := RandomTodo()
	attachment := RandomAttachment()

	attachmentWithTodo := RandomAttachmentOfTodo(todo)

	fileContents := []byte(util.RandomString(100))

	tcs := []struct {
		name             string
		todoID           int64
		attachmentID     int64
		buildDBStub      func(store *mockdb.MockStore)
		buildStorageStub func(mockStorage *mockStorage.MockStorage)
		checkResponse    func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:         "InvalidTodoID",
			todoID:       0,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "InvalidAttachmentID",
			todoID:       todo.ID,
			attachmentID: 0,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "NoAttachmentFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, db.ErrRecordNotFound)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:         "GetAttachmentQueryInternalError",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, sql.ErrConnDone)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:         "AttachmentTodoIDNETodoID",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(attachment, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:         "StorageInternalError",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(1).
					Return(nil, errors.New("storage failure"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:         "OK",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(1).
					Return(fileContents, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				require.NoError(t, err)
				require.Equal(t, data, fileContents)
			},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store)

			mockStorage := mockStorage.NewMockStorage(ctrl)
			tc.buildStorageStub(mockStorage)

			// start test server and send request
			server := NewGinHandler(store, mockStorage, nil)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d/attachments/%d", tc.todoID, tc.attachmentID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestGetTodoAttachmentMetadataAPI(t *testing.T) {
	todo := RandomTodo()
	attachment1 := RandomAttachment()
	attachment1.TodoID = todo.ID
	attachment2 := RandomAttachment()
	attachment2.TodoID = todo.ID

	tcs := []struct {
		name          string
		todoID        int64
		buildDBStub   func(store *mockdb.MockStore)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "InvalidTodoID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:   "TodoNotFound",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "GetTodoQueryInternalError",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, sql.ErrConnDone)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "NoAttachmentsFound",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{}, db.ErrRecordNotFound)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:   "ListAttachmentQueryInternalError",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{}, sql.ErrConnDone)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:   "OK",
			todoID: todo.ID,
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{attachment1, attachment2}, nil)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
				requireBodyMatchAttachmentsMetadata(t, recorder.Body, []db.Attachment{attachment1, attachment2})
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

			url := fmt.Sprintf("/todos/%d/attachments", tc.todoID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}

func TestDeleteTodoAttachmentAPI(t *testing.T) {
	todo := RandomTodo()
	attachment := RandomAttachment()

	attachmentWithTodo := RandomAttachmentOfTodo(todo)

	tcs := []struct {
		name          string
		todoID        int64
		attachmentID  int64
		buildDBStub   func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkResponse func(recorder *httptest.ResponseRecorder)
	}{
		{
			name:   "InvalidTodoID",
			todoID: 0,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "InvalidAttachmentID",
			todoID:       todo.ID,
			attachmentID: 0,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "NoAttachmentFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, db.ErrRecordNotFound)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, recorder.Code)
			},
		},
		{
			name:         "GetAttachmentQueryInternalError",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, sql.ErrConnDone)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:         "AttachmentTodoIDNETodoID",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(attachment, nil)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, recorder.Code)
			},
		},
		{
			name:         "DeleteTodoTxInternalError",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
				arg := db.DeleteAttachmentTxParams{
					TodoID:     todo.ID,
					Attachment: attachmentWithTodo,
					Storage:    mockStorage,
				}
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(errors.New("Delete todo tx error"))
			},
			checkResponse: func(recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:         "OK",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
				arg := db.DeleteAttachmentTxParams{
					Attachment: attachmentWithTodo,
					TodoID:     todo.ID,
					Storage:    mockStorage,
				}
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(nil)
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

			url := fmt.Sprintf("/todos/%d/attachments/%d", tc.todoID, tc.attachmentID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			require.NoError(t, err)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(recorder)
		})
	}
}
