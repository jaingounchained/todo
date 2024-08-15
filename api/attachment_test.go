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
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang/mock/gomock"
	mockdb "github.com/jaingounchained/todo/db/mock"
	db "github.com/jaingounchained/todo/db/sqlc"
	"github.com/jaingounchained/todo/storage"
	mockStorage "github.com/jaingounchained/todo/storage/mock"
	"github.com/jaingounchained/todo/token"
	"github.com/jaingounchained/todo/util"
	"github.com/stretchr/testify/assert"
)

func randomAttachment() db.Attachment {
	return db.Attachment{
		ID:               util.RandomInt(1, 1000),
		TodoID:           util.RandomInt(1, 1000),
		StorageFilename:  util.RandomString(10),
		OriginalFilename: util.RandomString(10),
	}
}

func randomAttachmentOfTodo(todo db.Todo) db.Attachment {
	return db.Attachment{
		ID:               util.RandomInt(1, 1000),
		TodoID:           todo.ID,
		StorageFilename:  util.RandomString(10),
		OriginalFilename: util.RandomString(10),
	}
}

func assertBodyMatchAttachment(t *testing.T, body *bytes.Buffer, attachment db.Attachment) {
	data, err := io.ReadAll(body)
	assert.NoError(t, err)

	var gotAttachment db.Attachment
	err = json.Unmarshal(data, &gotAttachment)
	assert.NoError(t, err)
	assert.Equal(t, attachment, gotAttachment)
}

func assertBodyMatchAttachmentsMetadata(t *testing.T, body *bytes.Buffer, attachments []db.Attachment) {
	expectedAttachmentMetadataResponse := make([]getTodoAttachmentMetadataResponse, 0)
	for _, attachment := range attachments {
		expectedAttachmentMetadataResponse = append(expectedAttachmentMetadataResponse, getTodoAttachmentMetadataResponse{
			ID:       attachment.ID,
			TodoID:   attachment.TodoID,
			Filename: attachment.OriginalFilename,
		})
	}

	data, err := io.ReadAll(body)
	assert.NoError(t, err)

	var gotAttachments []getTodoAttachmentMetadataResponse
	err = json.Unmarshal(data, &gotAttachments)
	assert.NoError(t, err)
	assert.Equal(t, expectedAttachmentMetadataResponse, gotAttachments)
}

func TestUploadTodoAttachmentsAPI(t *testing.T) {
	user, _ := randomUser(t)

	todo := randomTodo(user.Username)
	todo.FileCount = 1

	todoWithMaxFileCount := randomTodo(user.Username)
	todoWithMaxFileCount.FileCount = 5

	todoWithFileCount4 := randomTodo(user.Username)
	todoWithFileCount4.FileCount = 4

	randomMimeType := util.RandomString(10)

	type File struct {
		fileName     string
		fileMimeType string
		fileContents []byte
	}

	tcs := []struct {
		name               string
		todoID             int64
		fieldName          string
		files              []File
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "InvalidID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: todoIDInvalidError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "TodoNotFound",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
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
			name:   "TodoQueryDBError",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, sql.ErrConnDone)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "MaxTodoFileCount",
			todoID: todoWithMaxFileCount.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todoWithMaxFileCount.ID)).Times(1).Return(todoWithMaxFileCount, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: newTodoAttachmentLimitReachedError(int(todoWithMaxFileCount.FileCount)),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: uploadAttachmentAPIContentLengthLimitError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: attachmentKeyEmptyError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todoWithFileCount4.ID)).Times(1).Return(todoWithFileCount4, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: newTodoAttachmentLimitReachedError(2),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
					fileMimeType: randomMimeType,
					fileContents: []byte(util.RandomString(100)),
				},
			},
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: newInvalidMimeTypeError("example.zip", randomMimeType),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusUnsupportedMediaType, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage, fileContents storage.FileContents) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().UploadAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: newFileSizeTooLargeError("example.txt", FileSizeLimit),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
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
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
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
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
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

			expectedFleContents := make(map[string][]byte)
			for _, file := range tc.files {
				expectedFleContents[file.fileName] = file.fileContents
			}
			store := mockdb.NewMockStore(ctrl)
			tc.buildDBStub(store, mockStorage, expectedFleContents)

			// start test server and send request
			server := newTestServer(t, store, mockStorage)
			recorder := httptest.NewRecorder()

			// Create a buffer to hold the multipart form data
			var requestBody bytes.Buffer
			multipartWriter := multipart.NewWriter(&requestBody)

			for _, file := range tc.files {
				header := textproto.MIMEHeader{}
				header.Set("Content-Disposition", fmt.Sprintf("form-data; name=\"%s\"; filename=\"%s\"", tc.fieldName, file.fileName))
				header.Set("Content-Type", file.fileMimeType)

				partWriter, err := multipartWriter.CreatePart(header)
				assert.NoError(t, err)

				_, err = partWriter.Write(file.fileContents)
				assert.NoError(t, err)
			}

			err := multipartWriter.Close()
			assert.NoError(t, err)

			url := fmt.Sprintf("/todos/%d/attachments", tc.todoID)
			request, err := http.NewRequest(http.MethodPost, url, &requestBody)
			assert.NoError(t, err)

			request.Header.Set("Content-Type", multipartWriter.FormDataContentType())

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}
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
		server := newTestServer(t, store, mockStorage)
		recorder := httptest.NewRecorder()

		// Marshal body data to JSON
		data, err := json.Marshal(&gin.H{})
		assert.NoError(t, err)

		url := fmt.Sprintf("/todos/%d/attachments", todo.ID)
		request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
		assert.NoError(t, err)

		request.Header.Set("Content-Type", util.RandomString(10))

		addAuthorization(t, request, server.tokenMaker, authorizationTypeBearer, user.Username, time.Minute)

		server.router.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusBadRequest, recorder.Code)
		assertBodyMatchError(t, recorder.Body, invalidHeaderContentTypeError)
	})
}

