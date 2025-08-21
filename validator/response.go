package xVaild

import (
	"errors"
	awakenErr "github.com/bamboo-services/bamboo-base-go/error"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationErrorDetail 验证错误详情
type ValidationErrorDetail struct {
	Field   string      `json:"field"`   // 字段名（英文）
	Tag     string      `json:"tag"`     // 验证标签
	Message string      `json:"message"` // 错误消息
	Value   interface{} `json:"value"`   // 字段值
}

// HandleValidationError 处理验证错误并返回友好的响应
func HandleValidationError(ctx *gin.Context, bindErr error) {
	var errorDetails []ValidationErrorDetail
	var firstErrorMessage string

	var validationErrors validator.ValidationErrors
	if errors.As(bindErr, &validationErrors) {
		for i, fe := range validationErrors {
			// 构建错误消息
			errorMessage := GetValidationErrorMessage(fe)

			// 记录第一个错误作为主要错误消息
			if i == 0 {
				firstErrorMessage = errorMessage
			}

			errorDetails = append(errorDetails, ValidationErrorDetail{
				Field:   fe.Field(),
				Tag:     fe.Tag(),
				Message: errorMessage,
				Value:   fe.Value(),
			})
		}
	}

	// 如果没有详细错误，使用默认消息
	if firstErrorMessage == "" {
		firstErrorMessage = "请求数据验证失败"
	}

	// 创建错误响应
	_ = ctx.Error(awakenErr.NewErrorHasData(
		ctx,
		awakenErr.BodyInvalid,
		firstErrorMessage,
		bindErr,
		false,
		errorDetails,
	))
}
