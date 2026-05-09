package routes

import (
	"github.com/gin-gonic/gin"

	"mangahub/internal/api-server/handlers"
	"mangahub/internal/auth"
)

func RegisterUserRoutes(rg *gin.RouterGroup, handler *handlers.UserHandler, jwtSecret string) {
	userRoutes := rg.Group("/users")
	userRoutes.Use(auth.JWTMiddleware(jwtSecret))
	{
		userRoutes.POST("/library", handler.AddToLibrary)
		userRoutes.GET("/library", handler.GetLibrary)
		userRoutes.PUT("/progress", handler.UpdateProgress)
	}
}
