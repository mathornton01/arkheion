package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/mathornton01/arkheion/config"
	"github.com/mathornton01/arkheion/models"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// WebhookService dispatches webhook events to registered endpoints.
// Dispatch runs in background goroutines and does not block the caller.
type WebhookService struct {
	cfg    *config.Config
	db     *pgxpool.Pool
	client *http.Client
	sem    chan struct{} // Limits concurrent webhook goroutines
}

// NewWebhookService creates a new WebhookService.
func NewWebhookService(cfg *config.Config, db *pgxpool.Pool) *WebhookService {
	return &WebhookService{
		cfg:  cfg,
		db:   db,
		sem:  make(chan struct{}, 10), // Max 10 concurrent webhook dispatches
		client: &http.Client{
			Timeout: time.Duration(cfg.WebhookTimeoutSeconds) * time.Second,
		},
	}
}

// Dispatch sends an event to all active webhooks that are subscribed to it.
// Runs asynchronously — each webhook URL is notified in its own goroutine.
func (s *WebhookService) Dispatch(event models.WebhookEvent, payload interface{}) {
	ctx := context.Background()

	// Fetch active webhooks subscribed to this event
	rows, err := s.db.Query(ctx, `
		SELECT id, url, secret
		FROM webhooks
		WHERE active = TRUE AND $1 = ANY(events)`,
		string(event))
	if err != nil {
		log.Error().Err(err).Str("event", string(event)).Msg("Webhook dispatch: failed to query webhooks")
		return
	}
	defer rows.Close()

	type webhookTarget struct {
		id     string
		url    string
		secret string
	}
	var targets []webhookTarget
	for rows.Next() {
		var t webhookTarget
		if err := rows.Scan(&t.id, &t.url, &t.secret); err != nil {
			log.Error().Err(err).Msg("Webhook dispatch: scan failed")
			continue
		}
		targets = append(targets, t)
	}

	if len(targets) == 0 {
		return
	}

	webhookPayload := models.WebhookPayload{
		Event:     event,
		Timestamp: time.Now().UTC(),
		Book:      payload,
	}

	body, err := json.Marshal(webhookPayload)
	if err != nil {
		log.Error().Err(err).Msg("Webhook dispatch: JSON marshal failed")
		return
	}

	for _, target := range targets {
		t := target
		s.sem <- struct{}{}
		go func() {
			defer func() { <-s.sem }()
			s.deliverWithRetry(t.id, t.url, t.secret, string(event), body)
		}()
	}
}

// deliverWithRetry attempts to deliver a webhook with exponential backoff retries.
func (s *WebhookService) deliverWithRetry(webhookID, url, secret, event string, body []byte) {
	maxRetries := s.cfg.WebhookMaxRetries
	delay := time.Duration(s.cfg.WebhookRetryInitialDelaySecs) * time.Second

	for attempt := 1; attempt <= maxRetries+1; attempt++ {
		statusCode, responseBody, err := s.deliver(url, secret, event, body)

		succeeded := err == nil && statusCode >= 200 && statusCode < 300

		// Log delivery attempt
		s.logDelivery(webhookID, event, body, statusCode, responseBody, attempt, succeeded)

		if succeeded {
			log.Info().
				Str("webhook_id", webhookID).
				Str("url", url).
				Str("event", event).
				Int("status", statusCode).
				Int("attempt", attempt).
				Msg("Webhook delivered successfully")
			return
		}

		if attempt <= maxRetries {
			log.Warn().
				Str("webhook_id", webhookID).
				Str("url", url).
				Str("event", event).
				Int("attempt", attempt).
				Dur("retry_in", delay).
				Msg("Webhook delivery failed, retrying")
			time.Sleep(delay)
			delay *= 2
		}
	}

	log.Error().
		Str("webhook_id", webhookID).
		Str("url", url).
		Str("event", event).
		Int("max_retries", maxRetries).
		Msg("Webhook delivery failed after all retries")
}

// deliver makes a single HTTP POST to a webhook URL.
func (s *WebhookService) deliver(url, secret, event string, body []byte) (int, string, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, "", fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Arkheion-Event", event)
	req.Header.Set("X-Arkheion-Signature", computeHMAC(body, secret))
	req.Header.Set("User-Agent", "Arkheion-Webhook/1.0")

	resp, err := s.client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)

	return resp.StatusCode, buf.String(), nil
}

// computeHMAC signs the body with HMAC-SHA256 using the webhook secret.
// The signature format matches the GitHub webhook signature convention:
//   sha256=<hex_digest>
func computeHMAC(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// logDelivery records a webhook delivery attempt in the database.
func (s *WebhookService) logDelivery(webhookID, event string, payload []byte, statusCode int, responseBody string, attempts int, succeeded bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var payloadJSON interface{}
	_ = json.Unmarshal(payload, &payloadJSON)

	_, err := s.db.Exec(ctx, `
		INSERT INTO webhook_deliveries
		    (webhook_id, event, payload, response_code, response_body, attempts, succeeded, last_attempt)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT DO NOTHING`,
		webhookID, event, payloadJSON, statusCode, responseBody, attempts, succeeded,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to log webhook delivery")
	}
}