func TestGetTodoAttachmentAPI(t *testing.T) {
	user, _ := randomUser(t)

	todo := randomTodo(user.Username)
	attachment := randomAttachment()

	attachmentWithTodo := randomAttachmentOfTodo(todo)

	fileContents := []byte(util.RandomString(100))

	tcs := []struct {
		name               string
		todoID             int64
		attachmentID       int64
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore)
		buildStorageStub   func(mockStorage *mockStorage.MockStorage)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:         "InvalidTodoID",
			todoID:       0,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(0)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "InvalidAttachmentID",
			todoID:       todo.ID,
			attachmentID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "TodoNotFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
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
			name:         "NoAttachmentFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, db.ErrRecordNotFound)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "attachment",
				id:           attachment.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "GetAttachmentQueryInternalError",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, sql.ErrConnDone)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "AttachmentTodoIDNETodoID",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(attachment, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(0)
			},
			errorExpected: true,
			expectedError: newAttachmentNotAssociatedWithTodoError(todo.ID, attachment.ID),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "StorageInternalError",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(1).
					Return(nil, errors.New("storage failure"))
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
			},
		},
		{
			name:         "OK",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
			},
			buildStorageStub: func(mockStorage *mockStorage.MockStorage) {
				mockStorage.EXPECT().
					GetFileContents(gomock.Any(), gomock.Eq(todo.ID), gomock.Eq(attachmentWithTodo.StorageFilename)).
					Times(1).
					Return(fileContents, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				data, err := io.ReadAll(recorder.Body)
				assert.NoError(t, err)
				assert.Equal(t, data, fileContents)
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
			server := newTestServer(t, store, mockStorage)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/todos/%d/attachments/%d", tc.todoID, tc.attachmentID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}

		})
	}
}

func TestGetTodoAttachmentMetadataAPI(t *testing.T) {
	user, _ := randomUser(t)
	todo := randomTodo(user.Username)
	attachment1 := randomAttachment()
	attachment1.TodoID = todo.ID
	attachment2 := randomAttachment()
	attachment2.TodoID = todo.ID

	tcs := []struct {
		name               string
		todoID             int64
		buildDBStub        func(store *mockdb.MockStore)
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "InvalidTodoID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: todoIDInvalidError,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "TodoNotFound",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
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
			name:   "GetTodoQueryInternalError",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, sql.ErrConnDone)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "NoAttachmentsFound",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{}, db.ErrRecordNotFound)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "attachment",
				id:           todo.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:   "ListAttachmentQueryInternalError",
			todoID: todo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{}, sql.ErrConnDone)
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
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().ListAttachmentOfTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return([]db.Attachment{attachment1, attachment2}, nil)
			},
			checkOKResponse: func(recorder *httptest.ResponseRecorder) {
				assert.Equal(t, http.StatusOK, recorder.Code)
				assertBodyMatchAttachmentsMetadata(t, recorder.Body, []db.Attachment{attachment1, attachment2})
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

			url := fmt.Sprintf("/todos/%d/attachments", tc.todoID)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}

		})
	}
}

