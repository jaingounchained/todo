package api

import (
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

func validateMimeType(filename string, mimeHeader textproto.MIMEHeader) error {
	mimeType := mimeHeader.Get(ContentType)
	if !util.IsSupportedMimeType(mimeType) {
		return newInvalidMimeTypeError(filename, mimeType)
	}

	return nil
}

func validateFileSize(filename string, fileSize int64) error {
	if fileSize > FileSizeLimit {
		return newFileSizeTooLargeError(filename, FileSizeLimit)
	}

	return nil
}
