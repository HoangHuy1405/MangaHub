package utils

import "github.com/gin-gonic/gin"

type APIResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, APIResponse{
		Success: true,
		Code:    statusCode,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, statusCode int, message string) {
	c.JSON(statusCode, APIResponse{
		Success: false,
		Code:    statusCode,
		Message: message,
		Data:    nil,
	})
}

func Created(c *gin.Context, message string, data interface{}) {
	Success(c, 201, message, data)
}

func BadRequest(c *gin.Context, message string) {
	Error(c, 400, message)
}

func Unauthorized(c *gin.Context, message string) {
	Error(c, 401, message)
}

func NotFound(c *gin.Context, message string) {
	Error(c, 404, message)
}

func Conflict(c *gin.Context, message string) {
	Error(c, 409, message)
}

func InternalError(c *gin.Context, message string) {
	Error(c, 500, message)
}
