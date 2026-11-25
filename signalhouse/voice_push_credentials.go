package signalhouse

import (
	"context"
	"net/url"
)

// VoicePushCredentialsService handles mobile push credential API calls — the
// Apple APNs auth key and Firebase service account the Voice service uses to
// wake a customer's mobile app for incoming calls.
//
// One credential per app. Secrets are encrypted and never returned; responses
// carry a fingerprint instead. Each credential's metadata is { id, provider,
// name, appId, keyId, teamId, bundleId, projectId, clientEmail, fingerprint,
// createdAt, updatedAt }. Wraps the Voice service's
// /voice/api/v1/push-credentials surface. Access via
// client.Voice.PushCredentials.
type VoicePushCredentialsService struct {
	client *Client
}

// PushCredentialParams is the body for CreatePushCredential and
// UpdatePushCredential. Provider is required. An Apple ("apns") credential
// sets KeyID, TeamID, BundleID and PrivateKey; a Firebase ("fcm") credential
// sets ServiceAccount.
type PushCredentialParams struct {
	// Provider is "apns" (Apple Push Notification service, iOS) or "fcm"
	// (Firebase Cloud Messaging, Android). It cannot change on update.
	Provider string `json:"provider"`
	// Name is an optional label, up to 100 characters.
	Name string `json:"name,omitempty"`
	// KeyID is the 10-character key ID of the APNs auth key (.p8). apns only.
	KeyID string `json:"keyId,omitempty"`
	// TeamID is your 10-character Apple developer team ID. apns only.
	TeamID string `json:"teamId,omitempty"`
	// BundleID is your app's bundle ID; VoIP pushes go to its ".voip" topic.
	// apns only.
	BundleID string `json:"bundleId,omitempty"`
	// PrivateKey is the contents of the .p8 file. apns only.
	PrivateKey string `json:"privateKey,omitempty"`
	// ServiceAccount is the Firebase service account key file, as its JSON
	// text (a string) or the parsed object (e.g. a map[string]interface{} or
	// json.RawMessage). fcm only.
	ServiceAccount interface{} `json:"serviceAccount,omitempty"`
}

// ListPushCredentials returns the account's push credentials (metadata only).
// Returns { data: [...credentials] }.
// GET /voice/api/v1/push-credentials
func (s *VoicePushCredentialsService) ListPushCredentials(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/push-credentials", nil, opts)
}

// GetPushCredential returns one push credential (metadata only).
// Returns { data: { ...credential } }.
// GET /voice/api/v1/push-credentials/:id
func (s *VoicePushCredentialsService) GetPushCredential(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/push-credentials/"+url.PathEscape(id), nil, opts)
}

// CreatePushCredential uploads push credentials for one app: an Apple APNs
// auth key, or a Firebase service account. The key is checked on upload.
// Returns { data: { ...credential } } (201).
// POST /voice/api/v1/push-credentials
func (s *VoicePushCredentialsService) CreatePushCredential(ctx context.Context, params PushCredentialParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"provider": params.Provider}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/push-credentials", params, opts)
}

// UpdatePushCredential replaces a credential's key or service account (and
// name). Same body as CreatePushCredential; the provider cannot change.
// Registered devices keep working with the new key.
// Returns { data: { ...credential } }.
// PUT /voice/api/v1/push-credentials/:id
func (s *VoicePushCredentialsService) UpdatePushCredential(ctx context.Context, id string, params PushCredentialParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id, "provider": params.Provider}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", "/voice/api/v1/push-credentials/"+url.PathEscape(id), params, opts)
}

// DeletePushCredential deletes a credential and every device registered for
// that app; their calls stop being pushed. Returns { success }.
// DELETE /voice/api/v1/push-credentials/:id
func (s *VoicePushCredentialsService) DeletePushCredential(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", "/voice/api/v1/push-credentials/"+url.PathEscape(id), nil, opts)
}
