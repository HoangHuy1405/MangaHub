package handlers

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"

	"mangahub/internal/service"
	"mangahub/pkg/models"
	"mangahub/pkg/utils"
)

type AuthHandler struct {
	service service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{service: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	log.Println("[AUTH] Register request received")

	var input models.RegisterInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[AUTH] Register validation failed: %v", err)
		utils.BadRequest(c, "Invalid input: username (min 3 chars) and password (min 6 chars) are required")
		return
	}

	id, err := h.service.Register(input)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			log.Printf("[AUTH] Username already exists: %s", input.Username)
			utils.Conflict(c, "Username already exists")
			return
		}
		log.Printf("[AUTH] Service error: %v", err)
		utils.InternalError(c, "Failed to process registration")
		return
	}

	log.Printf("[AUTH] User registered successfully: id=%d, username=%s", id, input.Username)

	utils.Created(c, "User registered successfully", gin.H{
		"id":       id,
		"username": input.Username,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	log.Println("[AUTH] Login request received")

	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[AUTH] Login validation failed: %v", err)
		utils.BadRequest(c, "Username and password are required")
		return
	}

	tokenString, user, err := h.service.Login(input)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			log.Printf("[AUTH] Login failed for user: %s", input.Username)
			utils.Unauthorized(c, "Invalid username or password")
			return
		}
		log.Printf("[AUTH] Service error: %v", err)
		utils.InternalError(c, "Login failed")
		return
	}

	log.Printf("[AUTH] User logged in successfully: id=%d, username=%s", user.ID, user.Username)

	utils.Success(c, 200, "Login successful", gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
		},
	})
}
