package signalhouse

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
)

// RcsService groups RCS (Google RBM) resources. Access via client.Rcs.Agents. Templates for RCS live under
// client.Templates.
type RcsService struct {
	client *Client

	// Agents — RCS agents under /rcs/agent.
	Agents *RcsAgentsService
}

// RcsAgentsService handles RCS agents under /rcs/agent. An agent links one 10DLC brand and one 10DLC campaign: the
// campaign's numbers are the SMS fallback, and the brand and campaign supply the company identity and consent text
// for the carrier launch. One brand or campaign can serve many agents. An agent is a registration (type RCS_AGENT,
// id RCSA…), so the registrations methods read and edit the same record. 404 while the environment's RCS switch is
// off; 400 while the group's rcsEnabled flag is off.
type RcsAgentsService struct {
	client *Client
}

// RcsAgentProfile is the agent profile shown on the handset. Empty fields are omitted. To clear a field on Update,
// pass a map[string]any with that key set to nil instead.
type RcsAgentProfile struct {
	// Name is shown on the handset (max 40).
	Name string `json:"name,omitempty"`
	// Description is at most 100 characters.
	Description string `json:"description,omitempty"`
	// Color is a hex brand color (default #6868ab).
	Color string `json:"color,omitempty"`
	// LogoURL is an HTTPS URL of a 224 × 224 px logo (UploadAsset hosts one).
	LogoURL string `json:"logoUrl,omitempty"`
	// HeroURL is an HTTPS URL of a 1440 × 448 px hero image.
	HeroURL string `json:"heroUrl,omitempty"`
	// Phone is the contact phone in E.164 (e.g. +18005550142).
	Phone string `json:"phone,omitempty"`
	// Website is an HTTPS website.
	Website string `json:"website,omitempty"`
	Email   string `json:"email,omitempty"`
	// PhoneLabel captions the Call action (max 40; submit defaults it to "Call").
	PhoneLabel string `json:"phoneLabel,omitempty"`
	// WebsiteLabel captions the Website action (max 40; submit defaults it to "Website").
	WebsiteLabel string `json:"websiteLabel,omitempty"`
	// EmailLabel captions the Email action (max 40; submit defaults it to "Email").
	EmailLabel string `json:"emailLabel,omitempty"`
	// PrivacyPolicyURL is an HTTPS privacy policy URL.
	PrivacyPolicyURL string `json:"privacyPolicyUrl,omitempty"`
	// TermsURL is an HTTPS terms of service URL.
	TermsURL string `json:"termsUrl,omitempty"`
}

// RcsAgentConsent is the agent's launch text. Every field is required before submit, the opt-in URL and the launch
// video included. Blank fields fill from the linked campaign where noted. Empty fields are omitted. To clear a field
// on Update, pass a map[string]any with that key set to nil instead.
type RcsAgentConsent struct {
	// OptInDetails says how recipients opt in (filled from the campaign's messageFlow when blank).
	OptInDetails string `json:"optInDetails,omitempty"`
	// OptInURL is an HTTPS page that shows the opt-in.
	OptInURL string `json:"optInUrl,omitempty"`
	// OptInResponse is the opt-in confirmation (filled from the campaign's optinMessage).
	OptInResponse string `json:"optInResponse,omitempty"`
	// OptOutDetails says how recipients opt out.
	OptOutDetails string `json:"optOutDetails,omitempty"`
	// OptOutResponse is the opt-out confirmation (filled from the campaign's optoutMessage).
	OptOutResponse string `json:"optOutResponse,omitempty"`
	// HelpResponse is the HELP reply (filled from the campaign's helpMessage).
	HelpResponse string `json:"helpResponse,omitempty"`
	// CampaignDescription says what the agent sends (filled from the campaign's description).
	CampaignDescription string `json:"campaignDescription,omitempty"`
	// MessageExamples holds 1-6 example messages (filled from the campaign's samples).
	MessageExamples []string `json:"messageExamples,omitempty"`
	// EstimatedMonthlyVolume is UP_TO_10K, 10K_100K, 100K_500K, 500K_1M or 1M_PLUS.
	EstimatedMonthlyVolume string `json:"estimatedMonthlyVolume,omitempty"`
	// LaunchVideoURL is an HTTPS video of the opt-in flow.
	LaunchVideoURL string `json:"launchVideoUrl,omitempty"`
}

