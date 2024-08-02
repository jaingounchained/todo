package api

import (
	"github.com/go-playground/validator/v10"
	"github.com/jaingounchained/todo/util"
)

var validTodoStatus validator.Func = func(fl validator.FieldLevel) bool {
	if todoStatus, ok := fl.Field().Interface().(string); ok {
		return util.IsSupportedTodoStatus(todoStatus)
	}

	return false
}
