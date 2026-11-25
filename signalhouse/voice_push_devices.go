package signalhouse

import (
	"context"
	"net/url"
)

// VoicePushDevicesService handles mobile push device API calls — a customer's
// mobile app registers each device so incoming calls wake it.
//
// These methods are called by the app with the user's voice token (the JWT
// from client.Voice.Tokens.CreateToken, minted with a SubgroupID) as the bearer
// token, not an API key: pass it as RequestOptions.Token on every call. The
// device is tied to that token's identity. Wraps the Voice service's
// /voice/api/v1/push-devices surface. Access via client.Voice.PushDevices.
type VoicePushDevicesService struct {
	client *Client
}

// RegisterPushDeviceParams is the body for RegisterPushDevice. Provider and
// DeviceToken are required.
type RegisterPushDeviceParams struct {
	// Provider is "apns" for iOS (a PushKit VoIP token) or "fcm" for Android
	// (a Firebase registration token).
	Provider string `json:"provider"`
	// DeviceToken is the device's push token.
	DeviceToken string `json:"deviceToken"`
	// Environment is "production" (the default) or "sandbox" for development
	// builds. apns only.
	Environment string `json:"environment,omitempty"`
	// AppID is the bundle ID or Firebase project ID. Only needed when the
	// account has several apps for the provider.
	AppID string `json:"appId,omitempty"`
}

// ListPushDeliveriesParams filters ListPushDeliveries. Limit is a string to
// match this SDK's query-param convention.
type ListPushDeliveriesParams struct {
	// Limit is how many deliveries to return (1-100) as a string. Defaults
	// server-side to "20".
	Limit string
}

// RegisterPushDevice registers this device for push. Call it after each voice
// token mint and whenever the OS gives the app a new push token. A push token
// already registered to another user moves to this one. opts.Token must carry
// the voice token.
// Returns { data: { id, identity, provider, environment, appId, registeredAt } } (201).
// POST /voice/api/v1/push-devices
func (s *VoicePushDevicesService) RegisterPushDevice(ctx context.Context, params RegisterPushDeviceParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"provider": params.Provider, "deviceToken": params.DeviceToken}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/push-devices", params, opts)
}

// UnregisterPushDevice unregisters this device, e.g. when the user signs out,
// so calls stop waking it. The device token is path-escaped (FCM tokens contain
// ":"). opts.Token must carry the voice token. Returns { data: { removed } }.
// DELETE /voice/api/v1/push-devices/:deviceToken
func (s *VoicePushDevicesService) UnregisterPushDevice(ctx context.Context, deviceToken string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"deviceToken": deviceToken}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", "/voice/api/v1/push-devices/"+url.PathEscape(deviceToken), nil, opts)
}

// ListPushDeliveries returns the pushes recently sent to the voice token's
// user, newest first (kept 7 days), for debugging an app. opts.Token must carry
// the voice token.
// Returns { data: [{ id, callUuid, provider, kind, status, error, createdAt }] }.
// GET /voice/api/v1/push-devices/deliveries
func (s *VoicePushDevicesService) ListPushDeliveries(ctx context.Context, params ListPushDeliveriesParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{"limit": params.Limit})
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/push-devices/deliveries"+qs, nil, opts)
}
