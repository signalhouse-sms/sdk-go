package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// AuthService handles authentication-related API calls.
type AuthService struct {
	client *Client

	// Admin contains admin-only auth methods. Only populated when EnableAdmin is true.
	Admin *AuthAdminService
}

// AuthAdminService contains admin-only auth methods.
type AuthAdminService struct {
	client *Client
}

// ResetUserTwoFactor turns off two-factor authentication for a user who has lost every factor
// (lockout recovery), clearing their methods, backup codes and trusted devices. Admin only.
func (s *AuthAdminService) ResetUserTwoFactor(ctx context.Context, userID string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"userId": userID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", fmt.Sprintf("/auth/2fa/users/%s", url.PathEscape(userID)), nil, opts)
}

// LoginParams are the parameters for the Login method.
type LoginParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	// DeviceToken is a trusted-device token from an earlier VerifyTwoFactorChallenge; skips 2FA while valid.
	DeviceToken string `json:"deviceToken,omitempty"`
}

// Login authenticates a user with email and password. The response holds a token, or
// mfaRequired + challengeId when the account has 2FA on, or a token plus twoFactorSetupRequired
// (a setup-only session) when the user's group requires 2FA and they haven't turned it on.
func (s *AuthService) Login(ctx context.Context, params LoginParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/auth", params, opts)
}

// ResetPasswordParams are the parameters for the ResetPassword method.
type ResetPasswordParams struct {
	UserID      string `json:"-"`
	NewPassword string `json:"newPassword"`
}

// ResetPassword resets a user's password.
func (s *AuthService) ResetPassword(ctx context.Context, params ResetPasswordParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"userId": params.UserID, "newPassword": params.NewPassword}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/auth/resetpassword/%s", url.PathEscape(params.UserID))
	body := map[string]string{"newPassword": params.NewPassword}
	return s.client.doJSON(ctx, "PUT", path, body, opts)
}

// ForgotPasswordParams are the parameters for the ForgotPassword method.
type ForgotPasswordParams struct {
	Email string `json:"email"`
}

// ForgotPassword requests a password-reset link be emailed to the given address (public).
// Always succeeds regardless of whether an account exists for the email, so it cannot be
// used to enumerate registered emails.
func (s *AuthService) ForgotPassword(ctx context.Context, params ForgotPasswordParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"email": params.Email}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/forgot-password", params, opts)
}

// ResetPasswordWithTokenParams are the parameters for the ResetPasswordWithToken method.
type ResetPasswordWithTokenParams struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

// ResetPasswordWithToken resets a password using the single-use token from a reset email (public).
func (s *AuthService) ResetPasswordWithToken(ctx context.Context, params ResetPasswordWithTokenParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"token": params.Token, "password": params.Password}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/reset-password", params, opts)
}

// GetAuthHistoryParams are the parameters for the GetAuthHistory method.
type GetAuthHistoryParams struct {
	GroupID string
	UserID  string
	Page    string
	Limit   string
}

