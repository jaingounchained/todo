// Code generated by MockGen. DO NOT EDIT.
// Source: github.com/jaingounchained/todo/storage (interfaces: Storage)

// Package mockStorage is a generated GoMock package.
package mockStorage

import (
	context "context"
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
	storage "github.com/jaingounchained/todo/storage"
)

// MockStorage is a mock of Storage interface.
type MockStorage struct {
	ctrl     *gomock.Controller
	recorder *MockStorageMockRecorder
}

// MockStorageMockRecorder is the mock recorder for MockStorage.
type MockStorageMockRecorder struct {
	mock *MockStorage
}

// NewMockStorage creates a new mock instance.
func NewMockStorage(ctrl *gomock.Controller) *MockStorage {
	mock := &MockStorage{ctrl: ctrl}
	mock.recorder = &MockStorageMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorage) EXPECT() *MockStorageMockRecorder {
	return m.recorder
}

// CreateTodoDirectory mocks base method.
func (m *MockStorage) CreateTodoDirectory(arg0 context.Context, arg1 int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CreateTodoDirectory", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// CreateTodoDirectory indicates an expected call of CreateTodoDirectory.
func (mr *MockStorageMockRecorder) CreateTodoDirectory(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CreateTodoDirectory", reflect.TypeOf((*MockStorage)(nil).CreateTodoDirectory), arg0, arg1)
}

// DeleteFile mocks base method.
func (m *MockStorage) DeleteFile(arg0 context.Context, arg1 int64, arg2 string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteFile", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteFile indicates an expected call of DeleteFile.
func (mr *MockStorageMockRecorder) DeleteFile(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteFile", reflect.TypeOf((*MockStorage)(nil).DeleteFile), arg0, arg1, arg2)
}

// DeleteTodoDirectory mocks base method.
func (m *MockStorage) DeleteTodoDirectory(arg0 context.Context, arg1 int64) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteTodoDirectory", arg0, arg1)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteTodoDirectory indicates an expected call of DeleteTodoDirectory.
func (mr *MockStorageMockRecorder) DeleteTodoDirectory(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteTodoDirectory", reflect.TypeOf((*MockStorage)(nil).DeleteTodoDirectory), arg0, arg1)
}

// GetFileContents mocks base method.
func (m *MockStorage) GetFileContents(arg0 context.Context, arg1 int64, arg2 string) ([]byte, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFileContents", arg0, arg1, arg2)
	ret0, _ := ret[0].([]byte)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFileContents indicates an expected call of GetFileContents.
func (mr *MockStorageMockRecorder) GetFileContents(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFileContents", reflect.TypeOf((*MockStorage)(nil).GetFileContents), arg0, arg1, arg2)
}

// SaveFile mocks base method.
func (m *MockStorage) SaveFile(arg0 context.Context, arg1 int64, arg2 string, arg3 []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveFile", arg0, arg1, arg2, arg3)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveFile indicates an expected call of SaveFile.
func (mr *MockStorageMockRecorder) SaveFile(arg0, arg1, arg2, arg3 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveFile", reflect.TypeOf((*MockStorage)(nil).SaveFile), arg0, arg1, arg2, arg3)
}

// SaveMultipleFilesSafely mocks base method.
func (m *MockStorage) SaveMultipleFilesSafely(arg0 context.Context, arg1 int64, arg2 storage.FileContents) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SaveMultipleFilesSafely", arg0, arg1, arg2)
	ret0, _ := ret[0].(error)
	return ret0
}

// SaveMultipleFilesSafely indicates an expected call of SaveMultipleFilesSafely.
func (mr *MockStorageMockRecorder) SaveMultipleFilesSafely(arg0, arg1, arg2 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SaveMultipleFilesSafely", reflect.TypeOf((*MockStorage)(nil).SaveMultipleFilesSafely), arg0, arg1, arg2)
}