// RcsAgentFallback is the agent's default fallback. Empty fields take the server defaults.
type RcsAgentFallback struct {
	// Mode is auto (default), rcs_only, sms_only or none.
	Mode string `json:"mode,omitempty"`
	// Channel is auto (default), sms or mms.
	Channel string `json:"channel,omitempty"`
	// From is a number on the linked campaign.
	From string `json:"from,omitempty"`
}

// CreateRcsAgentParams are the parameters for the Create method.
type CreateRcsAgentParams struct {
	// BrandID is the 10DLC brand: carrier brand ID, provisional UUID or _id. Required.
	BrandID string
	// CampaignID is a 10DLC campaign of that brand (campaign ID or _id); required before submit. Empty omits it.
	CampaignID string
	// Agent is the profile: an RcsAgentProfile, *RcsAgentProfile or map[string]any. nil omits it.
	Agent any
	// Consent is the launch text: an RcsAgentConsent, *RcsAgentConsent or map[string]any. nil omits it. Required in full
	// before the launch request.
	Consent any
	// Contact is the launch contact { firstName, lastName, role, email, mobile }: the individual at the brand the carrier
	// partner's reviewers may write to, as a map[string]any. nil omits it. Required in full before the launch request.
	Contact any
	// UseCase is OTP, TRANSACTIONAL, PROMOTIONAL or MULTI_USE; required before submit.
	UseCase string
	// BillingCategory is NON_CONVERSATIONAL (default) or CONVERSATIONAL; submit refuses CONVERSATIONAL.
	BillingCategory string
	// DefaultFallback is an RcsAgentFallback, *RcsAgentFallback or map[string]any. nil omits it.
	DefaultFallback any
	// ExcludedCarriers is [] or ["T_MOBILE"] (the carrier partner's carrier name); nil omits it.
	ExcludedCarriers []string
	// Testers are tester numbers (up to 200), invited once the agent exists at the carrier; nil omits them.
	Testers []string
}

// Create creates an agent draft on a 10DLC brand, optionally with its campaign, and returns the agent (HTTP 201,
// status DRAFT). The agent's subgroup is the brand's. Blank consent fields fill from the linked campaign. Set
// RequestOptions.IdempotencyKey to make a retry safe.
func (s *RcsAgentsService) Create(ctx context.Context, params CreateRcsAgentParams, opts *RequestOptions) (*Response, error) {
	body, err := rcsAgentCreateBody(params)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/rcs/agent", body, opts)
}

// ListRcsAgentsParams are the parameters for the List method.
type ListRcsAgentsParams struct {
	GroupID    string
	SubgroupID string
	// Status is DRAFT, PENDING_REVIEW, REJECTED, AGENT_CREATING or ARCHIVED, one value or a comma-separated list
	// (e.g. "DRAFT,REJECTED").
	Status string
	// BrandID returns agents linked to this 10DLC brand (carrier brand ID, provisional UUID or _id).
	BrandID string
	// CampaignID returns agents linked to this 10DLC campaign (campaign ID or _id).
	CampaignID string
	// Page defaults to 1.
	Page string
	// Limit defaults to 25, max 100.
	Limit string
}

// List lists agents, newest first, as {data, total, page, limit}.
func (s *RcsAgentsService) List(ctx context.Context, params ListRcsAgentsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":    params.GroupID,
		"subgroupId": params.SubgroupID,
		"status":     params.Status,
		"brandId":    params.BrandID,
		"campaignId": params.CampaignID,
		"page":       params.Page,
		"limit":      params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/rcs/agent"+qs, nil, opts)
}

// RcsAgentParams identify one agent for the Get, Validate, Submit and Archive methods.
type RcsAgentParams struct {
	// AgentID is the agent ID (e.g. RCSA7K2M1QZ8RP). Required.
	AgentID string
}