// GetAuthHistory returns login history for a group or user.
func (s *AuthService) GetAuthHistory(ctx context.Context, params GetAuthHistoryParams, opts *RequestOptions) (*Response, error) {
	if err := requireAny(map[string]string{"groupId": params.GroupID, "userId": params.UserID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId": params.GroupID,
		"userId":  params.UserID,
		"page":    params.Page,
		"limit":   params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/auth/history"+qs, nil, opts)
}

// LogoutAll logs out all other users in the caller's active group.
func (s *AuthService) LogoutAll(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/auth/logout-all", nil, opts)
}

// GetGroupID returns the Group ID associated with the caller's JWT (their active group).
func (s *AuthService) GetGroupID(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/auth/group-id", nil, opts)
}

// RequestExternalLinkTokenParams are the parameters for the RequestExternalLinkToken method.
type RequestExternalLinkTokenParams struct {
	Product string `json:"product"`
}

// RequestExternalLinkToken mints a single-use, short-lived external-link token for the
// authenticated caller. The token is handed to the GHL/Shopify backend so it can link the
// caller's existing V2 group to its tenant.
func (s *AuthService) RequestExternalLinkToken(ctx context.Context, params RequestExternalLinkTokenParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"product": params.Product}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/external-link-token", params, opts)
}

// GetOAuthProviders lists the OAuth sign-in providers enabled on this API (only providers with
// credentials configured).
func (s *AuthService) GetOAuthProviders(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/auth/oauth/providers", nil, opts)
}

// GetOAuthAuthorizationURL builds the URL that starts an OAuth sign-in for provider ("google",
// "microsoft" or "apple"). Navigate a browser to it; it is a redirect, not an API call. The API
// returns the user to the portal's /auth/callback with a single-use ticket.
//
// next is optional (pass "" for none): a same-site path (e.g. "/oauth/consent?request=abc") to
// return the user to after sign-in. The API hands it back on the /auth/callback redirect as
// `next`; anything that isn't a rooted path on the portal is dropped.
//
// To pass an email (Enterprise SSO) or a link ticket, use GetOAuthAuthorizationURLWithOptions.
func (s *AuthService) GetOAuthAuthorizationURL(provider string, next string) (string, error) {
	return s.GetOAuthAuthorizationURLWithOptions(provider, OAuthAuthorizationURLParams{Next: next})
}

// OAuthAuthorizationURLParams are the optional parameters for the GetOAuthAuthorizationURLWithOptions
// method. Empty fields are left out of the URL.
type OAuthAuthorizationURLParams struct {
	// Next is a same-site path (e.g. "/oauth/consent?request=abc") to return the user to after
	// sign-in. The API hands it back on the /auth/callback redirect as `next`; anything that isn't a
	// rooted path on the portal is dropped.
	Next string
	// Email is required for "sso" sign-in: the user's work email, whose domain picks the
	// organization's identity provider. Not needed with a ReauthTicket.
	Email string
	// LinkTicket (from CreateOAuthLink) links the provider to the signed-in account that requested
	// the ticket instead of signing in.
	LinkTicket string
	// ReauthTicket (from CreateOAuthReauth) signs the account that requested the ticket in again with
	// a linked provider, to confirm it's them.
	ReauthTicket string
}

// GetOAuthAuthorizationURLWithOptions builds the URL that starts an OAuth sign-in for provider
// ("google", "microsoft", "apple" or "sso" for Enterprise SSO, which requires Email). Navigate a
// browser to it; it is a redirect, not an API call. The API returns the user to the portal's
// /auth/callback with a single-use ticket. With a LinkTicket (from CreateOAuthLink) it links the
// provider to the signed-in account instead, and the API returns the user to /Settings/Security with
// `pendingLink=<ticket>` (pass it to ConfirmOAuthLink to finish linking) or `linkError`. With a
// ReauthTicket (from CreateOAuthReauth) the signed-in user signs in again to confirm it's them, and the
// API returns them to /Settings/Security with a single-use `pendingReauth` (the session that started it redeems it with ConfirmOAuthReauth) or `reauthError`.
func (s *AuthService) GetOAuthAuthorizationURLWithOptions(provider string, params OAuthAuthorizationURLParams) (string, error) {
	if err := require(map[string]string{"provider": provider}); err != nil {
		return "", err
	}
	authURL := fmt.Sprintf("%s/auth/oauth/%s", s.client.baseURL, url.PathEscape(provider))
	// url.Values encodes keys sorted, which yields email, linkTicket, next, reauthTicket.
	qs := buildQueryString(map[string]string{
		"email":        params.Email,
		"linkTicket":   params.LinkTicket,
		"next":         params.Next,
		"reauthTicket": params.ReauthTicket,
	})
	return authURL + qs, nil
}

// GetOAuthLinks lists the caller's sign-in methods: whether they chose their own password
// (passwordKnown is false for an account created through a provider or Enterprise SSO), and every
// provider linked to the account ({ passwordKnown, links: [{ provider, email, linkedAt, lastLoginAt }] }).
func (s *AuthService) GetOAuthLinks(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/auth/oauth/links", nil, opts)
}

// CreateOAuthLink starts linking provider ("google", "microsoft" or "apple") to the caller's account.
// Returns a single-use link ticket valid 2 minutes ({ linkTicket, expiresInSeconds }): navigate the
// browser to GetOAuthAuthorizationURLWithOptions(provider, OAuthAuthorizationURLParams{LinkTicket: ...}).
// Requires a fresh ConfirmTwoFactorIdentity from the same session, which this spends.
func (s *AuthService) CreateOAuthLink(ctx context.Context, provider string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"provider": provider}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", fmt.Sprintf("/auth/oauth/%s/link", url.PathEscape(provider)), nil, opts)
}