func TestDeleteTodoAttachmentAPI(t *testing.T) {
	user, _ := randomUser(t)

	todo := randomTodo(user.Username)
	attachment := randomAttachment()

	attachmentWithTodo := randomAttachmentOfTodo(todo)

	tcs := []struct {
		name               string
		todoID             int64
		attachmentID       int64
		setupAuth          func(t *testing.T, request *http.Request, tokenMaker token.Maker)
		buildDBStub        func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage)
		checkOKResponse    func(recorder *httptest.ResponseRecorder)
		errorExpected      bool
		expectedError      error
		checkErrorResponse func(recorder *httptest.ResponseRecorder, err error)
	}{
		{
			name:   "InvalidTodoID",
			todoID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "InvalidAttachmentID",
			todoID:       todo.ID,
			attachmentID: 0,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Any()).Times(0)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusBadRequest, recorder.Code)
			},
		},
		{
			name:         "TodoNotFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(db.Todo{}, db.ErrRecordNotFound)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
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
			name:         "NoAttachmentFound",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, db.ErrRecordNotFound)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: &ResourceNotFoundError{
				resourceType: "attachment",
				id:           attachment.ID,
			},
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusNotFound, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "GetAttachmentQueryInternalError",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(db.Attachment{}, sql.ErrConnDone)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "AttachmentTodoIDNETodoID",
			todoID:       todo.ID,
			attachmentID: attachment.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachment.ID)).Times(1).Return(attachment, nil)
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Any()).Times(0)
			},
			errorExpected: true,
			expectedError: newAttachmentNotAssociatedWithTodoError(todo.ID, attachment.ID),
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusForbidden, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "DeleteTodoTxInternalError",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
				arg := db.DeleteAttachmentTxParams{
					TodoID:     todo.ID,
					Attachment: attachmentWithTodo,
					Storage:    mockStorage,
				}
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(sql.ErrConnDone)
			},
			errorExpected: true,
			expectedError: sql.ErrConnDone,
			checkErrorResponse: func(recorder *httptest.ResponseRecorder, err error) {
				assert.Equal(t, http.StatusInternalServerError, recorder.Code)
				assertBodyMatchError(t, recorder.Body, err)
			},
		},
		{
			name:         "OK",
			todoID:       todo.ID,
			attachmentID: attachmentWithTodo.ID,
			setupAuth: func(t *testing.T, request *http.Request, tokenMaker token.Maker) {
				addAuthorization(t, request, tokenMaker, authorizationTypeBearer, user.Username, time.Minute)
			},
			buildDBStub: func(store *mockdb.MockStore, mockStorage *mockStorage.MockStorage) {
				store.EXPECT().GetTodo(gomock.Any(), gomock.Eq(todo.ID)).Times(1).Return(todo, nil)
				store.EXPECT().GetAttachment(gomock.Any(), gomock.Eq(attachmentWithTodo.ID)).Times(1).Return(attachmentWithTodo, nil)
				arg := db.DeleteAttachmentTxParams{
					Attachment: attachmentWithTodo,
					TodoID:     todo.ID,
					Storage:    mockStorage,
				}
				store.EXPECT().DeleteAttachmentTx(gomock.Any(), gomock.Eq(arg)).Times(1).Return(nil)
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

			url := fmt.Sprintf("/todos/%d/attachments/%d", tc.todoID, tc.attachmentID)
			request, err := http.NewRequest(http.MethodDelete, url, nil)
			assert.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)
			server.router.ServeHTTP(recorder, request)
			// check response/error
			if tc.errorExpected {
				tc.checkErrorResponse(recorder, tc.expectedError)
			} else {
				tc.checkOKResponse(recorder)
			}

		})
	}
}
