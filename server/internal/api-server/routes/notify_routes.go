package routes

import "github.com/gin-gonic/gin"

import "mangahub/internal/api-server/handlers"

// RegisterNotifyRoutes mounts the UDP notification trigger endpoint.
//
// POST /api/v1/notify/chapter
//
//	Body: { "manga_id": "...", "message": "..." }
//	→ broadcasts a new-chapter notification to all registered UDP clients.
func RegisterNotifyRoutes(rg *gin.RouterGroup, h *handlers.NotifyHandler) {
	notify := rg.Group("/notify")
	{
		notify.POST("/chapter", h.NotifyNewChapter)
	}
}
