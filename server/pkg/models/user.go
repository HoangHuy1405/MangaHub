package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialize to JSON
	CreatedAt    time.Time `json:"created_at"`
}

type UserProgress struct {
	UserID         int       `json:"user_id"`
	MangaID        string    `json:"manga_id"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status"` // reading, completed, plan_to_read
	Rating         int       `json:"rating"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

type LoginInput struct {
	Username string `json:"username"` // Optional if email is provided
	Email    string `json:"email"`    // Optional if username is provided
	Password string `json:"password" binding:"required"`
}

type AddLibraryInput struct {
	MangaID string `json:"manga_id" binding:"required"`
	Status  string `json:"status"   binding:"required,oneof=reading completed plan_to_read on_hold dropped"`
	Rating  int    `json:"rating"   binding:"omitempty,min=1,max=10"`
}

type UpdateLibraryInput struct {
	MangaID string `json:"manga_id" binding:"required"`
	Status  string `json:"status"   binding:"required,oneof=reading completed plan_to_read on_hold dropped"`
	Rating  int    `json:"rating"   binding:"omitempty,min=1,max=10"`
}

type UpdateProgressInput struct {
	MangaID        string `json:"manga_id"        binding:"required"`
	CurrentChapter int    `json:"current_chapter"  binding:"required,gt=0"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}