// Get returns one agent with the live state of its brand and campaign links.
func (s *RcsAgentsService) Get(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// UpdateRcsAgentParams are the parameters for the Update method. Unset fields are omitted and keep their value.
type UpdateRcsAgentParams struct {
	// AgentID is the agent ID. Required.
	AgentID string
	// BrandID relinks the agent to another 10DLC brand of the same subgroup.
	BrandID string
	// CampaignID links or relinks the campaign.
	CampaignID string
	// ClearCampaign sends campaignId: null to unlink the campaign; it overrides CampaignID.
	ClearCampaign bool
	// Agent holds the profile fields to change (merged field by field): an RcsAgentProfile or map[string]any.
	Agent any
	// Consent holds the consent fields to change (merged field by field): an RcsAgentConsent or map[string]any.
	Consent any
	// Contact holds the launch contact fields to change (merged field by field), as a map[string]any.
	Contact any
	// UseCase sets the use case.
	UseCase string
	// ClearUseCase sends useCase: null; it overrides UseCase.
	ClearUseCase bool
	// BillingCategory sets the billing category.
	BillingCategory string
	// DefaultFallback replaces the fallback: an RcsAgentFallback or map[string]any.
	DefaultFallback any
	// ClearDefaultFallback sends defaultFallback: null; it overrides DefaultFallback.
	ClearDefaultFallback bool
	// ExcludedCarriers replaces the list ([] or ["T_MOBILE"]); nil keeps it, an empty non-nil slice clears it.
	ExcludedCarriers []string
	// Testers replaces the tester numbers; nil keeps them, an empty non-nil slice clears them.
	Testers []string
}

// Update edits an agent while it may change (a draft, a Signal House rejection, a request for changes, or an agent in
// testing, whose change is pushed to the carrier partner) and returns it. Agent, Consent and Contact merge field by
// field; every other key replaces. 409 in any other status; BrandID is locked once the agent exists at the partner and
// BillingCategory once a launch was requested.
func (s *RcsAgentsService) Update(ctx context.Context, params UpdateRcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", path, rcsAgentUpdateBody(params), opts)
}

// Validate checks an agent against every submit requirement without writing and returns {valid, errors:
// [{field, reason}], warnings: [{field, reason}]}.
func (s *RcsAgentsService) Validate(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/validate")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// Submit submits a draft: creates the agent at the carrier partner, and returns it (status
// AGENT_CREATING, then IN_TESTING once the partner reports it ready). It needs the full profile, a use case and
// NON_CONVERSATIONAL billing; otherwise a 400 whose details carry {code: "REGISTRATION_INCOMPLETE", errors, warnings}.
func (s *RcsAgentsService) Submit(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/submit")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// Launch requests the agent's launch: the launch requirements are checked and the agent goes to Signal House review
// (status PENDING_REVIEW), whose approval files the launch with the carrier partner. Allowed from IN_TESTING, REJECTED
// and CARRIER_REJECTED (409 otherwise); a failed requirement is a 400 whose details carry {code:
// "REGISTRATION_INCOMPLETE", errors, warnings}. It sends POST /registration/:agentId/launch: the agent is a registration.
func (s *RcsAgentsService) Launch(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", fmt.Sprintf("/registration/%s/launch", url.PathEscape(params.AgentID)), nil, opts)
}

// Archive archives an agent and returns it (status ARCHIVED).
func (s *RcsAgentsService) Archive(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/archive")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// ResendRcsTesterInviteParams are the parameters for the ResendTesterInvite method.
type ResendRcsTesterInviteParams struct {
	// AgentID is the agent ID. Required.
	AgentID string
	// PhoneNumber is the tester's US number, with or without the +. Required.
	PhoneNumber string
}

// ResendTesterInvite resends a tester's expired invitation, which starts a new 24 hours for the tester to accept it,
// and returns the agent. The number must be a tester on the agent (404) and its invitation must be EXPIRED (409); up
// to 20 invitations go out a day (429).
func (s *RcsAgentsService) ResendTesterInvite(ctx context.Context, params ResendRcsTesterInviteParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	path, err := rcsAgentPath(params.AgentID, "/testers/"+url.PathEscape(params.PhoneNumber)+"/resend")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// UploadRcsAgentAssetParams are the parameters for the UploadAsset method.
type UploadRcsAgentAssetParams struct {
	// AgentID is the agent ID. Required.
	AgentID string
	// Kind is "logo" or "hero". Required.
	Kind string
	// File is the PNG or JPEG image; its Filename is optional. Required.
	File *FileUpload
}

// UploadAsset hosts the agent's logo (224 × 224 px, 50 KB max) or hero image (1440 × 448 px, 200 KB max), PNG or
// JPEG, sets it on the draft and returns the agent with the new URL. Images of any other size are refused.
func (s *RcsAgentsService) UploadAsset(ctx context.Context, params UploadRcsAgentAssetParams, opts *RequestOptions) (*Response, error) {
	path, writeForm, err := rcsAgentAssetForm(params)
	if err != nil {
		return nil, err
	}
	return s.client.doMultipart(ctx, "POST", path, writeForm, opts)
}

// rcsAgentPath validates agentID and returns /rcs/agent/{agentID}{suffix} with the ID escaped.
func rcsAgentPath(agentID, suffix string) (string, error) {
	if err := require(map[string]string{"agentId": agentID}); err != nil {
		return "", err
	}
	return fmt.Sprintf("/rcs/agent/%s%s", url.PathEscape(agentID), suffix), nil
}

// rcsAgentCreateBody validates a create request and builds its body; unset fields are omitted.
func rcsAgentCreateBody(params CreateRcsAgentParams) (map[string]any, error) {
	if err := require(map[string]string{"brandId": params.BrandID}); err != nil {
		return nil, err
	}
	body := map[string]any{"brandId": params.BrandID}
	setString(body, "campaignId", params.CampaignID)
	setAny(body, "agent", params.Agent)
	setAny(body, "consent", params.Consent)
	setAny(body, "contact", params.Contact)
	setString(body, "useCase", params.UseCase)
	setString(body, "billingCategory", params.BillingCategory)
	setAny(body, "defaultFallback", params.DefaultFallback)
	setSlice(body, "excludedCarriers", params.ExcludedCarriers)
	setSlice(body, "testers", params.Testers)
	return body, nil
}

// rcsAgentUpdateBody builds the partial update body; unset fields are omitted and the Clear flags send null.
func rcsAgentUpdateBody(params UpdateRcsAgentParams) map[string]any {
	body := map[string]any{}
	setString(body, "brandId", params.BrandID)
	if params.ClearCampaign {
		body["campaignId"] = nil
	} else {
		setString(body, "campaignId", params.CampaignID)
	}
	setAny(body, "agent", params.Agent)
	setAny(body, "consent", params.Consent)
	setAny(body, "contact", params.Contact)
	if params.ClearUseCase {
		body["useCase"] = nil
	} else {
		setString(body, "useCase", params.UseCase)
	}
	setString(body, "billingCategory", params.BillingCategory)
	if params.ClearDefaultFallback {
		body["defaultFallback"] = nil
	} else {
		setAny(body, "defaultFallback", params.DefaultFallback)
	}
	setSlice(body, "excludedCarriers", params.ExcludedCarriers)
	setSlice(body, "testers", params.Testers)
	return body
}

// rcsAgentAssetForm validates an upload and returns its path and a writer for the kind + file multipart form.
func rcsAgentAssetForm(params UploadRcsAgentAssetParams) (string, func(w *multipart.Writer) error, error) {
	path, err := rcsAgentPath(params.AgentID, "/asset")
	if err != nil {
		return "", nil, err
	}
	if err := require(map[string]string{"kind": params.Kind}); err != nil {
		return "", nil, err
	}
	if params.File == nil || params.File.Reader == nil {
		return "", nil, &ValidationError{Field: "file"}
	}
	return path, func(w *multipart.Writer) error {
		if err := w.WriteField("kind", params.Kind); err != nil {
			return err
		}
		filename := params.File.Filename
		if filename == "" {
			filename = "upload"
		}
		part, err := w.CreateFormFile("file", filename)
		if err != nil {
			return err
		}
		_, err = io.Copy(part, params.File.Reader)
		return err
	}, nil
}

// setString sets body[key] when value is non-empty.
func setString(body map[string]any, key, value string) {
	if value != "" {
		body[key] = value
	}
}

// setAny sets body[key] when value is non-nil, treating a typed nil pointer, map or slice as unset; the Clear flags
// are the way to send null.
func setAny(body map[string]any, key string, value any) {
	if value == nil {
		return
	}
	switch v := reflect.ValueOf(value); v.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface:
		if v.IsNil() {
			return
		}
	}
	body[key] = value
}

// setSlice sets body[key] when value is non-nil, so an empty non-nil slice is sent as [].
func setSlice(body map[string]any, key string, value []string) {
	if value != nil {
		body[key] = value
	}
}
