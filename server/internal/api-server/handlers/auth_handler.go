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
		utils.BadRequest(c, "Invalid input: username, email and password (min 8 chars) are required")
		return
	}

	id, err := h.service.Register(input)
	if err != nil {
		if errors.Is(err, service.ErrConflict) {
			log.Printf("[AUTH] Username or email already exists: %s, %s", input.Username, input.Email)
			utils.Conflict(c, "Username or email already exists")
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
		"email":    input.Email,
	})
}

func (h *AuthHandler) Login(c *gin.Context) {
	log.Println("[AUTH] Login request received")

	var input models.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		log.Printf("[AUTH] Login validation failed: %v", err)
		utils.BadRequest(c, "Username (or email) and password are required")
		return
	}

	tokenString, user, err := h.service.Login(input)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			log.Printf("[AUTH] Login failed for user/email: %s %s", input.Username, input.Email)
			utils.Unauthorized(c, "Invalid credentials")
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
