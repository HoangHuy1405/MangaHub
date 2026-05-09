package handlers

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"

	"mangahub/internal/udp"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

// NotifyHandler exposes an HTTP endpoint that triggers a UDP broadcast
// to all registered notification clients when a new manga chapter is released.
//
// The UDP server is injected at construction time (dependency injection),
// keeping the handler testable and the coupling explicit.
type NotifyHandler struct {
	udpSrv *udp.NotificationServer
}

// NewNotifyHandler creates a NotifyHandler backed by the given UDP server.
func NewNotifyHandler(udpSrv *udp.NotificationServer) *NotifyHandler {
	return &NotifyHandler{udpSrv: udpSrv}
}

// notifyChapterInput is the expected JSON body for the notify endpoint.
type notifyChapterInput struct {
	MangaID string `json:"manga_id" binding:"required"`
	Message string `json:"message"  binding:"required"`
}

// NotifyNewChapter handles POST /api/v1/notify/chapter.
// It constructs a Notification and hands it to the UDP server for broadcast.
//
// Graceful degradation: a UDP broadcast failure is logged but does NOT
// return an HTTP 5xx — the REST layer must stay healthy even when the
// notification layer is unavailable.
func (h *NotifyHandler) NotifyNewChapter(c *gin.Context) {
	var input notifyChapterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "manga_id and message are required")
		return
	}

	n := models.Notification{
		Type:      "new_chapter",
		MangaID:   input.MangaID,
		Message:   input.Message,
		Timestamp: time.Now().Unix(),
	}

	if err := h.udpSrv.BroadcastNotification(n); err != nil {
		// Log the error but still return 200 — the chapter data itself is not
		// affected by a notification failure (graceful degradation per NFR).
		log.Printf("[NOTIFY] UDP broadcast failed for manga_id=%s: %v", input.MangaID, err)
		utils.Success(c, 200, "Chapter registered but notification broadcast failed", gin.H{
			"manga_id": input.MangaID,
			"notified": false,
		})
		return
	}

	utils.Success(c, 200, "Chapter notification broadcast sent", gin.H{
		"manga_id": input.MangaID,
		"notified": true,
	})
}
