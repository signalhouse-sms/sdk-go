package signalhouse

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"reflect"
)

// RcsService groups RCS (Google RBM) resources. Access via client.Rcs.Agents and client.Rcs.Messages. Templates for RCS
// live under client.Templates.
type RcsService struct {
	client *Client

	// Agents — RCS agents under /rcs/agent.
	Agents *RcsAgentsService
	// Messages — RCS messages: send, validate, estimate, capabilities, get and list.
	Messages *RcsMessagesService
}

// RcsAgentsService handles RCS agents under /rcs/agent. An agent links one brand and one campaign, of any registration
// type: the campaign's numbers are the SMS fallback, and the brand and campaign supply the company identity and consent
// text for the carrier launch. One brand or campaign can serve many agents. An agent is a registration (type RCS_AGENT,
// id RCSA…), so the registrations methods read and edit the same record. 404 while the environment's RCS switch is
// off; 400 while the group's rcsEnabled flag is off.
type RcsAgentsService struct {
	client *Client
}

// RcsMessagesService handles RCS messages. Sends go through an agent to POST /message/rcs. A message is a Message record
// (messageType RCS), so client.Messages reads the same record. 404 while the environment's RCS switch is off; 403 while
// the group's rcsEnabled flag is off; 409 while the agent cannot send yet.
type RcsMessagesService struct {
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

// RcsAgentAutofillMarker says where one prefilled field's value came from. An agent's autofill holds one per field,
// keyed by its path (consent.optInDetails); the API marks a field edited once its value changes.
type RcsAgentAutofillMarker struct {
	// Source is brand, campaign or landing.
	Source string `json:"source"`
	// SourceID is that record's id (brand ID, campaign ID or landing page ID); empty omits it.
	SourceID string `json:"sourceId,omitempty"`
	// At is when the field was filled (ISO 8601); empty means now.
	At string `json:"at,omitempty"`
	// Edited says the value changed since.
	Edited bool `json:"edited"`
}

// CreateRcsAgentParams are the parameters for the Create method.
type CreateRcsAgentParams struct {
	// BrandID is the brand: carrier brand ID, provisional UUID or _id. Required.
	BrandID string
	// CampaignID is a campaign of that brand (campaign ID or _id); required before submit. Empty omits it.
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
	// BillingCategory is NON_CONVERSATIONAL (default, per message) or CONVERSATIONAL (per interactive session).
	BillingCategory string
	// DefaultFallback is an RcsAgentFallback, *RcsAgentFallback or map[string]any. nil omits it.
	DefaultFallback any
	// ExcludedCarriers is [] or ["T_MOBILE"] (the carrier partner's carrier name); nil omits it.
	ExcludedCarriers []string
	// Testers are tester numbers (up to 200), invited once the agent exists at the carrier; nil omits them.
	Testers []string
	// Autofill says where the fields you prefilled came from, by path; the API adds its own for the fields it fills. nil omits it.
	Autofill map[string]RcsAgentAutofillMarker
}

// Create creates an agent draft on a brand, optionally with its campaign, and returns the agent (HTTP 201,
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
	// Status is DRAFT, PENDING_REVIEW, REJECTED, AGENT_CREATING, ARCHIVE_PENDING or ARCHIVED, one value or a comma-separated list
	// (e.g. "DRAFT,REJECTED").
	Status string
	// BrandID returns agents linked to this brand (carrier brand ID, provisional UUID or _id).
	BrandID string
	// CampaignID returns agents linked to this campaign (campaign ID or _id).
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
	// AcknowledgeFees accepts the fees due on Launch; required when any are. Other methods ignore it.
	AcknowledgeFees bool
	// AcknowledgePhi accepts the PHI disclosure on Launch; required for a healthcare or insurance brand. Other methods ignore it.
	AcknowledgePhi bool
}

// Get returns one agent with the live state of its brand and campaign links. googleAgentId is Google's id for the agent,
// null until Google has created it.
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
	// BrandID relinks the agent to another brand of the same subgroup.
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
	// Autofill replaces the prefill markers; nil keeps the stored ones, and a changed field's marker turns edited.
	Autofill map[string]RcsAgentAutofillMarker
}

