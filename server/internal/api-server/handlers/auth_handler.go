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

func (h *AuthHandler) Check(c *gin.Context) {
	username := c.Query("username")
	email := c.Query("email")

	if username == "" && email == "" {
		utils.BadRequest(c, "Username or email is required")
		return
	}

	if err := h.service.CheckAvailability(username, email); err != nil {
		if errors.Is(err, service.ErrConflict) {
			utils.Conflict(c, "Username or email already exists")
			return
		}
		log.Printf("[AUTH] Service error checking availability: %v", err)
		utils.InternalError(c, "Failed to check availability")
		return
	}

	utils.Success(c, 200, "Available", nil)
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

func (h *AuthHandler) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "unauthorized")
		return
	}

	var input models.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		utils.BadRequest(c, "Invalid input: old_password and new_password (min 8 chars) are required")
		return
	}

	uid, ok := userID.(int)
	if !ok {
		// In case jwt parses it as float64
		if f, ok := userID.(float64); ok {
			uid = int(f)
		} else {
			utils.InternalError(c, "invalid user id type")
			return
		}
	}

	if err := h.service.ChangePassword(int64(uid), input.OldPassword, input.NewPassword); err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			utils.Unauthorized(c, "Incorrect current password")
			return
		}
		log.Printf("[AUTH] Change password error: %v", err)
		utils.InternalError(c, "Failed to change password")
		return
	}

	utils.Success(c, 200, "Password changed successfully", nil)
}
