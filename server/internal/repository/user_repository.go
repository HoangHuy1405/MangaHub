package repository

import (
	"database/sql"
	"fmt"
)

// LibraryEntry is the joined result of user_progress + manga for library queries.
type LibraryEntry struct {
	UserID         int    `json:"user_id"`
	MangaID        string `json:"manga_id"`
	CurrentChapter int    `json:"current_chapter"`
	Status         string `json:"status"`
	UpdatedAt      string `json:"updated_at"`
	MangaTitle     string `json:"manga_title"`
	MangaAuthor    string `json:"manga_author"`
	MangaGenres    string `json:"manga_genres"`
	MangaCoverURL  string `json:"manga_cover_url"`
}

type UserRepository interface {
	AddToLibrary(userID int, mangaID, status string) error
	GetLibrary(userID int, statusFilter string) ([]LibraryEntry, error)
	UpdateProgress(userID int, mangaID string, chapter int) (int64, error)
}

type userRepositoryImpl struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) UserRepository {
	return &userRepositoryImpl{db: db}
}

func (r *userRepositoryImpl) AddToLibrary(userID int, mangaID, status string) error {
	_, err := r.db.Exec(
		`INSERT INTO user_progress (user_id, manga_id, current_chapter, status)
		 VALUES (?, ?, 0, ?)
		 ON CONFLICT(user_id, manga_id) DO UPDATE SET status = excluded.status, updated_at = CURRENT_TIMESTAMP`,
		userID, mangaID, status,
	)
	if err != nil {
		return fmt.Errorf("insert user library entry: %w", err)
	}
	return nil
}

func (r *userRepositoryImpl) GetLibrary(userID int, statusFilter string) ([]LibraryEntry, error) {
	query := `SELECT up.user_id, up.manga_id, up.current_chapter, up.status, up.updated_at,
	                 m.title, m.author, m.genres, m.cover_url
	          FROM user_progress up
	          JOIN manga m ON up.manga_id = m.id
	          WHERE up.user_id = ?`
	args := []interface{}{userID}

	if statusFilter != "" {
		query += " AND up.status = ?"
		args = append(args, statusFilter)
	}

	query += " ORDER BY up.updated_at DESC"

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query user library: %w", err)
	}
	defer rows.Close()

	var entries []LibraryEntry
	for rows.Next() {
		var e LibraryEntry
		if err := rows.Scan(
			&e.UserID, &e.MangaID, &e.CurrentChapter, &e.Status, &e.UpdatedAt,
			&e.MangaTitle, &e.MangaAuthor, &e.MangaGenres, &e.MangaCoverURL,
		); err != nil {
			return nil, fmt.Errorf("scan library entry: %w", err)
		}
		entries = append(entries, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate library rows: %w", err)
	}

	return entries, nil
}

func (r *userRepositoryImpl) UpdateProgress(userID int, mangaID string, chapter int) (int64, error) {
	result, err := r.db.Exec(
		`UPDATE user_progress SET current_chapter = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = ? AND manga_id = ?`,
		chapter, userID, mangaID,
	)
	if err != nil {
		return 0, fmt.Errorf("update user progress: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rowsAffected, nil
}
