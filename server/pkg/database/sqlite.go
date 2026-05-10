package database

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"

	_ "github.com/glebarez/go-sqlite"

	"mangahub/pkg/models"
)

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		log.Printf("[WARN] Failed to set WAL mode: %v", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id            INTEGER PRIMARY KEY AUTOINCREMENT,
		username      TEXT    NOT NULL UNIQUE,
		email         TEXT    NOT NULL UNIQUE,
		password_hash TEXT    NOT NULL,
		created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS manga (
		id                TEXT PRIMARY KEY,
		title             TEXT NOT NULL,
		author            TEXT NOT NULL,
		genres            TEXT NOT NULL DEFAULT '[]',
		status            TEXT NOT NULL CHECK(status IN ('ongoing', 'completed', 'hiatus', 'cancelled')),
		total_chapters    INTEGER NOT NULL DEFAULT 0,
		description       TEXT NOT NULL DEFAULT '',
		cover_url         TEXT NOT NULL DEFAULT '',
		year              INTEGER NOT NULL DEFAULT 0,
		content_rating    TEXT NOT NULL DEFAULT 'safe',
		demographic       TEXT NOT NULL DEFAULT '',
		original_language TEXT NOT NULL DEFAULT 'ja'
	);

	CREATE TABLE IF NOT EXISTS manga_relationships (
		id           TEXT NOT NULL,
		manga_id     TEXT NOT NULL,
		type         TEXT NOT NULL CHECK(type IN ('author', 'artist', 'cover_art', 'manga')),
		related_type TEXT,
		name         TEXT,
		biography    TEXT,
		twitter      TEXT,
		pixiv        TEXT,
		website      TEXT,
		file_name    TEXT,
		volume       TEXT,
		locale       TEXT,
		cover_url    TEXT,
		PRIMARY KEY (id, manga_id, type),
		FOREIGN KEY (manga_id) REFERENCES manga(id)
	);

	CREATE TABLE IF NOT EXISTS user_progress (
		user_id         INTEGER NOT NULL,
		manga_id        TEXT    NOT NULL,
		current_chapter INTEGER NOT NULL DEFAULT 0,
		status          TEXT    NOT NULL CHECK(status IN ('reading', 'completed', 'plan_to_read', 'on_hold', 'dropped')),
		rating          INTEGER DEFAULT 0,
		updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (user_id, manga_id),
		FOREIGN KEY (user_id)  REFERENCES users(id),
		FOREIGN KEY (manga_id) REFERENCES manga(id)
	);
	`

	if _, err := db.Exec(schema); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("[DB] Database initialized successfully")
	return db, nil
}

// Rule D7 — idempotent: only seeds when manga table is empty
func SeedManga(db *sql.DB, jsonPath string) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM manga").Scan(&count); err != nil {
		return fmt.Errorf("failed to count manga: %w", err)
	}

	if count > 0 {
		log.Printf("[SEED] Manga table already has %d records, skipping seed", count)
		return nil
	}

	data, err := os.ReadFile(jsonPath)
	if err != nil {
		return fmt.Errorf("failed to read seed file %s: %w", jsonPath, err)
	}

	var mangaList []models.MangaSeed
	if err := json.Unmarshal(data, &mangaList); err != nil {
		return fmt.Errorf("failed to parse seed JSON: %w", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO manga
			(id, title, author, genres, status, total_chapters, description, cover_url, year, content_rating, demographic, original_language)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare seed statement: %w", err)
	}
	defer stmt.Close()

	relStmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO manga_relationships
			(id, manga_id, type, related_type, name, biography, twitter, pixiv, website, file_name, volume, locale, cover_url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare relationship statement: %w", err)
	}
	defer relStmt.Close()

	for _, m := range mangaList {
		// Rule D6 — marshal []string genres to JSON TEXT for SQLite storage
		genresJSON, err := json.Marshal(m.Genres)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to marshal genres for %s: %w", m.ID, err)
		}
		if _, err := stmt.Exec(
			m.ID, m.Title, m.Author, string(genresJSON), m.Status, m.TotalChapters,
			m.Description, m.CoverURL, m.Year, m.ContentRating, m.Demographic, m.OriginalLanguage,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert manga %s: %w", m.ID, err)
		}

		for _, r := range m.Relationships {
			if _, err := relStmt.Exec(
				r.ID, m.ID, r.Type, r.RelatedType,
				r.Name, r.Biography, r.Twitter, r.Pixiv, r.Website,
				r.FileName, r.Volume, r.Locale, r.CoverURL,
			); err != nil {
				tx.Rollback()
				return fmt.Errorf("failed to insert relationship %s for manga %s: %w", r.ID, m.ID, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit seed transaction: %w", err)
	}

	log.Printf("[SEED] Successfully seeded %d manga records", len(mangaList))
	return nil
}
