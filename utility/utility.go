package xUtil

import (
	pack "github.com/bamboo-services/bamboo-base-go/utility/package"
	"github.com/gin-gonic/gin"
)

func Bind[T any](ctx *gin.Context, data *T) *pack.Binding[T] {
	return &pack.Binding[T]{
		Context: ctx,
		Data:    data,
	}
}
