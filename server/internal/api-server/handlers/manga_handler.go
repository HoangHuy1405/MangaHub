package handlers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"mangahub/internal/service"
	"mangahub/pkg/utils"
)

type MangaHandler struct {
	service service.MangaService
}

func NewMangaHandler(svc service.MangaService) *MangaHandler {
	return &MangaHandler{service: svc}
}

func (h *MangaHandler) GetAllManga(c *gin.Context) {
	log.Println("[MANGA] GetAllManga request received")

	genre := c.Query("genre")
	status := c.Query("status")
	search := c.Query("search")

	log.Printf("[MANGA] Filters — genre=%q, status=%q, search=%q", genre, status, search)

	mangaList, err := h.service.GetAllManga(genre, status, search)
	if err != nil {
		log.Printf("[MANGA] Service error: %v", err)
		utils.InternalError(c, "Failed to fetch manga list")
		return
	}

	log.Printf("[MANGA] Returning %d manga records", len(mangaList))

	utils.Success(c, 200, "Manga list retrieved successfully", gin.H{
		"manga": mangaList,
		"count": len(mangaList),
	})
}

func (h *MangaHandler) GetMangaByID(c *gin.Context) {
	id := c.Param("id")
	log.Printf("[MANGA] GetMangaByID request received: id=%s", id)

	detail, err := h.service.GetMangaByID(id)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("[MANGA] Manga not found: id=%s", id)
			utils.NotFound(c, "Manga not found")
			return
		}
		log.Printf("[MANGA] Service error: %v", err)
		utils.InternalError(c, "Failed to fetch manga")
		return
	}

	log.Printf("[MANGA] Manga found: id=%s, relationships=%d", detail.ID, len(detail.Relationships))

	utils.Success(c, 200, "Manga retrieved successfully", detail)
}
