package api

import (
	"fmt"
	"net/textproto"

	"github.com/go-playground/validator/v10"
	"github.com/jaingounchained/todo/util"
)

var validTodoStatus validator.Func = func(fl validator.FieldLevel) bool {
	if todoStatus, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedTodoStatus(todoStatus)
	}

	return false
}

type invalidMimeTypeError error

func newInvalidMimeTypeError(filename, mimeType string) invalidMimeTypeError {
	return fmt.Errorf("%s file of invalid mime type: %s", filename, mimeType)
}

func validateMimeType(filename string, mimeHeader textproto.MIMEHeader) error {
	mimeType := mimeHeader.Get(ContentType)
	if !util.IsSupportedMimeType(mimeType) {
		return newInvalidMimeTypeError(filename, mimeType)
	}

	return nil
}

type fileSizeTooLargeError error

func newFileSizeTooLargeError(filename string, fileSize int64) fileSizeTooLargeError {
	return fmt.Errorf("%s file size too large: %d MB", filename, fileSize/1024/1024)
}

func validateFileSize(filename string, fileSize int64) error {
	if fileSize > FileSizeLimit {
		return newFileSizeTooLargeError(filename, fileSize)
	}

	return nil
}