// Update edits an agent while it may change (a draft, a Signal House rejection, a request for changes, or an agent in
// testing, whose change is pushed to the carrier partner) and returns it. Agent, Consent and Contact merge field by
// field; every other key replaces. 409 in any other status; BrandID is locked once the agent exists at the partner and
// BillingCategory once a launch was requested. After a carrier rejected a launch other carriers took
// (PARTIALLY_LAUNCHED with a REJECTED carrier), only Consent and Contact may change, and nothing is pushed: send them
// with RequestCarrierUpdate.
func (s *RcsAgentsService) Update(ctx context.Context, params UpdateRcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", path, rcsAgentUpdateBody(params), opts)
}

// Validate checks an agent against every submit requirement without writing and returns {valid, errors:
// [{field, reason, code?}], warnings: [{field, reason}]}. A copy-rule error at the launch gate carries the carrier
// partner's key as code (for example OPT_IN_MSG_RATE_DISCL_MISSING).
func (s *RcsAgentsService) Validate(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/validate")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// Submit submits a draft: creates the agent at the carrier partner, and returns it (status
// AGENT_CREATING, then IN_TESTING once the partner reports it ready). It needs the full profile, a use case and
// a billing category; otherwise a 400 whose details carry {code: "REGISTRATION_INCOMPLETE", errors, warnings}.
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
// "REGISTRATION_INCOMPLETE", errors, warnings}, and a copy-rule error carries the carrier partner's key as its code (for
// example OPT_IN_MSG_RATE_DISCL_MISSING). A group that has not finished account verification gets a 403 with code
// GROUP_UNVERIFIED, and a rejected one a 403 with code GROUP_REJECTED, here and at Submit; a pending verification
// passes. It sends POST /registration/:agentId/launch: the agent is a registration.
// The request charges the agent's launch fees: a wallet that cannot cover them gets a 402, and the agent stays where it is. When any fee is due the request must set AcknowledgeFees, or it gets a 400 with code FEE_ACKNOWLEDGEMENT_REQUIRED and nothing is held; a launch with nothing due needs no acknowledgement. The agent's launchFees quotes what the next request charges.
// When the agent's brand vertical is healthcare or insurance, the launch needs the protected health information (PHI) disclosure acknowledged: set AcknowledgePhi, or the request gets a 400 REGISTRATION_INCOMPLETE whose errors name consent.phiAcknowledgement. The API records the acknowledgement as consent.phiAcknowledgement {acknowledgedAt, acknowledgedBy} from the caller; the client sends only the boolean, and an agent update cannot set it. Once recorded, a later launch request needs no new acknowledgement.
func (s *RcsAgentsService) Launch(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", fmt.Sprintf("/registration/%s/launch", url.PathEscape(params.AgentID)), launchBody(params.AcknowledgeFees, params.AcknowledgePhi), opts)
}

// RequestCarrierUpdate sends Signal House the corrected launch answers after a carrier rejected a launch other carriers
// took, and returns the agent's registration with data.carrierUpdate. The carrier partner takes no change through its
// API once a carrier launched, so Signal House emails it the update; edit Consent and Contact first. carrierUpdate reads
// REQUESTED for the rejected carriers, then SENT once Signal House sent it. 409 when no carrier is REJECTED after a
// partial launch, or an update already waits. It sends POST /registration/:agentId/carrier-update.
func (s *RcsAgentsService) RequestCarrierUpdate(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", fmt.Sprintf("/registration/%s/carrier-update", url.PathEscape(params.AgentID)), nil, opts)
}

// Archive archives an agent and returns it (status ARCHIVE_PENDING or ARCHIVED). One the carrier partner created stops
// sending at once and stays ARCHIVE_PENDING until Signal House confirms the partner cancelled it, then ARCHIVED.
// Archiving cannot be undone; an agent already ARCHIVE_PENDING answers 409.
func (s *RcsAgentsService) Archive(ctx context.Context, params RcsAgentParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/archive")
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// CreateRcsEntryPointsParams are the parameters for the CreateEntryPoints method.
type CreateRcsEntryPointsParams struct {
	// AgentID is the agent ID. Required.
	AgentID string
	// Body is the message the chat opens with, prefilled on the handset (max 1,000 characters). Empty omits it.
	Body string
}

// CreateEntryPoints makes the links that start a chat with the agent and returns { googleAgentId, deepLink, shortLink }:
// an sms: deep link in Google's format, and a short link that redirects to it, for a QR code. With the agent's
// defaultFallback.from, the deep link names that number, so a handset that cannot reach the agent texts it. The same
// request returns the same short link. The links reach approved testers until a carrier launches the agent. 409 with
// code GOOGLE_AGENT_PENDING before Google has created the agent.
func (s *RcsAgentsService) CreateEntryPoints(ctx context.Context, params CreateRcsEntryPointsParams, opts *RequestOptions) (*Response, error) {
	path, err := rcsAgentPath(params.AgentID, "/entry-points")
	if err != nil {
		return nil, err
	}
	body := map[string]any{}
	setString(body, "body", params.Body)
	return s.client.doJSON(ctx, "POST", path, body, opts)
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

// RemoveRcsTesterParams are the parameters for the RemoveTester method.
type RemoveRcsTesterParams struct {
	// AgentID is the agent ID. Required.
	AgentID string
	// PhoneNumber is the tester's US number, with or without the +. Required.
	PhoneNumber string
}

// RemoveTester removes a tester at once, at any point before the agent is archived, and returns the agent. The number
// must be a tester on the agent (404); an archived agent answers 409.
func (s *RcsAgentsService) RemoveTester(ctx context.Context, params RemoveRcsTesterParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	path, err := rcsAgentPath(params.AgentID, "/testers/"+url.PathEscape(params.PhoneNumber))
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
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

// SendRcsMessageParams are the parameters for the Send, SendBatch and Validate methods.
type SendRcsMessageParams struct {
	// AgentID is the agent that sends (RCSA…). Required.
	AgentID string
	// Recipients holds up to 100 recipients (10,000 on SendBatch), each { phoneNumber, variables?, clientMessageId?,
	// metadata? }. Required.
	Recipients []map[string]any
	// TemplateID is a template to send; exactly one of TemplateID or Content.
	TemplateID string
	// TemplateVersion is a published version of TemplateID; 0 omits it.
	TemplateVersion int
	// Content is inline content, as a map[string]any or json.RawMessage; exactly one of TemplateID or Content.
	Content any
	// ContentFormat is content.v1 (default) or google.rcs.
	ContentFormat string
	// TrafficType is AUTHENTICATION, TRANSACTION, SERVICEREQUEST, ACKNOWLEDGEMENT or PROMOTION; required on a MULTI_USE
	// agent when the template sets none.
	TrafficType string
	// TTLSeconds is how long the RCS message may wait for delivery; at most one of TTLSeconds or ExpireTime. 0 omits it.
	TTLSeconds int
	// ExpireTime is the ISO-8601 time the RCS message expires.
	ExpireTime string
	// Fallback is { mode? (auto, rcs_only, sms_only), channel? (auto, sms, mms), from?, content?: { text, mediaUrls? },
	// validitySeconds? }, as a map[string]any. from is a Ready number on the agent's linked Active campaign. If content is
	// omitted the template's fallback, else a generated one, is used. nil omits it.
	Fallback any
	// StatusCallbackURL receives this message's status events.
	StatusCallbackURL string
	// EnableShortlink shortens the links in the message; nil omits it.
	EnableShortlink *bool
	// IsTest sends as a test, to approved testers only; nil omits it.
	IsTest *bool
	// Simulate is not_capable, ttl_expired, delivered or read; approved testers only.
	Simulate string
}

// Send sends an RCS message to up to 100 recipients and returns { requested, enqueued, failed, bulkId,
// messages } (HTTP 201). A check that blocks one recipient writes a BLOCKED message for it and the rest still send; its row
// carries blockReason, reason and issues. An archived template answers 409 TEMPLATE_ARCHIVED. A recipient from 19999990000
// to 19999990359 makes the whole request a simulation, as on SMS: it returns { simulated: true, webhooks } and nothing is
// stored, sent or charged. Set RequestOptions.IdempotencyKey to make a retry safe.
func (s *RcsMessagesService) Send(ctx context.Context, params SendRcsMessageParams, opts *RequestOptions) (*Response, error) {
	body, err := rcsSendBody(params)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/message/rcs", body, opts)
}

// SendBatch sends an RCS message to up to 10,000 recipients in the background and returns { bulkId, status:
// "ACCEPTED", recipientCount } (HTTP 202). The request runs Send's request checks, splits the recipients into chunks of
// 100, and queues each chunk. Every message carries the shared bulkId; track progress with the messages list filtered by
// bulkId (GET /message?bulkId=) and per-message webhooks. A chunk the wallet cannot cover, or that is refused after
// acceptance, gets BLOCKED messages with blockReason INSUFFICIENT_FUNDS or SEND_REFUSED; a chunk that keeps failing gets
// SEND_FAILED. A recipient from 19999990000 to 19999990359 makes the whole request a simulation, as on Send. Set
// RequestOptions.IdempotencyKey to make a retry safe (same bulkId, no duplicate sends); a retry without one may send
// duplicates.
func (s *RcsMessagesService) SendBatch(ctx context.Context, params SendRcsMessageParams, opts *RequestOptions) (*Response, error) {
	body, err := rcsSendBody(params)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/message/rcs/batch", body, opts)
}

// Validate runs a send's checks without writing anything or holding funds. It returns { valid, errors, warnings,
// billingClass, segments, estimatedCostMicros, unsupportedFiles, recipients }. valid is true only with no errors; each
// blocked recipient adds an error and carries blockReason, reason and issues.
func (s *RcsMessagesService) Validate(ctx context.Context, params SendRcsMessageParams, opts *RequestOptions) (*Response, error) {
	body, err := rcsSendBody(params)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/message/rcs/validate", body, opts)
}

// EstimateRcsMessageParams are the parameters for the Estimate method.
type EstimateRcsMessageParams struct {
	// AgentID is the agent that would send (RCSA…). Required.
	AgentID string
	// Recipients are named recipients; exactly one of Recipients or RecipientCount.
	Recipients []map[string]any
	// RecipientCount is a recipient count; carriers are unknown, so the default fees apply. 0 omits it.
	RecipientCount int
	// TemplateID is a template to price; exactly one of TemplateID or Content.
	TemplateID string
	// TemplateVersion is a published version of TemplateID; 0 omits it.
	TemplateVersion int
	// Content is inline content, as a map[string]any or json.RawMessage; exactly one of TemplateID or Content.
	Content any
	// ContentFormat is content.v1 (default) or google.rcs.
	ContentFormat string
	// TrafficType is as on SendRcsMessageParams.
	TrafficType string
	// TTLSeconds is how long the RCS message may wait for delivery; at most one of TTLSeconds or ExpireTime. 0 omits it.
	TTLSeconds int
	// ExpireTime is the ISO-8601 time the RCS message expires.
	ExpireTime string
	// Fallback is as on SendRcsMessageParams, but its from is optional: with none, the leg is priced on the channel of the
	// agent's linked campaign. nil omits it.
	Fallback any
}

// Estimate prices an RCS draft in microdollars and returns { billingClass, segments, recipientCount, rcsLeg, fallbackLeg,
// totals, balance, issues, warnings }; it never sends or charges. Sends POST /message/estimate with channel "rcs". The fallback's from is
// optional for an estimate: with none, the fallback leg is priced on the channel of the agent's linked campaign.
// rcsLeg.mediaSurcharge is the carrier's surcharge on media over 5 MB for one recipient, and rcsLeg.cost already holds it for every recipient; rcsLeg.cost is null with the fallback mode sms_only; a message over 100 MB adds a MEDIA_TOO_LARGE issue.
func (s *RcsMessagesService) Estimate(ctx context.Context, params EstimateRcsMessageParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	body := map[string]any{"channel": "rcs", "agentId": params.AgentID}
	if params.Recipients != nil {
		body["recipients"] = params.Recipients
	}
	setInt(body, "recipientCount", params.RecipientCount)
	setString(body, "templateId", params.TemplateID)
	setInt(body, "templateVersion", params.TemplateVersion)
	setAny(body, "content", params.Content)
	setString(body, "contentFormat", params.ContentFormat)
	setString(body, "trafficType", params.TrafficType)
	setInt(body, "ttlSeconds", params.TTLSeconds)
	setString(body, "expireTime", params.ExpireTime)
	setAny(body, "fallback", params.Fallback)
	return s.client.doJSON(ctx, "POST", "/message/estimate", body, opts)
}

// RcsCapabilitiesParams are the parameters for the Capabilities method.
type RcsCapabilitiesParams struct {
	// AgentID is the agent (RCSA…). Required.
	AgentID string
	// PhoneNumbers holds up to 500 numbers, 10-15 digits, with or without the +. Required.
	PhoneNumbers []string
}

// Capabilities reports which numbers can receive RCS from an agent as { results: [{ phoneNumber, capability, rcsEnabled, reason, features, carrier, checkedAt, cached }] },
// capability being ENABLED, UNREACHABLE or UNKNOWN, and reason CAPABLE, NOT_CAPABLE, NOT_LAUNCHED_ON_CARRIER or UNKNOWN; rcsEnabled is true only for CAPABLE, carrier and checkedAt may be null, and features is empty today.
func (s *RcsMessagesService) Capabilities(ctx context.Context, params RcsCapabilitiesParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/rcs/capabilities", map[string]any{"agentId": params.AgentID, "phoneNumbers": params.PhoneNumbers}, opts)
}

// SimulateRcsInboundParams are the parameters for the SimulateInbound method.
type SimulateRcsInboundParams struct {
	// AgentID is the agent (RCSA…). Required.
	AgentID string
	// From is the approved tester's number. Required.
	From string
	// Type is TEXT, SUGGESTION, LOCATION, FILE or UNSUBSCRIBE. Required.
	Type string
	// Text is the text; required for TEXT.
	Text string
	// PostbackData is the tapped suggestion's postback; SUGGESTION needs this or ButtonID.
	PostbackData string
	// MessageID is the message a tap answers; the latest message delivered to the tester by default.
	MessageID string
	// ButtonID is the suggestion's position, as webhooks report it.
	ButtonID string
	// Latitude is required for LOCATION.
	Latitude *float64
	// Longitude is required for LOCATION.
	Longitude *float64
	// FileName is the file's name, for FILE.
	FileName string
}

// SimulateInbound fakes an inbound event. Fake an inbound event from an approved tester of the agent, with no carrier involved: it runs the real inbound handlers and webhooks, as free test traffic. A simulated STOP really opts the tester out; a simulated START clears it. Anyone else gets a 403 with code NOT_TESTER. Returns 202 with { status, messageId }.
func (s *RcsMessagesService) SimulateInbound(ctx context.Context, params SimulateRcsInboundParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentId": params.AgentID, "from": params.From, "type": params.Type}); err != nil {
		return nil, err
	}
	body := map[string]any{"agentId": params.AgentID, "from": params.From, "type": params.Type}
	for key, value := range map[string]string{"text": params.Text, "postbackData": params.PostbackData, "messageId": params.MessageID, "buttonId": params.ButtonID, "fileName": params.FileName} {
		if value != "" {
			body[key] = value
		}
	}
	if params.Latitude != nil {
		body["latitude"] = *params.Latitude
	}
	if params.Longitude != nil {
		body["longitude"] = *params.Longitude
	}
	return s.client.doJSON(ctx, "POST", "/message/rcs/simulate-inbound", body, opts)
}

// GetRcsMessageParams are the parameters for the Get method.
type GetRcsMessageParams struct {
	// MessageID is the message ID. Required.
	MessageID string
}

// Get returns one message by its ID; an unknown ID is a 404. The same route as client.Messages.GetMessage. A settled RCS
// message carries carrierFeeMicros, the carrier pass-through and media surcharge it was charged.
func (s *RcsMessagesService) Get(ctx context.Context, params GetRcsMessageParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"messageId": params.MessageID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/message/"+url.PathEscape(params.MessageID), nil, opts)
}

// ListRcsMessagesParams are the parameters for the List method.
type ListRcsMessagesParams struct {
	// AgentID returns messages this agent sent; sent as registrationId.
	AgentID string
	// From is an ISO-8601 date or timestamp; sent as startDate. Required.
	From string
	// To is an ISO-8601 date or timestamp; sent as endDate. Required.
	To string
	// GroupID is the group whose messages to list. Required.
	GroupID string
	// Status is ENQUEUED, DEQUEUED, SENT, FAILED, DELIVERED, READ, EXPIRED, FALLBACK or BLOCKED.
	Status          string
	Page            string
	Limit           string
	TemplateID      string
	TemplateVersion string
	// BulkID returns the messages of one send.
	BulkID string
	// FallbackReason is NOT_CAPABLE, CARRIER_NOT_LAUNCHED, TTL_EXPIRED, RCS_ERROR, PROVIDER_ERROR or SMS_ONLY.
	FallbackReason string
	// BillingClass is RICH or RICH_MEDIA.
	BillingClass string
	// DeliveredChannel is the channel that delivered the message: rcs, sms or mms.
	DeliveredChannel string
	// IsTest set to true returns test messages only, and to false leaves them out. nil omits it.
	IsTest *bool
}

// List lists RCS messages. Sends GET /message with messageType RCS. A fallback leg is an SMS or MMS message: read it
// with client.Messages.GetMessage and the parent's fallbackMessageId.
func (s *RcsMessagesService) List(ctx context.Context, params ListRcsMessagesParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "from": params.From, "to": params.To}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"registrationId":   params.AgentID,
		"startDate":        params.From,
		"endDate":          params.To,
		"groupId":          params.GroupID,
		"status":           params.Status,
		"page":             params.Page,
		"limit":            params.Limit,
		"templateId":       params.TemplateID,
		"templateVersion":  params.TemplateVersion,
		"bulkId":           params.BulkID,
		"fallbackReason":   params.FallbackReason,
		"billingClass":     params.BillingClass,
		"deliveredChannel": params.DeliveredChannel,
		"isTest":           boolQueryValue(params.IsTest),
		"messageType":      "RCS",
		"dateBounds":       "exact",
	})
	return s.client.doJSON(ctx, "GET", "/message"+qs, nil, opts)
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
	setAutofill(body, params.Autofill)
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
	setAutofill(body, params.Autofill)
	return body
}

