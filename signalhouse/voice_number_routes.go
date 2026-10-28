package signalhouse

import (
	"context"
	"net/url"
)

// VoiceNumberRoutesService handles per-number inbound route API calls.
//
// A number route is one inbound route set directly on a single phone number. A
// number's own route takes priority over its Programmable Voice Profile and
// subgroup voice settings.
type VoiceNumberRoutesService struct {
	client *Client
}

// SetNumberRouteParams is the body for SetNumberRoute. RouteAction is required.
// The whole route is replaced: fields the new RouteAction doesn't use are cleared.
type SetNumberRouteParams struct {
	RouteAction         string `json:"routeAction"`                   // "FORWARD" | "WEBRTC" | "SIP_TRUNK" | "SIP_PROFILE" | "CALL_CONTROL" | "AI" | "VOICEMAIL". Targets: FORWARD → ForwardToE164, SIP_TRUNK → RouteSipTrunkID, SIP_PROFILE → RouteSipProfileID, CALL_CONTROL → WebhookURL, AI → RouteAgentProfileID.
	ForwardToE164       string `json:"forwardToE164,omitempty"`       // FORWARD destination (required for FORWARD), or the WEBRTC no-answer destination when NoAnswerAction is "FORWARD".
	NoAnswerAction      string `json:"noAnswerAction,omitempty"`      // WEBRTC only: "VOICEMAIL" | "FORWARD" | "KEEP_RINGING" (default "VOICEMAIL").
	NoAnswerSeconds     *int   `json:"noAnswerSeconds,omitempty"`     // WEBRTC only: seconds to ring before the no-answer action, 5-120 (default 20).
	RouteSipTrunkID     string `json:"routeSipTrunkId,omitempty"`     // SIP trunk id, required when RouteAction is "SIP_TRUNK".
	RouteSipProfileID   string `json:"routeSipProfileId,omitempty"`   // SIP endpoint id, required when RouteAction is "SIP_PROFILE".
	WebhookURL          string `json:"webhookUrl,omitempty"`          // SHML program URL, required when RouteAction is "CALL_CONTROL".
	RouteAgentProfileID string `json:"routeAgentProfileId,omitempty"` // AI agent profile id (AGP…), required when RouteAction is "AI".
	RecordingEnabled    *bool  `json:"recordingEnabled,omitempty"`    // Record calls handled by this route.
}

// GetNumberRoute returns a number's own route and the routing currently in
// effect for it. A leading "+" on e164 is optional.
// Returns { e164, route | null, effective }.
// GET /voice/api/v1/numbers/:e164/route
func (s *VoiceNumberRoutesService) GetNumberRoute(ctx context.Context, e164 string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"e164": e164}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/numbers/"+url.PathEscape(e164)+"/route", nil, opts)
}

// SetNumberRoute sets (creates or replaces) a number's route. The whole route is
// replaced: fields the new RouteAction doesn't use are cleared. A leading "+" on
// e164 is optional. Returns { e164, route, effective }.
// PUT /voice/api/v1/numbers/:e164/route
func (s *VoiceNumberRoutesService) SetNumberRoute(ctx context.Context, e164 string, params SetNumberRouteParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"e164": e164, "routeAction": params.RouteAction}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", "/voice/api/v1/numbers/"+url.PathEscape(e164)+"/route", params, opts)
}

// DeleteNumberRoute removes a number's own route. The number falls back to its
// Programmable Voice Profile, subgroup voice settings, or the default. A leading
// "+" on e164 is optional. Returns { e164, route: null, effective }.
// DELETE /voice/api/v1/numbers/:e164/route
func (s *VoiceNumberRoutesService) DeleteNumberRoute(ctx context.Context, e164 string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"e164": e164}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", "/voice/api/v1/numbers/"+url.PathEscape(e164)+"/route", nil, opts)
}
