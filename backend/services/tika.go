package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/mathornton01/arkheion/config"
	"github.com/rs/zerolog/log"
)

// TikaService wraps the Apache Tika REST API for text extraction.
// Tika accepts file content via PUT /tika and returns plain text.
type TikaService struct {
	cfg    *config.Config
	client *http.Client
}

// NewTikaService creates a new TikaService.
func NewTikaService(cfg *config.Config) *TikaService {
	return &TikaService{
		cfg: cfg,
		client: &http.Client{
			Timeout: time.Duration(cfg.TikaTimeoutSeconds) * time.Second,
		},
	}
}

// ExtractText sends the file content to Apache Tika and returns the extracted plain text.
//
// fileType is used to set the appropriate Content-Type for the Tika request,
// which helps Tika choose the correct parser.
//
// Tika's plain text output may contain a large amount of whitespace and formatting
// artifacts; callers may want to normalize whitespace before storing.
func (s *TikaService) ExtractText(ctx context.Context, reader io.Reader, fileType string) (string, error) {
	tikaURL := strings.TrimRight(s.cfg.TikaURL, "/") + "/tika"

	contentType := fileTypeToMIME(fileType)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, tikaURL, reader)
	if err != nil {
		return "", fmt.Errorf("create Tika request: %w", err)
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "text/plain")

	log.Debug().Str("url", tikaURL).Str("content_type", contentType).Msg("Sending file to Tika for extraction")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Tika request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("Tika returned status %d: %s", resp.StatusCode, string(body))
	}

	// Read extracted text (potentially very large — no size cap; rely on Tika's output)
	extracted, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read Tika response: %w", err)
	}

	text := normalizeWhitespace(string(extracted))
	log.Debug().Int("input_bytes", -1).Int("output_bytes", len(text)).Msg("Tika extraction complete")

	return text, nil
}

// Ping checks that the Tika server is reachable.
func (s *TikaService) Ping(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.cfg.TikaURL, nil)
	if err != nil {
		return err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Tika health check returned status %d", resp.StatusCode)
	}
	return nil
}

// fileTypeToMIME maps Arkheion file type strings to MIME types for Tika.
func fileTypeToMIME(fileType string) string {
	switch strings.ToLower(fileType) {
	case "pdf":
		return "application/pdf"
	case "epub":
		return "application/epub+zip"
	case "docx":
		return "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	case "doc":
		return "application/msword"
	case "txt":
		return "text/plain"
	case "rtf":
		return "application/rtf"
	case "html", "htm":
		return "text/html"
	default:
		return "application/octet-stream"
	}
}

// normalizeWhitespace collapses runs of whitespace in Tika output while
// preserving paragraph breaks (double newlines).
func normalizeWhitespace(s string) string {
	// Replace CRLF with LF
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	// Collapse 3+ consecutive newlines to double newline (paragraph break)
	for strings.Contains(s, "\n\n\n") {
		s = strings.ReplaceAll(s, "\n\n\n", "\n\n")
	}

	return strings.TrimSpace(s)
}
