package errors

import (
	"errors"
	"fmt"
)

// ErrorType 领域错误类型。
type ErrorType string

const (
	// ErrorValidation 表示输入验证错误。
	ErrorValidation ErrorType = "validation"
	// ErrorNotFound 表示资源未找到。
	ErrorNotFound ErrorType = "not_found"
	// ErrorConversion 表示文档转换错误。
	ErrorConversion ErrorType = "conversion"
	// ErrorStorage 表示存储错误。
	ErrorStorage ErrorType = "storage"
	// ErrorAuthentication 表示认证错误。
	ErrorAuthentication ErrorType = "authentication"
)

// DomainError 领域错误结构。
type DomainError struct {
	Type    ErrorType
	Message string
	Details string
	Err     error
}

// Error 实现 error 接口。
func (e *DomainError) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// Unwrap 返回原始错误。
func (e *DomainError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New 创建领域错误。
func New(t ErrorType, message, details string, err error) *DomainError {
	return &DomainError{
		Type:    t,
		Message: message,
		Details: details,
		Err:     err,
	}
}

// IsType 判断错误类型。
func IsType(err error, t ErrorType) bool {
	var de *DomainError
	if errors.As(err, &de) {
		return de.Type == t
	}
	return false
}

// AsDomainError 判断并返回领域错误。
func AsDomainError(err error, target **DomainError) bool {
	return errors.As(err, target)
}

// NewValidation 创建校验错误。
func NewValidation(message, details string, err error) *DomainError {
	return New(ErrorValidation, message, details, err)
}

// NewNotFound 创建未找到错误。
func NewNotFound(message, details string, err error) *DomainError {
	return New(ErrorNotFound, message, details, err)
}

// NewConversion 创建转换错误。
func NewConversion(message, details string, err error) *DomainError {
	return New(ErrorConversion, message, details, err)
}

// NewStorage 创建存储错误。
func NewStorage(message, details string, err error) *DomainError {
	return New(ErrorStorage, message, details, err)
}

// NewAuthentication 创建认证错误。
func NewAuthentication(message, details string, err error) *DomainError {
	return New(ErrorAuthentication, message, details, err)
}
