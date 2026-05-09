package routes

import (
	"github.com/gin-gonic/gin"

	"mangahub/internal/api-server/handlers"
)

func RegisterMangaRoutes(rg *gin.RouterGroup, handler *handlers.MangaHandler) {
	mangaRoutes := rg.Group("/manga")
	{
		mangaRoutes.GET("", handler.GetAllManga)
		mangaRoutes.GET("/:id", handler.GetMangaByID)
	}
}
