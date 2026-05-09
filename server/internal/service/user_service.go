package service

import (
	"fmt"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

type UserService interface {
	AddToLibrary(userID int, input models.AddLibraryInput) error
	GetLibrary(userID int, statusFilter string) ([]repository.LibraryEntry, error)
	UpdateProgress(userID int, input models.UpdateProgressInput) error
}

type userServiceImpl struct {
	userRepo  repository.UserRepository
	mangaRepo repository.MangaRepository
}

func NewUserService(userRepo repository.UserRepository, mangaRepo repository.MangaRepository) UserService {
	return &userServiceImpl{userRepo: userRepo, mangaRepo: mangaRepo}
}

func (s *userServiceImpl) AddToLibrary(userID int, input models.AddLibraryInput) error {
	exists, err := s.mangaRepo.Exists(input.MangaID)
	if err != nil {
		return fmt.Errorf("check manga existence: %w", err)
	}
	if !exists {
		return ErrNotFound
	}

	if err := s.userRepo.AddToLibrary(userID, input.MangaID, input.Status); err != nil {
		return fmt.Errorf("add to library: %w", err)
	}

	return nil
}

func (s *userServiceImpl) GetLibrary(userID int, statusFilter string) ([]repository.LibraryEntry, error) {
	entries, err := s.userRepo.GetLibrary(userID, statusFilter)
	if err != nil {
		return nil, fmt.Errorf("get library: %w", err)
	}
	return entries, nil
}

func (s *userServiceImpl) UpdateProgress(userID int, input models.UpdateProgressInput) error {
	rowsAffected, err := s.userRepo.UpdateProgress(userID, input.MangaID, input.CurrentChapter)
	if err != nil {
		return fmt.Errorf("update progress: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
