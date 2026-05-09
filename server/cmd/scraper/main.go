package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"mangahub/pkg/models"
)

// MangaDex API response types
type mdResponse struct {
	Data  []mdManga `json:"data"`
	Total int       `json:"total"`
}

type mdManga struct {
	ID            string           `json:"id"`
	Attributes    mdAttributes     `json:"attributes"`
	Relationships []mdRelationship `json:"relationships"`
}

type mdAttributes struct {
	Title                  map[string]string `json:"title"`
	Description            map[string]string `json:"description"`
	Status                 string            `json:"status"`
	Tags                   []mdTag           `json:"tags"`
	LastChapter            *string           `json:"lastChapter"`
	PublicationDemographic *string           `json:"publicationDemographic"`
	Year                   int               `json:"year"`
	ContentRating          string            `json:"contentRating"`
	OriginalLanguage       string            `json:"originalLanguage"`
}

type mdTag struct {
	Attributes mdTagAttribs `json:"attributes"`
}

type mdTagAttribs struct {
	Name  map[string]string `json:"name"`
	Group string            `json:"group"`
}

type mdRelationship struct {
	ID         string        `json:"id"`
	Type       string        `json:"type"`
	Related    string        `json:"related"` // filled for type="manga" (sequel/prequel/etc.)
	Attributes *mdRelAttribs `json:"attributes"`
}

type mdRelAttribs struct {
	// Person (author/artist)
	Name      string            `json:"name"`
	Biography map[string]string `json:"biography"`
	Twitter   string            `json:"twitter"`
	Pixiv     *string           `json:"pixiv"`
	Website   string            `json:"website"`
	// Cover art
	FileName string `json:"fileName"`
	Volume   string `json:"volume"`
	Locale   string `json:"locale"`
}

var (
	client        = &http.Client{Timeout: 15 * time.Second}
	nonAlphanumRe = regexp.MustCompile(`[^a-z0-9]+`)
	mdLinkRe      = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
)

