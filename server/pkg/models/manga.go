package models

// Manga is the runtime model — Genres is a JSON string as stored in SQLite.
type Manga struct {
	ID               string `json:"id"`
	Title            string `json:"title"`
	Author           string `json:"author"`
	Genres           string `json:"genres"` // JSON array stored as TEXT in SQLite
	Status           string `json:"status"`
	TotalChapters    int    `json:"total_chapters"`
	Description      string `json:"description"`
	CoverURL         string `json:"cover_url"`
	Year             int    `json:"year"`
	ContentRating    string `json:"content_rating"`
	Demographic      string `json:"demographic"`
	OriginalLanguage string `json:"original_language"`
}

// MangaRelationship is a linked entity for a manga.
// Nullable *string fields are omitted from JSON when nil, keeping the payload clean per type:
//   - author/artist  → Name, Biography, Twitter, Pixiv, Website
//   - cover_art      → FileName, Volume, Locale, CoverURL
//   - manga          → RelatedType (sequel/prequel/spin_off/etc.)
type MangaRelationship struct {
	ID          string  `json:"id"`
	Type        string  `json:"type"`
	RelatedType *string `json:"related_type,omitempty"`
	Name        *string `json:"name,omitempty"`
	Biography   *string `json:"biography,omitempty"`
	Twitter     *string `json:"twitter,omitempty"`
	Pixiv       *string `json:"pixiv,omitempty"`
	Website     *string `json:"website,omitempty"`
	FileName    *string `json:"file_name,omitempty"`
	Volume      *string `json:"volume,omitempty"`
	Locale      *string `json:"locale,omitempty"`
	CoverURL    *string `json:"cover_url,omitempty"`
}

// MangaDetail is the enriched API response for a single manga — includes relationship graph.
type MangaDetail struct {
	Manga
	Relationships []MangaRelationship `json:"relationships"`
}

// MangaSeed matches the JSON structure of data/manga_seed.json.
// Genres is a proper []string; SeedManga marshals it to TEXT before INSERT.
type MangaSeed struct {
	ID               string              `json:"id"`
	Title            string              `json:"title"`
	Author           string              `json:"author"`
	Genres           []string            `json:"genres"`
	Status           string              `json:"status"`
	TotalChapters    int                 `json:"total_chapters"`
	Description      string              `json:"description"`
	CoverURL         string              `json:"cover_url"`
	Year             int                 `json:"year"`
	ContentRating    string              `json:"content_rating"`
	Demographic      string              `json:"demographic"`
	OriginalLanguage string              `json:"original_language"`
	Relationships    []MangaRelationship `json:"relationships"`
}
