package handlers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"mangahub/internal/service"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(svc service.UserService) *UserHandler {
	return &UserHandler{service: svc}
}

func (h *UserHandler) AddToLibrary(c *gin.Context) {
	userID := c.GetInt("user_id")
	log.Printf("[USER] AddToLibrary request from user_id=%d", userID)

	var input models.AddLibraryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[USER] AddToLibrary validation failed: %v", err)
		utils.BadRequest(c, "Invalid input: manga_id (required) and status (reading/completed/plan_to_read) are required")
		return
	}

	err := h.service.AddToLibrary(userID, input)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("[USER] Manga not found: manga_id=%s", input.MangaID)
			utils.NotFound(c, "Manga not found")
			return
		}
		log.Printf("[USER] Service error: %v", err)
		utils.InternalError(c, "Failed to add manga to library")
		return
	}

	log.Printf("[USER] Manga added to library: user_id=%d, manga_id=%s, status=%s", userID, input.MangaID, input.Status)

	utils.Created(c, "Manga added to library successfully", gin.H{
		"user_id":  userID,
		"manga_id": input.MangaID,
		"status":   input.Status,
	})
}

func (h *UserHandler) GetLibrary(c *gin.Context) {
	userID := c.GetInt("user_id")
	statusFilter := c.Query("status")
	sortBy := c.Query("sort_by")
	order := c.Query("order")
	log.Printf("[USER] GetLibrary request from user_id=%d, status_filter=%q", userID, statusFilter)

	entries, err := h.service.GetLibrary(userID, statusFilter, sortBy, order)
	if err != nil {
		log.Printf("[USER] Service error: %v", err)
		utils.InternalError(c, "Failed to fetch library")
		return
	}

	log.Printf("[USER] Returning %d library entries for user_id=%d", len(entries), userID)

	utils.Success(c, 200, "Library retrieved successfully", gin.H{
		"library": entries,
		"count":   len(entries),
	})
}

func (h *UserHandler) UpdateLibrary(c *gin.Context) {
	userID := c.GetInt("user_id")
	mangaID := c.Param("id")

	var input models.UpdateLibraryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Invalid input")
		return
	}
	input.MangaID = mangaID

	err := h.service.UpdateLibrary(userID, input)
	if err != nil {
		utils.InternalError(c, "Failed to update library")
		return
	}
	utils.Success(c, 200, "Library updated", gin.H{})
}

func (h *UserHandler) RemoveFromLibrary(c *gin.Context) {
	userID := c.GetInt("user_id")
	mangaID := c.Param("id")

	err := h.service.RemoveFromLibrary(userID, mangaID)
	if err != nil {
		utils.InternalError(c, "Failed to remove from library")
		return
	}
	utils.Success(c, 200, "Removed from library", gin.H{})
}

func (h *UserHandler) UpdateProgress(c *gin.Context) {
	userID := c.GetInt("user_id")
	log.Printf("[USER] UpdateProgress request from user_id=%d", userID)

	var input models.UpdateProgressInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[USER] UpdateProgress validation failed: %v", err)
		utils.BadRequest(c, "Invalid input: manga_id (required) and current_chapter (required, > 0) are required")
		return
	}

	err := h.service.UpdateProgress(userID, input)
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			log.Printf("[USER] Progress not found: user_id=%d, manga_id=%s", userID, input.MangaID)
			utils.NotFound(c, "Manga not found in your library. Add it first using POST /users/library")
			return
		}
		log.Printf("[USER] Service error: %v", err)
		utils.InternalError(c, "Failed to update reading progress")
		return
	}

	log.Printf("[USER] Progress updated: user_id=%d, manga_id=%s, chapter=%d", userID, input.MangaID, input.CurrentChapter)

	utils.Success(c, 200, "Reading progress updated successfully", gin.H{
		"user_id":         userID,
		"manga_id":        input.MangaID,
		"current_chapter": input.CurrentChapter,
	})
}
