// Package services contains the business logic layer for Arkheion.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mathornton01/arkheion/config"
)

// ErrISBNNotFound is returned when neither OpenLibrary nor Google Books can find the ISBN.
var ErrISBNNotFound = errors.New("ISBN not found in any catalog")

// ISBNLookupResult is the normalized metadata returned by the ISBN lookup service.
// Field names match the CreateBookRequest to make form pre-fill trivial on the frontend.
type ISBNLookupResult struct {
	ISBN          string   `json:"isbn"`
	Title         string   `json:"title"`
	Subtitle      string   `json:"subtitle,omitempty"`
	Authors       []string `json:"authors"`
	Publisher     string   `json:"publisher,omitempty"`
	PublishedDate string   `json:"published_date,omitempty"` // ISO date string
	Description   string   `json:"description,omitempty"`
	PageCount     int      `json:"page_count,omitempty"`
	Categories    []string `json:"categories,omitempty"`
	Language      string   `json:"language,omitempty"`
	CoverURL      string   `json:"cover_url,omitempty"`
	Source        string   `json:"source"` // "openlibrary" or "google_books"
}

// ISBNService looks up book metadata by ISBN using OpenLibrary and Google Books.
type ISBNService struct {
	cfg    *config.Config
	client *http.Client
}

// NewISBNService creates a new ISBNService.
func NewISBNService(cfg *config.Config) *ISBNService {
	return &ISBNService{
		cfg: cfg,
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Lookup queries OpenLibrary first, then falls back to Google Books.
func (s *ISBNService) Lookup(ctx context.Context, isbn string) (*ISBNLookupResult, error) {
	// Try OpenLibrary first (no API key required)
	result, err := s.lookupOpenLibrary(ctx, isbn)
	if err == nil && result != nil {
		return result, nil
	}

	// Fall back to Google Books
	result, err = s.lookupGoogleBooks(ctx, isbn)
	if err == nil && result != nil {
		return result, nil
	}

	return nil, ErrISBNNotFound
}

// lookupOpenLibrary queries the OpenLibrary Books API.
// API docs: https://openlibrary.org/dev/docs/api
func (s *ISBNService) lookupOpenLibrary(ctx context.Context, isbn string) (*ISBNLookupResult, error) {
	url := fmt.Sprintf("%s/api/books?bibkeys=ISBN:%s&jscmd=data&format=json",
		s.cfg.OpenLibraryBaseURL, isbn)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Arkheion/1.0 (https://github.com/mathornton01/arkheion)")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenLibrary request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenLibrary returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err != nil {
		return nil, err
	}

	var data map[string]openLibraryBook
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	key := "ISBN:" + isbn
	book, ok := data[key]
	if !ok {
		return nil, ErrISBNNotFound
	}

	result := &ISBNLookupResult{
		ISBN:      isbn,
		Title:     book.Title,
		Subtitle:  book.Subtitle,
		PageCount: book.NumberOfPages,
		Source:    "openlibrary",
	}

	for _, a := range book.Authors {
		result.Authors = append(result.Authors, a.Name)
	}
	if len(book.Publishers) > 0 {
		result.Publisher = book.Publishers[0].Name
	}
	if book.PublishDate != "" {
		result.PublishedDate = parseFlexDate(book.PublishDate)
	}
	if len(book.Subjects) > 0 {
		for _, s := range book.Subjects {
			result.Categories = append(result.Categories, s.Name)
			if len(result.Categories) >= 5 {
				break
			}
		}
	}
	if book.Cover.Large != "" {
		result.CoverURL = book.Cover.Large
	} else if book.Cover.Medium != "" {
		result.CoverURL = book.Cover.Medium
	}

	return result, nil
}

// lookupGoogleBooks queries the Google Books API.
// API docs: https://developers.google.com/books/docs/v1/reference/volumes/list
func (s *ISBNService) lookupGoogleBooks(ctx context.Context, isbn string) (*ISBNLookupResult, error) {
	url := fmt.Sprintf("%s/volumes?q=isbn:%s", s.cfg.GoogleBooksBaseURL, isbn)
	if s.cfg.GoogleBooksAPIKey != "" {
		url += "&key=" + s.cfg.GoogleBooksAPIKey
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Google Books request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Google Books returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}

	var data googleBooksResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	if data.TotalItems == 0 || len(data.Items) == 0 {
		return nil, ErrISBNNotFound
	}

	info := data.Items[0].VolumeInfo

	result := &ISBNLookupResult{
		ISBN:          isbn,
		Title:         info.Title,
		Subtitle:      info.Subtitle,
		Authors:       info.Authors,
		Publisher:     info.Publisher,
		PublishedDate: info.PublishedDate,
		Description:   info.Description,
		PageCount:     info.PageCount,
		Categories:    info.Categories,
		Language:      info.Language,
		Source:        "google_books",
	}

	if info.ImageLinks.Large != "" {
		result.CoverURL = info.ImageLinks.Large
	} else if info.ImageLinks.Thumbnail != "" {
		result.CoverURL = info.ImageLinks.Thumbnail
	}

	return result, nil
}

// parseFlexDate tries to parse common OpenLibrary date formats into ISO 8601.
func parseFlexDate(s string) string {
	formats := []string{"January 2, 2006", "2006", "January 2006", "2006-01-02"}
	for _, f := range formats {
		if t, err := time.Parse(f, strings.TrimSpace(s)); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return ""
}

// --- OpenLibrary API response types ------------------------------------------

type openLibraryBook struct {
	Title         string                 `json:"title"`
	Subtitle      string                 `json:"subtitle"`
	Authors       []openLibraryEntity    `json:"authors"`
	Publishers    []openLibraryEntity    `json:"publishers"`
	PublishDate   string                 `json:"publish_date"`
	NumberOfPages int                    `json:"number_of_pages"`
	Subjects      []openLibraryEntity    `json:"subjects"`
	Cover         openLibraryCover       `json:"cover"`
}

type openLibraryEntity struct {
	Name string `json:"name"`
}

type openLibraryCover struct {
	Small  string `json:"small"`
	Medium string `json:"medium"`
	Large  string `json:"large"`
}

// --- Google Books API response types -----------------------------------------

type googleBooksResponse struct {
	TotalItems int                `json:"totalItems"`
	Items      []googleBooksItem  `json:"items"`
}

type googleBooksItem struct {
	VolumeInfo googleBooksVolumeInfo `json:"volumeInfo"`
}

type googleBooksVolumeInfo struct {
	Title         string                  `json:"title"`
	Subtitle      string                  `json:"subtitle"`
	Authors       []string                `json:"authors"`
	Publisher     string                  `json:"publisher"`
	PublishedDate string                  `json:"publishedDate"`
	Description   string                  `json:"description"`
	PageCount     int                     `json:"pageCount"`
	Categories    []string                `json:"categories"`
	Language      string                  `json:"language"`
	ImageLinks    googleBooksImageLinks   `json:"imageLinks"`
}

type googleBooksImageLinks struct {
	Thumbnail string `json:"thumbnail"`
	Large     string `json:"large"`
}
