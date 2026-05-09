package routes

import (
	"github.com/gin-gonic/gin"

	"mangahub/internal/api-server/handlers"
	"mangahub/pkg/utils/config"
)

func SetupRouter(
	cfg *config.Config,
	authHandler *handlers.AuthHandler,
	mangaHandler *handlers.MangaHandler,
	userHandler *handlers.UserHandler,
) *gin.Engine {
	r := gin.Default()

	v1 := r.Group("/api/v1")
	{
		RegisterAuthRoutes(v1, authHandler)
		RegisterMangaRoutes(v1, mangaHandler)
		RegisterUserRoutes(v1, userHandler, cfg.API_CONFIG.JWT_SECRET)
	}

	return r
}
