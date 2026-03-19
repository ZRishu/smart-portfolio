package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/rs/zerolog/log"
)

// NotificationService defines the interface for sending notifications.
// This abstraction allows swapping Discord for Slack, email, etc. in the future.
type NotificationService interface {
	// SendContactNotification formats and sends a notification about a new
	// contact form submission. It is non-blocking — the actual HTTP call runs
	// in a background goroutine.
	SendContactNotification(ctx context.Context, senderName, senderEmail, messageBody string)

	// SendSponsorNotification formats and sends a notification about a new
	// sponsorship payment. It is non-blocking.
	SendSponsorNotification(ctx context.Context, sponsorName, email, currency string, amount float64)

	// SendRaw sends an arbitrary string message to the notification channel.
	// It is non-blocking.
	SendRaw(ctx context.Context, message string)

	// Shutdown waits for all in-flight notification goroutines to finish.
	// Call this during graceful application shutdown.
	Shutdown()
}

// discordPayload is the JSON body Discord's webhook API expects.
type discordPayload struct {
	Content string `json:"content"`
}

// DiscordNotificationService sends notifications via a Discord webhook URL.
// All sends are dispatched asynchronously in goroutines so the calling code
// is never blocked by network I/O.
type DiscordNotificationService struct {
	webhookURL string
	client     *http.Client
	wg         sync.WaitGroup
}

// NewDiscordNotificationService creates a new DiscordNotificationService.
// If the webhook URL is empty, all send operations become silent no-ops and
// a warning is logged at construction time.
func NewDiscordNotificationService(cfg config.DiscordConfig) *DiscordNotificationService {
	if cfg.WebhookURL == "" {
		log.Warn().Msg("discord: webhook URL is not configured — notifications will be silently skipped")
	} else {
		log.Info().Msg("discord: notification service initialized")
	}

	return &DiscordNotificationService{
		webhookURL: cfg.WebhookURL,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SendContactNotification formats a rich markdown message about a new contact
// form submission and sends it to Discord asynchronously.
func (d *DiscordNotificationService) SendContactNotification(ctx context.Context, senderName, senderEmail, messageBody string) {
	msg := fmt.Sprintf(
		"📬 **New Portfolio Contact Message!**\n"+
			"> **Name:** %s\n"+
			"> **Email:** %s\n"+
			"> **Message:**\n"+
			"```text\n%s\n```",
		senderName,
		senderEmail,
		messageBody,
	)

	d.sendAsync(ctx, msg)
}

// SendSponsorNotification formats a rich markdown message about a new
// sponsorship payment and sends it to Discord asynchronously.
func (d *DiscordNotificationService) SendSponsorNotification(ctx context.Context, sponsorName, email, currency string, amount float64) {
	msg := fmt.Sprintf(
		"🎉 **NEW SPONSOR ALERT!** 🎉\n"+
			"> **Name:** %s\n"+
			"> **Amount:** %.2f %s\n"+
			"> **Email:** %s\n"+
			"The outbox pipeline processed this payment successfully!",
		sponsorName,
		amount,
		currency,
		email,
	)

	d.sendAsync(ctx, msg)
}

// SendRaw sends an arbitrary string message to Discord asynchronously.
func (d *DiscordNotificationService) SendRaw(ctx context.Context, message string) {
	d.sendAsync(ctx, message)
}

// Shutdown blocks until every in-flight notification goroutine has completed.
// This prevents the process from exiting before all Discord webhook calls
// have finished.
func (d *DiscordNotificationService) Shutdown() {
	log.Info().Msg("discord: waiting for in-flight notifications to finish")
	d.wg.Wait()
	log.Info().Msg("discord: all notifications drained — shutdown complete")
}

// sendAsync dispatches the actual HTTP POST in a separate goroutine so the
// caller is never blocked. The goroutine is tracked via the WaitGroup so
// Shutdown can wait for it.
func (d *DiscordNotificationService) sendAsync(ctx context.Context, message string) {
	if d.webhookURL == "" {
		log.Debug().Msg("discord: skipping notification — webhook URL not configured")
		return
	}

	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		defer func() {
			if r := recover(); r != nil {
				log.Error().Interface("panic", r).Msg("discord: recovered from panic in notification goroutine")
			}
		}()

		if err := d.send(context.WithoutCancel(ctx), message); err != nil {
			log.Error().Err(err).Msg("discord: failed to send notification")
		}
	}()
}

// send performs the synchronous HTTP POST to the Discord webhook endpoint.
// It returns an error if the request fails or Discord responds with a
// non-2xx status code. It handles HTTP 429 (Too Many Requests) by retrying
// with exponential backoff and jitter, respecting the Retry-After header.
func (d *DiscordNotificationService) send(ctx context.Context, message string) error {
	payload := discordPayload{Content: message}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("discord: failed to marshal payload: %w", err)
	}

	// Retry configuration
	const maxRetries = 3
	initialBackoff := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
		if err != nil {
			return fmt.Errorf("discord: failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := d.client.Do(req)
		if err != nil {
			return fmt.Errorf("discord: request failed: %w", err)
		}
		defer resp.Body.Close()

		// Success!
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Info().Msg("discord: notification sent successfully")
			return nil
		}

		// Handle Rate Limiting (429)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfterStr := resp.Header.Get("Retry-After")
			var waitTime time.Duration

			if retryAfterStr != "" {
				// Discord usually sends Retry-After in milliseconds or seconds.
				// We try to parse it as seconds first.
				if seconds, err := strconv.Atoi(retryAfterStr); err == nil {
					waitTime = time.Duration(seconds) * time.Second
				} else {
					// If it's not a simple integer, it might be a date or 
					// we just fallback to our own backoff.
					waitTime = initialBackoff * time.Duration(1<<attempt)
				}
			} else {
				// Fallback to exponential backoff
				waitTime = initialBackoff * time.Duration(1<<attempt)
			}

			// Add jitter (±20%)
			jitter := time.Duration(float64(waitTime) * (0.8 + 0.4*rand.Float64()))

			log.Warn().
				Int("status", resp.StatusCode).
				Int("attempt", attempt+1).
				Dur("retry_after", jitter).
				Msg("discord: rate limited — retrying")

			select {
			case <-time.After(jitter):
				continue
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		// Other non-2xx status codes
		return fmt.Errorf("discord: unexpected status code %d", resp.StatusCode)
	}

	return fmt.Errorf("discord: failed after %d retries", maxRetries)
}
