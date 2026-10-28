package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// WebhooksService handles webhook-related API calls.
type WebhooksService struct {
	client *Client
}

// GetWebhooksParams are the parameters for the GetWebhooks method.
type GetWebhooksParams struct {
	ID           string
	GroupID      string
	EndpointType string
	PhoneNumber  string
	Page         string
	Limit        string
}

// GetWebhooks returns a list of webhooks with optional filters. Each returned endpoint carries
// "hasSigningSecret" (bool) instead of the signing secret itself — see CreateWebhook.
func (s *WebhooksService) GetWebhooks(ctx context.Context, params GetWebhooksParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"id":           params.ID,
		"groupId":      params.GroupID,
		"endpointType": params.EndpointType,
		"phoneNumber":  params.PhoneNumber,
		"page":         params.Page,
		"limit":        params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/webhook"+qs, nil, opts)
}

// CreateWebhookParams are the parameters for the CreateWebhook method.
type CreateWebhookParams struct {
	WebhookData map[string]interface{}
}

// CreateWebhook creates a new webhook. The response includes a "signingSecret" (plaintext
// HMAC-SHA256 secret) used to sign every delivery to this endpoint via the X-SignalHouse-Signature
// header. It is returned exactly once, here — store it now, it is never exposed again by any read.
//
// WebhookData "authType" is one of "NONE" (default), "BASIC", "API_KEY", "OAUTH2" or "HMAC". "HMAC"
// sends no credentials (the receiver authenticates by the signature alone) and must be sent without
// "credentials" or "apiHeaderPrefix".
//
// WebhookData "eventSettings" holds per-event options keyed by event name: "NUMBER_HEALTH_BELOW_THRESHOLD"
// and "CAMPAIGN_HEALTH_BELOW_THRESHOLD" each take {"threshold": n}, the 7-day health score (1.5–10, in 0.5
// steps) below which the event fires. A subscribed health event without a threshold is stored with the
// default, 6.
func (s *WebhooksService) CreateWebhook(ctx context.Context, params CreateWebhookParams, opts *RequestOptions) (*Response, error) {
	if err := requireNonNil(map[string]interface{}{"webhookData": params.WebhookData}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/webhook", params.WebhookData, opts)
}

// UpdateWebhookParams are the parameters for the UpdateWebhook method.
type UpdateWebhookParams struct {
	ID         string
	UpdateData map[string]interface{}
}

// UpdateWebhook updates an existing webhook. Like every other read of a webhook, the response
// carries "hasSigningSecret" (bool), never the signing secret itself — see CreateWebhook.
//
// Switching "authType" to "NONE" or "HMAC" clears the stored credentials. "HMAC" is rejected for an
// endpoint with no signing secret; call RotateWebhookSecret first.
//
// UpdateData "eventSettings" changes per-event options by event name ("NUMBER_HEALTH_BELOW_THRESHOLD",
// "CAMPAIGN_HEALTH_BELOW_THRESHOLD", each {"threshold": n}, 1.5–10 in 0.5 steps); events you omit keep
// their stored settings.
func (s *WebhooksService) UpdateWebhook(ctx context.Context, params UpdateWebhookParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	if err := requireNonNil(map[string]interface{}{"updateData": params.UpdateData}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/webhook/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// RotateWebhookSecretParams are the parameters for the RotateWebhookSecret method.
type RotateWebhookSecretParams struct {
	ID string
	// GraceSeconds is how long the replaced secret keeps working (default 86400, max 604800).
	// nil uses the default; a pointer to 0 stops the old secret immediately.
	GraceSeconds *int
}

// RotateWebhookSecret rotates a webhook's signing secret, or gives a webhook created before signing
// its first one. The response includes the new plaintext signingSecret, returned this one time only.
// During the grace window each delivery's X-SignalHouse-Signature-V1 header carries a signature for
// both secrets.
func (s *WebhooksService) RotateWebhookSecret(ctx context.Context, params RotateWebhookSecretParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	body := map[string]interface{}{}
	if params.GraceSeconds != nil {
		body["graceSeconds"] = *params.GraceSeconds
	}
	path := fmt.Sprintf("/webhook/%s/rotate-secret", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// DeleteWebhookParams are the parameters for the DeleteWebhook method.
type DeleteWebhookParams struct {
	ID string
}

// DeleteWebhook deletes (marks as inactive) a webhook.
func (s *WebhooksService) DeleteWebhook(ctx context.Context, params DeleteWebhookParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/webhook/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}
