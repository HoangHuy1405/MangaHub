package repository

import (
	"database/sql"
	"fmt"

	"mangahub/pkg/models"
)

type MangaRepository interface {
	GetAll(genre, status, search string) ([]models.Manga, error)
	FindByID(id string) (*models.Manga, error)
	GetRelationships(mangaID string) ([]models.MangaRelationship, error)
	Exists(id string) (bool, error)
}

type mangaRepositoryImpl struct {
	db *sql.DB
}

func NewMangaRepository(db *sql.DB) MangaRepository {
	return &mangaRepositoryImpl{db: db}
}

func (r *mangaRepositoryImpl) GetAll(genre, status, search string) ([]models.Manga, error) {
	query := `SELECT id, title, author, genres, status, total_chapters, description, cover_url,
	           year, content_rating, demographic, original_language FROM manga WHERE 1=1`
	args := []interface{}{}

	if genre != "" {
		query += " AND genres LIKE ?"
		args = append(args, "%\""+genre+"\"%")
	}
	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if search != "" {
		query += " AND (title LIKE ? OR author LIKE ?)"
		args = append(args, "%"+search+"%", "%"+search+"%")
	}

	query += " ORDER BY title ASC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query manga list: %w", err)
	}
	defer rows.Close()

	var mangaList []models.Manga
	for rows.Next() {
		var m models.Manga
		if err := rows.Scan(
			&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters,
			&m.Description, &m.CoverURL, &m.Year, &m.ContentRating, &m.Demographic, &m.OriginalLanguage,
		); err != nil {
			return nil, fmt.Errorf("scan manga row: %w", err)
		}
		mangaList = append(mangaList, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate manga rows: %w", err)
	}

	return mangaList, nil
}

// nullStr converts sql.NullString to *string for nullable model fields.
func nullStr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

func (r *mangaRepositoryImpl) FindByID(id string) (*models.Manga, error) {
	var m models.Manga
	err := r.db.QueryRow(`
		SELECT id, title, author, genres, status, total_chapters, description, cover_url,
		       year, content_rating, demographic, original_language
		FROM manga WHERE id = ?`, id,
	).Scan(
		&m.ID, &m.Title, &m.Author, &m.Genres, &m.Status, &m.TotalChapters,
		&m.Description, &m.CoverURL, &m.Year, &m.ContentRating, &m.Demographic, &m.OriginalLanguage,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query manga by id: %w", err)
	}

	return &m, nil
}

func (r *mangaRepositoryImpl) GetRelationships(mangaID string) ([]models.MangaRelationship, error) {
	rows, err := r.db.Query(`
		SELECT id, type, related_type, name, biography, twitter, pixiv, website,
		       file_name, volume, locale, cover_url
		FROM manga_relationships WHERE manga_id = ? ORDER BY type`, mangaID)
	if err != nil {
		return nil, fmt.Errorf("query manga relationships: %w", err)
	}
	defer rows.Close()

	var rels []models.MangaRelationship
	for rows.Next() {
		var rel models.MangaRelationship
		var relType, name, bio, twitter, pixiv, website, fileName, volume, locale, coverURL sql.NullString
		if err := rows.Scan(
			&rel.ID, &rel.Type, &relType, &name, &bio, &twitter, &pixiv, &website,
			&fileName, &volume, &locale, &coverURL,
		); err != nil {
			return nil, fmt.Errorf("scan relationship row: %w", err)
		}
		rel.RelatedType = nullStr(relType)
		rel.Name = nullStr(name)
		rel.Biography = nullStr(bio)
		rel.Twitter = nullStr(twitter)
		rel.Pixiv = nullStr(pixiv)
		rel.Website = nullStr(website)
		rel.FileName = nullStr(fileName)
		rel.Volume = nullStr(volume)
		rel.Locale = nullStr(locale)
		rel.CoverURL = nullStr(coverURL)
		rels = append(rels, rel)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate relationship rows: %w", err)
	}

	return rels, nil
}

func (r *mangaRepositoryImpl) Exists(id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM manga WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check manga existence: %w", err)
	}
	return exists, nil
}
