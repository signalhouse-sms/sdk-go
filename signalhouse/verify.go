package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// VerifyService handles the carrier phone-number checks under /verify: SIM Swap and Number Verify.
// Every check the carrier answers is billed to the group's wallet (402 on insufficient funds); a check
// the carrier cannot run for the number is a 422 and is not charged.
type VerifyService struct {
	client *Client
}

// CheckSimSwapParams are the parameters for the CheckSimSwap method.
type CheckSimSwapParams struct {
	// PhoneNumber is the number to check, in international format (e.g. "+12025550123"). Required.
	PhoneNumber string
	// MaxAge is the lookback window in hours, 1-2400. 0 omits it (the carrier's default).
	MaxAge int
	// IncludeSwapDate also returns latestSimChange (no extra charge). false omits it (the API default).
	IncludeSwapDate bool
}

// CheckSimSwap reports whether a number's SIM card changed recently, as
// {checkId, phoneNumber, swapped, maxAge, latestSimChange?, cost}. Billed once when the check answers;
// 422 when the check is not available for the number (not charged). Roles: api, admin, developer.
func (s *VerifyService) CheckSimSwap(ctx context.Context, params CheckSimSwapParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	body := map[string]any{"phoneNumber": params.PhoneNumber}
	if params.MaxAge != 0 {
		body["maxAge"] = params.MaxAge
	}
	if params.IncludeSwapDate {
		body["includeSwapDate"] = true
	}
	return s.client.doJSON(ctx, "POST", "/verify/sim-swap", body, opts)
}

// StartNumberVerificationParams are the parameters for the StartNumberVerification method.
type StartNumberVerificationParams struct {
	// PhoneNumber is the number to verify, in international format (e.g. "+12025550123"). Required.
	PhoneNumber string
	// DevicePhoneNumber is the phone's own number, sent to the carrier as a login hint. Empty omits it.
	DevicePhoneNumber string
	// RedirectURL is an https URL the end user's browser is sent to after the check, with ?checkId=…. Empty omits it.
	RedirectURL string
}

// StartNumberVerification starts a Number Verify check, returning
// {checkId, phoneNumber, status: "PENDING", authorizationUrl, expiresAt}. The end user's phone must open
// authorizationUrl on mobile data (Wi-Fi off) within 10 minutes; read the result with GetNumberVerification.
// Billed only when the check is answered (VERIFIED or NOT_VERIFIED). Roles: api, admin, developer.
func (s *VerifyService) StartNumberVerification(ctx context.Context, params StartNumberVerificationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	body := map[string]any{"phoneNumber": params.PhoneNumber}
	if params.DevicePhoneNumber != "" {
		body["devicePhoneNumber"] = params.DevicePhoneNumber
	}
	if params.RedirectURL != "" {
		body["redirectUrl"] = params.RedirectURL
	}
	return s.client.doJSON(ctx, "POST", "/verify/number", body, opts)
}

// GetNumberVerificationParams are the parameters for the GetNumberVerification method.
type GetNumberVerificationParams struct {
	// CheckID is the check ID (e.g. "NVC…"). Required.
	CheckID string
}

// GetNumberVerification returns a Number Verify check's status and result as
// {checkId, phoneNumber, status, devicePhoneNumberVerified, failureReason, createdAt, expiresAt, completedAt, cost}.
// A PENDING check past its expiresAt reads as EXPIRED. Roles: api, admin, developer, billing, user.
func (s *VerifyService) GetNumberVerification(ctx context.Context, params GetNumberVerificationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"checkId": params.CheckID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/verify/number/%s", url.PathEscape(params.CheckID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}