// CreateOAuthReauth starts signing in again with provider ("google", "microsoft", "apple" or "sso"),
// which must already be linked to the caller's account, to confirm it's them. It spends nothing.
// Returns a single-use ticket valid 2 minutes ({ reauthTicket, reauthSecret, expiresInSeconds }). Keep
// reauthSecret in the client that started the flow: it is returned only here, and ConfirmOAuthReauth
// needs it. Navigate the browser to GetOAuthAuthorizationURLWithOptions(provider, OAuthAuthorizationURLParams{ReauthTicket: ...}). The
// browser comes back to /Settings/Security with a single-use `pendingReauth` (the session that started it redeems it with ConfirmOAuthReauth) or `reauthError` (reauth_wrong_account,
// reauth_unsupported, reauth_stale, invalid_state, sso_not_configured, access_denied or provider_error).
func (s *AuthService) CreateOAuthReauth(ctx context.Context, provider string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"provider": provider}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", fmt.Sprintf("/auth/oauth/%s/reauth", url.PathEscape(provider)), nil, opts)
}

// ConfirmOAuthReauth finishes signing in again with a linked provider. The round trip started with
// CreateOAuthReauth returns the browser to /Settings/Security?pendingReauth=...; pass that single-use
// pendingReauth here (valid 5 minutes). It counts only for the same account and the same session that
// called CreateOAuthReauth, so a re-sign-in completed in someone else's browser confirms nothing. Fails
// with 400 when it expired or was used, and 403 for another account or session. Returns
// { identityConfirmed }: true when 2FA is off, so it counts as a confirmed identity. reauthSecret is the
// value CreateOAuthReauth returned; a wrong or missing secret is refused (403) and uses up the pending
// re-sign-in.
func (s *AuthService) ConfirmOAuthReauth(ctx context.Context, pendingReauth string, reauthSecret string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"pendingReauth": pendingReauth, "reauthSecret": reauthSecret}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/oauth/reauth/confirm", map[string]string{"pendingReauth": pendingReauth, "reauthSecret": reauthSecret}, opts)
}

// ConfirmOAuthLink finishes linking a provider. The provider round trip started with CreateOAuthLink
// returns the browser to /Settings/Security?pendingLink=…; pass that single-use pendingLink here. It
// links the provider, but only for the account the link was started for, so a link URL opened in
// someone else's browser can't attach their provider to your account. Returns the caller's
// { passwordKnown, links: [{ provider, email, linkedAt, lastLoginAt }] }.
func (s *AuthService) ConfirmOAuthLink(ctx context.Context, pendingLink string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"pendingLink": pendingLink}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/oauth/links/confirm", map[string]string{"pendingLink": pendingLink}, opts)
}

// DeleteOAuthLink unlinks provider ("google", "microsoft", "apple" or "sso") from the caller's account.
// Refused (400) when it is the account's only way to sign in. Requires a fresh ConfirmTwoFactorIdentity
// from the same session. Returns the remaining { passwordKnown, links }.
func (s *AuthService) DeleteOAuthLink(ctx context.Context, provider string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"provider": provider}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", fmt.Sprintf("/auth/oauth/%s/link", url.PathEscape(provider)), nil, opts)
}

// ReadOAuthTicketParams are the parameters for the ReadOAuthTicket method.
type ReadOAuthTicketParams struct {
	Ticket string `json:"ticket"`
}

// ReadOAuthTicket reads what a single-use OAuth ticket resolves to without consuming it: a
// "session" (ready to exchange), a "select" (pick one of several accounts) or a "signup"
// (finish registration).
func (s *AuthService) ReadOAuthTicket(ctx context.Context, params ReadOAuthTicketParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"ticket": params.Ticket}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/oauth/ticket", params, opts)
}

// ExchangeOAuthTicketParams are the parameters for the ExchangeOAuthTicket method.
type ExchangeOAuthTicketParams struct {
	Ticket string `json:"ticket"`
	// UserID is required for a "select" ticket: the chosen account from "accounts".
	UserID string `json:"userId,omitempty"`
	// DeviceToken is a trusted-device token from an earlier VerifyTwoFactorChallenge; skips 2FA while valid.
	DeviceToken string `json:"deviceToken,omitempty"`
}

