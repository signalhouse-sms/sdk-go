package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// ShortlinksService handles shortlink-related API calls.
type ShortlinksService struct {
	client *Client
}

// GetShortlinkRedirectParams are the parameters for the GetShortlinkRedirect method.
type GetShortlinkRedirectParams struct {
	ShortlinkID string
}

// GetShortlinkRedirect returns the redirect URL for a shortlink.
func (s *ShortlinksService) GetShortlinkRedirect(ctx context.Context, params GetShortlinkRedirectParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"shortlinkId": params.ShortlinkID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/shortlink/redirect/%s", url.PathEscape(params.ShortlinkID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// OptOutParams are the parameters for the GetOptOut and ConfirmOptOut methods.
type OptOutParams struct {
	Code string
}

// GetOptOut returns the recipient opt-out details ({code, senderName, maskedRecipient, optedOut})
// behind an alphanumeric-sender opt-out link (sihou.io/o/{code}). Public; no auth required.
func (s *ShortlinksService) GetOptOut(ctx context.Context, params OptOutParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"code": params.Code}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/shortlink/optout/%s", url.PathEscape(params.Code))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// ConfirmOptOut opts the recipient behind an alphanumeric-sender opt-out link out; idempotent.
// Public; no auth required.
func (s *ShortlinksService) ConfirmOptOut(ctx context.Context, params OptOutParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"code": params.Code}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/shortlink/optout/%s", url.PathEscape(params.Code))
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// GetShortlinkParams are the parameters for the GetShortlink method.
type GetShortlinkParams struct {
	ShortlinkID string
	MessageID   string
	PhoneNumber string
	CampaignID  string
	BrandID     string
	SubgroupID  string
	GroupID     string
	Page        string
	Limit       string
}

// GetShortlink returns shortlink details with optional filters.
func (s *ShortlinksService) GetShortlink(ctx context.Context, params GetShortlinkParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"shortlinkId": params.ShortlinkID,
		"messageId":   params.MessageID,
		"phoneNumber": params.PhoneNumber,
		"campaignId":  params.CampaignID,
		"brandId":     params.BrandID,
		"subgroupId":  params.SubgroupID,
		"groupId":     params.GroupID,
		"page":        params.Page,
		"limit":       params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/shortlink"+qs, nil, opts)
}
