package routes

import (
	"github.com/gin-gonic/gin"

	"mangahub/internal/api-server/handlers"
	"mangahub/internal/auth"
)

func RegisterAuthRoutes(rg *gin.RouterGroup, handler *handlers.AuthHandler, jwtSecret string) {
	authRoutes := rg.Group("/auth")
	{
		authRoutes.POST("/register", handler.Register)
		authRoutes.POST("/login", handler.Login)
		authRoutes.GET("/check", handler.Check)
		
		// Protected route
		protected := authRoutes.Group("")
		protected.Use(auth.JWTMiddleware(jwtSecret))
		{
			protected.PUT("/change-password", handler.ChangePassword)
		}
	}
}