// ExchangeOAuthTicket exchanges a single-use "session" or "select" OAuth ticket for a JWT.
// Consumes the ticket.
func (s *AuthService) ExchangeOAuthTicket(ctx context.Context, params ExchangeOAuthTicketParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"ticket": params.Ticket}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/oauth/exchange", params, opts)
}

// GetTwoFactorSettings returns the caller's two-factor settings: methods (targets masked), the
// default method, backup-code counts and trusted devices. Never returns secrets or codes.
func (s *AuthService) GetTwoFactorSettings(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/auth/2fa", nil, opts)
}

// ConfirmTwoFactorIdentityParams are the parameters for the ConfirmTwoFactorIdentity method.
type ConfirmTwoFactorIdentityParams struct {
	// Password is the account password. Required while 2FA is off.
	Password string `json:"password,omitempty"`
	// Code is a code from an enrolled method, or a backup code. Required while 2FA is on.
	Code string `json:"code,omitempty"`
	// Method is the method the code came from; defaults to the primary method.
	Method string `json:"method,omitempty"`
	// UsingBackupCode is true when Code is a backup code.
	UsingBackupCode bool `json:"usingBackupCode,omitempty"`
}

// ConfirmTwoFactorIdentity confirms it's really the account owner before a sensitive change:
// StartTwoFactorEnrollment, RemoveTwoFactorMethod, RegenerateBackupCodes or DisableTwoFactor; linking or
// unlinking a sign-in provider (CreateOAuthLink, DeleteOAuthLink); and saving or deleting a group's
// Enterprise SSO connection (Groups.SaveSsoConnection, Groups.DeleteSsoConnection). With 2FA on, pass a
// current Code (or a backup code); with 2FA off, pass the account Password, or, for an account without
// a password of its own (passwordKnown false in GetOAuthLinks), the Code from
// SendTwoFactorIdentityCode(ctx, "email", ...) with Method "email". A confirmation lasts 5 minutes,
// belongs to the session that made it, and unlocks exactly one change.
func (s *AuthService) ConfirmTwoFactorIdentity(ctx context.Context, params ConfirmTwoFactorIdentityParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/auth/2fa/confirm-identity", params, opts)
}

// SendTwoFactorIdentityCode sends a code by text or email (an enrolled "sms" or "email" method, or
// "email" for an account without a password of its own) to use with ConfirmTwoFactorIdentity. Not
// needed for "app". Returns { sent, target, expiresInSeconds? }.
func (s *AuthService) SendTwoFactorIdentityCode(ctx context.Context, method string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"method": method}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/2fa/confirm-identity/send", map[string]string{"method": method}, opts)
}

// StartTwoFactorEnrollmentParams are the parameters for the StartTwoFactorEnrollment method.
type StartTwoFactorEnrollmentParams struct {
	// Method is "app", "sms" or "email".
	Method string `json:"method"`
	// Phone is required for "sms".
	Phone string `json:"phone,omitempty"`
	// Address is required for "email".
	Address string `json:"address,omitempty"`
	// DeviceName optionally labels an authenticator app.
	DeviceName string `json:"deviceName,omitempty"`
}

// StartTwoFactorEnrollment starts enrolling a two-factor method. For "app" the response carries the
// setup secret and otpauth URI; for "sms" and "email" a code is sent.
// Requires a fresh ConfirmTwoFactorIdentity from the same session (403 otherwise); each confirmation unlocks one change.
func (s *AuthService) StartTwoFactorEnrollment(ctx context.Context, params StartTwoFactorEnrollmentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"method": params.Method}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/auth/2fa/enroll", params, opts)
}

// ConfirmTwoFactorEnrollmentParams are the parameters for the ConfirmTwoFactorEnrollment method.
type ConfirmTwoFactorEnrollmentParams struct {
	EnrollmentID string `json:"-"`
	Code         string `json:"code"`
}

// ConfirmTwoFactorEnrollment confirms a pending enrollment with its code. The first method turns 2FA
// on and returns backup codes, shown only this once.
func (s *AuthService) ConfirmTwoFactorEnrollment(ctx context.Context, params ConfirmTwoFactorEnrollmentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"enrollmentId": params.EnrollmentID, "code": params.Code}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/auth/2fa/enroll/%s/verify", url.PathEscape(params.EnrollmentID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// SetTwoFactorPrimaryMethod chooses which enrolled method ("app", "sms" or "email") is offered first at sign-in.
func (s *AuthService) SetTwoFactorPrimaryMethod(ctx context.Context, method string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"method": method}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", "/auth/2fa/primary", map[string]string{"method": method}, opts)
}

