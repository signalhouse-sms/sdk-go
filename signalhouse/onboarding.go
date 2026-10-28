package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// OnboardingService handles onboarding-related API calls. All methods are
// staff-only and are exposed through the Admin sub-service only.
type OnboardingService struct {
	client      *Client
	enableAdmin bool

	// Admin contains admin-only onboarding methods. Only populated when EnableAdmin is true.
	Admin *OnboardingAdminService
}

// OnboardingAdminService contains admin-only onboarding methods.
type OnboardingAdminService struct {
	client *Client
}

// AdminGetOnboardingsParams are the parameters for the admin GetOnboardings method.
// Reserved for future pagination options; currently empty.
type AdminGetOnboardingsParams struct{}

// GetOnboardings returns all onboarding records (one per group). Admin only.
func (s *OnboardingAdminService) GetOnboardings(ctx context.Context, _ AdminGetOnboardingsParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/group/onboarding", nil, opts)
}

// AdminGetOnboardingParams are the parameters for the admin GetOnboarding method.
type AdminGetOnboardingParams struct {
	GroupID string
}

// GetOnboarding returns a single onboarding record by group ID. Admin only.
func (s *OnboardingAdminService) GetOnboarding(ctx context.Context, params AdminGetOnboardingParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/onboarding/%s", url.PathEscape(params.GroupID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// AddPricingCommentParams are the parameters for the admin AddPricingComment method.
type AddPricingCommentParams struct {
	GroupID string `json:"-"`
	Comment string `json:"comment"`
}

// AddPricingComment adds an internal comment/note to a group's admin Pricing editor. Admin only.
func (s *OnboardingAdminService) AddPricingComment(ctx context.Context, params AddPricingCommentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "comment": params.Comment}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/onboarding/%s/pricing-comments", url.PathEscape(params.GroupID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// AddCreditCommentParams are the parameters for the admin AddCreditComment method.
type AddCreditCommentParams struct {
	GroupID string `json:"-"`
	Comment string `json:"comment"`
}

// AddCreditComment adds an internal comment/note to a group's admin Credits page. Admin only.
func (s *OnboardingAdminService) AddCreditComment(ctx context.Context, params AddCreditCommentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "comment": params.Comment}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/onboarding/%s/credit-comments", url.PathEscape(params.GroupID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// SetTollFreeProviderParams are the parameters for the admin SetTollFreeProvider method.
type SetTollFreeProviderParams struct {
	GroupID          string `json:"-"`
	TollFreeProvider string `json:"tollFreeProvider"`
}

// SetTollFreeProvider sets where a group's new toll-free brands and numbers go ("infobip" or "twilio"). Admin only.
// Existing brands, numbers and campaigns stay where they are.
func (s *OnboardingAdminService) SetTollFreeProvider(ctx context.Context, params SetTollFreeProviderParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "tollFreeProvider": params.TollFreeProvider}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/onboarding/%s/toll-free-provider", url.PathEscape(params.GroupID))
	return s.client.doJSON(ctx, "PUT", path, params, opts)
}