// setAutofill sets the prefill markers when given; nil omits them.
func setAutofill(body map[string]any, markers map[string]RcsAgentAutofillMarker) {
	if markers != nil {
		body["autofill"] = markers
	}
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

// rcsSendBody validates a send and builds its JSON body.
func rcsSendBody(params SendRcsMessageParams) (map[string]any, error) {
	if err := require(map[string]string{"agentId": params.AgentID}); err != nil {
		return nil, err
	}
	if len(params.Recipients) == 0 {
		return nil, &ValidationError{Field: "recipients"}
	}
	body := map[string]any{"agentId": params.AgentID, "recipients": params.Recipients}
	setString(body, "templateId", params.TemplateID)
	setInt(body, "templateVersion", params.TemplateVersion)
	setAny(body, "content", params.Content)
	setString(body, "contentFormat", params.ContentFormat)
	setString(body, "trafficType", params.TrafficType)
	setInt(body, "ttlSeconds", params.TTLSeconds)
	setString(body, "expireTime", params.ExpireTime)
	setAny(body, "fallback", params.Fallback)
	setString(body, "statusCallbackUrl", params.StatusCallbackURL)
	setAny(body, "enableShortlink", params.EnableShortlink)
	setAny(body, "isTest", params.IsTest)
	setString(body, "simulate", params.Simulate)
	return body, nil
}

// setString sets body[key] when value is non-empty.
func setString(body map[string]any, key, value string) {
	if value != "" {
		body[key] = value
	}
}

// setInt sets body[key] when value is non-zero.
func setInt(body map[string]any, key string, value int) {
	if value != 0 {
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
