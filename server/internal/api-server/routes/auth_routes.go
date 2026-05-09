package routes

import (
	"github.com/gin-gonic/gin"

	"mangahub/internal/api-server/handlers"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, handler *handlers.AuthHandler) {
	authRoutes := rg.Group("/auth")
	{
		authRoutes.POST("/register", handler.Register)
		authRoutes.POST("/login", handler.Login)
	}
}
