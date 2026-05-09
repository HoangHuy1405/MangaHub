package models

import "time"

type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // never serialize to JSON
	CreatedAt    time.Time `json:"created_at"`
}

type UserProgress struct {
	UserID         int       `json:"user_id"`
	MangaID        string    `json:"manga_id"`
	CurrentChapter int       `json:"current_chapter"`
	Status         string    `json:"status"` // reading, completed, plan_to_read
	UpdatedAt      time.Time `json:"updated_at"`
}

type RegisterInput struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type AddLibraryInput struct {
	MangaID string `json:"manga_id" binding:"required"`
	Status  string `json:"status"   binding:"required,oneof=reading completed plan_to_read"`
}

type UpdateProgressInput struct {
	MangaID        string `json:"manga_id"        binding:"required"`
	CurrentChapter int    `json:"current_chapter"  binding:"required,gt=0"`
}
