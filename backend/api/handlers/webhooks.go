package handlers

import (
	"context"
	"fmt"
	"strings"

	"github.com/mathornton01/arkheion/models"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// WebhooksHandler handles webhook registration and management.
type WebhooksHandler struct {
	db *pgxpool.Pool
}

// NewWebhooksHandler creates a new WebhooksHandler.
func NewWebhooksHandler(db *pgxpool.Pool) *WebhooksHandler {
	return &WebhooksHandler{db: db}
}

// ListWebhooks handles GET /api/v1/webhooks
func (h *WebhooksHandler) ListWebhooks(c *fiber.Ctx) error {
	ctx := context.Background()

	rows, err := h.db.Query(ctx, `
		SELECT id, url, events, active, description, created_at, updated_at
		FROM webhooks
		ORDER BY created_at DESC`)
	if err != nil {
		log.Error().Err(err).Msg("ListWebhooks: query failed")
		return fiber.ErrInternalServerError
	}
	defer rows.Close()

	webhooks := make([]models.Webhook, 0)
	for rows.Next() {
		var wh models.Webhook
		var events []string
		if err := rows.Scan(&wh.ID, &wh.URL, &events, &wh.Active, &wh.Description, &wh.CreatedAt, &wh.UpdatedAt); err != nil {
			return fiber.ErrInternalServerError
		}
		wh.Events = stringsToEvents(events)
		// Secret is intentionally omitted from list responses
		webhooks = append(webhooks, wh)
	}

	return c.JSON(fiber.Map{"data": webhooks})
}

// GetWebhook handles GET /api/v1/webhooks/:id
func (h *WebhooksHandler) GetWebhook(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Webhook ID must be a valid UUID", fiber.StatusBadRequest))
	}

	wh, err := h.getWebhookByID(ctx, id)
	if err != nil {
		if err == pgx.ErrNoRows {
			return c.Status(fiber.StatusNotFound).JSON(apiError("WEBHOOK_NOT_FOUND",
				fmt.Sprintf("No webhook found with id: %s", id), fiber.StatusNotFound))
		}
		return fiber.ErrInternalServerError
	}
	// Omit secret from response
	wh.Secret = ""

	return c.JSON(wh)
}

// CreateWebhook handles POST /api/v1/webhooks
func (h *WebhooksHandler) CreateWebhook(c *fiber.Ctx) error {
	ctx := context.Background()

	var req models.CreateWebhookRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_BODY", "Invalid JSON body", fiber.StatusBadRequest))
	}

	// Validate
	if req.URL == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(apiError("VALIDATION_ERROR", "url is required", fiber.StatusUnprocessableEntity))
	}
	if len(req.Secret) < 16 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(apiError("VALIDATION_ERROR", "secret must be at least 16 characters", fiber.StatusUnprocessableEntity))
	}
	if len(req.Events) == 0 {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(apiError("VALIDATION_ERROR", "at least one event is required", fiber.StatusUnprocessableEntity))
	}

	// Validate event names
	validEvents := map[models.WebhookEvent]bool{}
	for _, e := range models.AllWebhookEvents {
		validEvents[e] = true
	}
	for _, e := range req.Events {
		if !validEvents[e] {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(apiError("VALIDATION_ERROR",
				fmt.Sprintf("Invalid event: %s. Valid events: %s", e, eventsToString(models.AllWebhookEvents)),
				fiber.StatusUnprocessableEntity))
		}
	}

	eventStrings := make([]string, len(req.Events))
	for i, e := range req.Events {
		eventStrings[i] = string(e)
	}

	var id uuid.UUID
	err := h.db.QueryRow(ctx, `
		INSERT INTO webhooks (url, secret, events, description)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		req.URL, req.Secret, eventStrings, req.Description,
	).Scan(&id)
	if err != nil {
		log.Error().Err(err).Msg("CreateWebhook: insert failed")
		return fiber.ErrInternalServerError
	}

	wh, err := h.getWebhookByID(ctx, id)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	wh.Secret = "" // Don't echo secret back

	return c.Status(fiber.StatusCreated).JSON(wh)
}

// DeleteWebhook handles DELETE /api/v1/webhooks/:id
func (h *WebhooksHandler) DeleteWebhook(c *fiber.Ctx) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Webhook ID must be a valid UUID", fiber.StatusBadRequest))
	}

	result, err := h.db.Exec(ctx, "DELETE FROM webhooks WHERE id = $1", id)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(apiError("WEBHOOK_NOT_FOUND",
			fmt.Sprintf("No webhook found with id: %s", id), fiber.StatusNotFound))
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// ActivateWebhook handles PUT /api/v1/webhooks/:id/activate
func (h *WebhooksHandler) ActivateWebhook(c *fiber.Ctx) error {
	return h.setWebhookActive(c, true)
}

// DeactivateWebhook handles PUT /api/v1/webhooks/:id/deactivate
func (h *WebhooksHandler) DeactivateWebhook(c *fiber.Ctx) error {
	return h.setWebhookActive(c, false)
}

func (h *WebhooksHandler) setWebhookActive(c *fiber.Ctx, active bool) error {
	ctx := context.Background()
	id, err := parseUUID(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(apiError("INVALID_ID", "Webhook ID must be a valid UUID", fiber.StatusBadRequest))
	}

	result, err := h.db.Exec(ctx, "UPDATE webhooks SET active = $1, updated_at = NOW() WHERE id = $2", active, id)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	if result.RowsAffected() == 0 {
		return c.Status(fiber.StatusNotFound).JSON(apiError("WEBHOOK_NOT_FOUND",
			fmt.Sprintf("No webhook found with id: %s", id), fiber.StatusNotFound))
	}

	wh, err := h.getWebhookByID(ctx, id)
	if err != nil {
		return fiber.ErrInternalServerError
	}
	wh.Secret = ""
	return c.JSON(wh)
}

func (h *WebhooksHandler) getWebhookByID(ctx context.Context, id uuid.UUID) (*models.Webhook, error) {
	var wh models.Webhook
	var events []string
	err := h.db.QueryRow(ctx, `
		SELECT id, url, secret, events, active, description, created_at, updated_at
		FROM webhooks WHERE id = $1`, id).
		Scan(&wh.ID, &wh.URL, &wh.Secret, &events, &wh.Active, &wh.Description, &wh.CreatedAt, &wh.UpdatedAt)
	if err != nil {
		return nil, err
	}
	wh.Events = stringsToEvents(events)
	return &wh, nil
}

func stringsToEvents(ss []string) []models.WebhookEvent {
	events := make([]models.WebhookEvent, len(ss))
	for i, s := range ss {
		events[i] = models.WebhookEvent(s)
	}
	return events
}

func eventsToString(events []models.WebhookEvent) string {
	ss := make([]string, len(events))
	for i, e := range events {
		ss[i] = string(e)
	}
	return strings.Join(ss, ", ")
}
