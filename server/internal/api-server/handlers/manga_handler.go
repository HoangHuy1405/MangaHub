package handlers

import (
	"errors"
	"log"
	"math"
	"strconv"

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
	author := c.Query("author")
	yearFromStr := c.Query("year-from")
	yearToStr := c.Query("year-to")
	minChaptersStr := c.Query("min-chapters")
	sortBy := c.Query("sort-by")
	order := c.Query("order")

	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}
	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 {
		pageSize = 20
	}

	yearFrom, _ := strconv.Atoi(yearFromStr)
	yearTo, _ := strconv.Atoi(yearToStr)
	minChapters, _ := strconv.Atoi(minChaptersStr)

	log.Printf("[MANGA] Filters — genre=%q, status=%q, search=%q, page=%d, pageSize=%d", genre, status, search, page, pageSize)

	mangaList, total, err := h.service.GetAllManga(genre, status, search, author, yearFrom, yearTo, minChapters, sortBy, order, page, pageSize)
	if err != nil {
		log.Printf("[MANGA] Service error: %v", err)
		utils.InternalError(c, "Failed to fetch manga list")
		return
	}

	pages := int(math.Ceil(float64(total) / float64(pageSize)))

	log.Printf("[MANGA] Returning %d manga records", len(mangaList))

	utils.Success(c, 200, "Manga list retrieved successfully", gin.H{
		"manga": mangaList,
		"meta": gin.H{
			"page":     page,
			"pageSize": pageSize,
			"pages":    pages,
			"total":    total,
		},
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
