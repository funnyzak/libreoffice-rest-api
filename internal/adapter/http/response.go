package http

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	domainerrors "github.com/funnyzak/libreoffice-rest-api/pkg/errors"
)

// APIResponse 统一响应结构。
type APIResponse struct {
	Success bool         `json:"success"`
	Data    any          `json:"data,omitempty"`
	Message string       `json:"message,omitempty"`
	Error   *ErrorDetail `json:"error,omitempty"`
}

// ErrorDetail 错误信息。
type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// WriteSuccess 返回成功响应。
func WriteSuccess(c *gin.Context, status int, data any, message string) {
	c.JSON(status, APIResponse{
		Success: true,
		Data:    data,
		Message: message,
	})
}

// WriteError 返回错误响应。
func WriteError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "服务器内部错误"
	details := ""

	var de *domainerrors.DomainError
	if ok := domainerrors.AsDomainError(err, &de); ok {
		switch de.Type {
		case domainerrors.ErrorValidation:
			status = http.StatusBadRequest
			message = de.Message
			details = de.Details
		case domainerrors.ErrorAuthentication:
			status = http.StatusUnauthorized
			message = de.Message
			details = de.Details
		case domainerrors.ErrorNotFound:
			status = http.StatusNotFound
			message = de.Message
			details = de.Details
		case domainerrors.ErrorConversion:
			status = http.StatusInternalServerError
			message = de.Message
			details = de.Details
		case domainerrors.ErrorStorage:
			status = http.StatusInternalServerError
			message = de.Message
			details = de.Details
		}
	} else if err == gorm.ErrRecordNotFound {
		status = http.StatusNotFound
		message = "资源不存在"
	}

	c.JSON(status, APIResponse{
		Success: false,
		Error: &ErrorDetail{
			Code:    status,
			Message: message,
			Details: details,
		},
	})
}