// RemoveTwoFactorMethod removes an enrolled two-factor method. Removing the last one turns 2FA off.
// Requires a fresh ConfirmTwoFactorIdentity from the same session (403 otherwise); each confirmation unlocks one change.
func (s *AuthService) RemoveTwoFactorMethod(ctx context.Context, method string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"method": method}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", fmt.Sprintf("/auth/2fa/methods/%s", url.PathEscape(method)), nil, opts)
}

// RegenerateBackupCodes replaces all backup codes with a new set, shown only once. The old codes stop working.
// Requires a fresh ConfirmTwoFactorIdentity from the same session (403 otherwise); each confirmation unlocks one change.
func (s *AuthService) RegenerateBackupCodes(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/auth/2fa/backup-codes", nil, opts)
}

// RevokeTrustedDevice stops trusting a device, so its next sign-in asks for the second factor again.
func (s *AuthService) RevokeTrustedDevice(ctx context.Context, deviceID string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"deviceId": deviceID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", fmt.Sprintf("/auth/2fa/trusted-devices/%s", url.PathEscape(deviceID)), nil, opts)
}

// DisableTwoFactor turns two-factor authentication off, removing every method, backup code and trusted device.
// Requires a fresh ConfirmTwoFactorIdentity from the same session (403 otherwise); each confirmation unlocks one change.
func (s *AuthService) DisableTwoFactor(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "DELETE", "/auth/2fa", nil, opts)
}

// GetTwoFactorPrompt reports whether the "secure your account" enrollment prompt is due.
func (s *AuthService) GetTwoFactorPrompt(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/auth/2fa/prompt", nil, opts)
}

// DismissTwoFactorPrompt dismisses the enrollment prompt until the next sign-in. It stops for good after three dismissals.
func (s *AuthService) DismissTwoFactorPrompt(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/auth/2fa/prompt/dismiss", nil, opts)
}

// SendTwoFactorChallengeCodeParams are the parameters for the SendTwoFactorChallengeCode method.
type SendTwoFactorChallengeCodeParams struct {
	ChallengeID string `json:"-"`
	// Method is "sms" or "email".
	Method string `json:"method"`
}

// SendTwoFactorChallengeCode sends a sign-in code by text or email for a pending challenge. Not needed for "app".
func (s *AuthService) SendTwoFactorChallengeCode(ctx context.Context, params SendTwoFactorChallengeCodeParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"challengeId": params.ChallengeID, "method": params.Method}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/auth/2fa/challenge/%s/send", url.PathEscape(params.ChallengeID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// VerifyTwoFactorChallengeParams are the parameters for the VerifyTwoFactorChallenge method.
type VerifyTwoFactorChallengeParams struct {
	ChallengeID string `json:"-"`
	// Code is the 6-digit code, or a backup code (xxxx-xxxx).
	Code string `json:"code"`
	// Method is the method the code came from; defaults to the primary method.
	Method string `json:"method,omitempty"`
	// UsingBackupCode is true when Code is a backup code.
	UsingBackupCode bool `json:"usingBackupCode,omitempty"`
	// RememberDevice trusts this device for 30 days; the response then carries a deviceToken.
	RememberDevice bool `json:"rememberDevice,omitempty"`
	// DeviceName labels the trusted device.
	DeviceName string `json:"deviceName,omitempty"`
}

// VerifyTwoFactorChallenge verifies a sign-in code (or backup code) for a pending challenge and
// returns the session token (plus a deviceToken when RememberDevice is set).
func (s *AuthService) VerifyTwoFactorChallenge(ctx context.Context, params VerifyTwoFactorChallengeParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"challengeId": params.ChallengeID, "code": params.Code}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/auth/2fa/challenge/%s/verify", url.PathEscape(params.ChallengeID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// CancelTwoFactorChallenge cancels a pending two-factor challenge.
func (s *AuthService) CancelTwoFactorChallenge(ctx context.Context, challengeID string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"challengeId": challengeID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", fmt.Sprintf("/auth/2fa/challenge/%s", url.PathEscape(challengeID)), nil, opts)
}
