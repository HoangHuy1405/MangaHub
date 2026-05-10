package service

import (
	"fmt"

	"mangahub/internal/repository"
	"mangahub/pkg/models"
)

type MangaService interface {
	GetAllManga(genre, status, search, author string, yearFrom, yearTo, minChapters int, sortBy, order string, page, pageSize int) ([]models.Manga, int, error)
	GetMangaByID(id string) (*models.MangaDetail, error)
}

type mangaServiceImpl struct {
	repo repository.MangaRepository
}

func NewMangaService(repo repository.MangaRepository) MangaService {
	return &mangaServiceImpl{repo: repo}
}

func (s *mangaServiceImpl) GetAllManga(genre, status, search, author string, yearFrom, yearTo, minChapters int, sortBy, order string, page, pageSize int) ([]models.Manga, int, error) {
	mangaList, total, err := s.repo.GetAll(genre, status, search, author, yearFrom, yearTo, minChapters, sortBy, order, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("get all manga: %w", err)
	}
	return mangaList, total, nil
}

func (s *mangaServiceImpl) GetMangaByID(id string) (*models.MangaDetail, error) {
	manga, err := s.repo.FindByID(id)
	if err != nil {
		return nil, fmt.Errorf("find manga by id: %w", err)
	}
	if manga == nil {
		return nil, ErrNotFound
	}

	rels, err := s.repo.GetRelationships(id)
	if err != nil {
		return nil, fmt.Errorf("get manga relationships: %w", err)
	}

	return &models.MangaDetail{
		Manga:         *manga,
		Relationships: rels,
	}, nil
}
