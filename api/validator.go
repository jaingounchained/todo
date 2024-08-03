package api

import (
	"errors"
	"net/textproto"

	"github.com/go-playground/validator/v10"
	"github.com/jaingounchained/todo/util"
)

const (
	FileSizeLimit = 2 << 20
	ContentType   = "Content-Type"
)

var validTodoStatus validator.Func = func(fl validator.FieldLevel) bool {
	if todoStatus, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedTodoStatus(todoStatus)
	}

	return false
}

func validateMimeType(a textproto.MIMEHeader) error {
	mimeType := a.Get(ContentType)
	if !util.IsSupportedMimeType(mimeType) {
		return errors.New("Unsupported file type")
	}

	return nil
}

func validateFileSize(fileSize int64) error {
	if fileSize > FileSizeLimit {
		return errors.New("File size larger than permissible")
	}

	return nil
}