func slugify(s string) string {
	s = strings.ToLower(s)
	s = strings.Map(func(r rune) rune {
		if r > 127 {
			return '-'
		}
		return r
	}, s)
	s = nonAlphanumRe.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func cleanDesc(s string) string {
	s = mdLinkRe.ReplaceAllString(s, "$1")
	s = strings.TrimSpace(s)
	if len(s) > 400 {
		s = s[:400]
		if idx := strings.LastIndex(s, " "); idx > 300 {
			s = s[:idx]
		}
		s += "..."
	}
	return s
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func ptrStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func fetchManga(demographic string, offset int) ([]mdManga, int, error) {
	url := fmt.Sprintf(
		"https://api.mangadex.org/manga?publicationDemographic[]=%s&limit=25&offset=%d"+
			"&availableTranslatedLanguage[]=en&includes[]=author&includes[]=artist&includes[]=cover_art"+
			"&contentRating[]=safe&contentRating[]=suggestive&order[followedCount]=desc",
		demographic, offset,
	)

	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, 0, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result mdResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, 0, fmt.Errorf("decode failed: %w", err)
	}
	return result.Data, result.Total, nil
}

func convert(m mdManga, demographic string) (models.MangaSeed, bool) {
	title := m.Attributes.Title["en"]
	if title == "" {
		title = m.Attributes.Title["ja-ro"]
	}
	if title == "" {
		for _, t := range m.Attributes.Title {
			title = t
			break
		}
	}
	if title == "" {
		return models.MangaSeed{}, false
	}

	desc := cleanDesc(m.Attributes.Description["en"])

	genres := []string{capitalize(demographic)}
	for _, tag := range m.Attributes.Tags {
		if tag.Attributes.Group == "genre" {
			if name := tag.Attributes.Name["en"]; name != "" {
				genres = append(genres, name)
			}
		}
	}

	chapters := 0
	if m.Attributes.LastChapter != nil && *m.Attributes.LastChapter != "" {
		if f, err := strconv.ParseFloat(*m.Attributes.LastChapter, 64); err == nil {
			chapters = int(f)
		}
	}

	status := m.Attributes.Status
	if status == "cancelled" {
		status = "completed"
	}

	contentRating := m.Attributes.ContentRating
	if contentRating == "" {
		contentRating = "safe"
	}

	// Build relationship list for all 4 types.
	// author is still set from the first author relationship for backward compatibility.
	author := "Unknown"
	coverURL := ""
	var rels []models.MangaRelationship

	for _, rel := range m.Relationships {
		if rel.ID == "" {
			continue
		}
		r := models.MangaRelationship{ID: rel.ID, Type: rel.Type}

		switch rel.Type {
		case "author", "artist":
			if rel.Attributes == nil || rel.Attributes.Name == "" {
				continue
			}
			if rel.Type == "author" && author == "Unknown" {
				author = rel.Attributes.Name
			}
			r.Name = ptrStr(rel.Attributes.Name)
			if bio := rel.Attributes.Biography["en"]; bio != "" {
				r.Biography = ptrStr(bio)
			}
			r.Twitter = ptrStr(rel.Attributes.Twitter)
			if rel.Attributes.Pixiv != nil {
				r.Pixiv = rel.Attributes.Pixiv
			}
			r.Website = ptrStr(rel.Attributes.Website)

		case "cover_art":
			if rel.Attributes == nil || rel.Attributes.FileName == "" {
				continue
			}
			constructed := fmt.Sprintf(
				"https://uploads.mangadex.org/covers/%s/%s.256.jpg",
				m.ID, rel.Attributes.FileName,
			)
			if coverURL == "" {
				coverURL = constructed
			}
			r.FileName = ptrStr(rel.Attributes.FileName)
			r.Volume = ptrStr(rel.Attributes.Volume)
			r.Locale = ptrStr(rel.Attributes.Locale)
			r.CoverURL = ptrStr(constructed)

		case "manga":
			if rel.Related == "" {
				continue
			}
			r.RelatedType = ptrStr(rel.Related)

		default:
			continue
		}

		rels = append(rels, r)
	}

	return models.MangaSeed{
		ID:               slugify(title),
		Title:            title,
		Author:           author,
		Genres:           genres,
		Status:           status,
		TotalChapters:    chapters,
		Description:      desc,
		CoverURL:         coverURL,
		Year:             m.Attributes.Year,
		ContentRating:    contentRating,
		Demographic:      demographic,
		OriginalLanguage: m.Attributes.OriginalLanguage,
		Relationships:    rels,
	}, true
}

func main() {
	demographics := []string{"shounen", "shoujo", "seinen", "josei"}
	seen := map[string]bool{}
	var all []models.MangaSeed

	for _, demo := range demographics {
		collected := 0
		for offset := 0; collected < 50; offset += 25 {
			log.Printf("[SCRAPER] Fetching %s (offset=%d)...", demo, offset)

			manga, total, err := fetchManga(demo, offset)
			if err != nil {
				log.Printf("[SCRAPER] Error: %v", err)
				break
			}

			for _, m := range manga {
				seed, ok := convert(m, demo)
				if !ok || seen[seed.ID] {
					continue
				}
				seen[seed.ID] = true
				all = append(all, seed)
				collected++
			}

			log.Printf("[SCRAPER] %s: collected %d/%d (API total: %d)", demo, collected, 50, total)

			// Rule S1 — respect MangaDex rate limit
			time.Sleep(600 * time.Millisecond)

			if offset+25 >= total {
				break
			}
		}
		time.Sleep(1 * time.Second)
	}

	log.Printf("[SCRAPER] Total unique manga: %d", len(all))

	data, err := json.MarshalIndent(all, "", "  ")
	if err != nil {
		log.Fatalf("[SCRAPER] Marshal failed: %v", err)
	}

	if err := os.WriteFile("data/manga_seed.json", data, 0644); err != nil {
		log.Fatalf("[SCRAPER] Write failed: %v", err)
	}

	log.Printf("[SCRAPER] Saved %d manga to data/manga_seed.json", len(all))
}

