package xGrpcError

import (
	"errors"
	"fmt"

	xError "github.com/bamboo-services/bamboo-base-go/error"
)

type ErrMessage string

func (e ErrMessage) String() string {
	return string(e)
}

type IError interface {
	error
	GetErrorCode() *xError.ErrorCode
	GetErrorMessage() ErrMessage
	GetData() interface{}
}

type Error struct {
	*xError.ErrorCode
	ErrorMessage ErrMessage
	Data         interface{}
	cause        error
}

func New(errorCode *xError.ErrorCode, errorMessage ErrMessage, data interface{}) *Error {
	normalizedCode := normalizeErrorCode(errorCode)
	normalizedMessage := normalizeErrorMessage(normalizedCode, errorMessage)
	return &Error{
		ErrorCode:    normalizedCode,
		ErrorMessage: normalizedMessage,
		Data:         data,
	}
}

func Wrap(errorCode *xError.ErrorCode, errorMessage ErrMessage, data interface{}, cause error) *Error {
	grpcError := New(errorCode, errorMessage, data)
	grpcError.cause = cause
	return grpcError
}

func From(err error) *Error {
	if err == nil {
		return nil
	}

	var grpcError *Error
	if errors.As(err, &grpcError) {
		return grpcError
	}

	var baseError xError.IError
	if errors.As(err, &baseError) {
		return Wrap(
			baseError.GetErrorCode(),
			ErrMessage(baseError.GetErrorMessage()),
			baseError.GetData(),
			err,
		)
	}

	return Wrap(xError.ServerInternalError, ErrMessage(err.Error()), nil, err)
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	errorCode := normalizeErrorCode(e.ErrorCode)
	errorMessage := normalizeErrorMessage(errorCode, e.ErrorMessage)
	return fmt.Sprintf("[%d]%s | %s - %s", errorCode.Code, errorCode.Output, errorCode.Message, errorMessage)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *Error) GetErrorCode() *xError.ErrorCode {
	if e == nil {
		return xError.UnknownError
	}
	return normalizeErrorCode(e.ErrorCode)
}

func (e *Error) GetErrorMessage() ErrMessage {
	if e == nil {
		return ErrMessage("")
	}
	return normalizeErrorMessage(e.GetErrorCode(), e.ErrorMessage)
}

func (e *Error) GetData() interface{} {
	if e == nil {
		return nil
	}
	return e.Data
}

func normalizeErrorCode(errorCode *xError.ErrorCode) *xError.ErrorCode {
	if errorCode == nil {
		return xError.UnknownError
	}
	return errorCode
}

func normalizeErrorMessage(errorCode *xError.ErrorCode, errorMessage ErrMessage) ErrMessage {
	if errorMessage == "" {
		return ErrMessage(errorCode.Message)
	}
	return errorMessage
}
