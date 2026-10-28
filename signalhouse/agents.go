package signalhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

// AgentsService handles AI Agent profile API calls under the customer-facing
// /agent domain. Access via client.Agents.
type AgentsService struct {
	client *Client
}

// CreateAgentProfileData represents the data for creating an agent profile.
type CreateAgentProfileData struct {
	// GroupID is the parent group id (at least 8 characters, starts with 'G'). Required.
	GroupID string `json:"groupId"`
	// SubgroupID is an optional subgroup id (at least 8 characters, starts with 'S');
	// nil/omitted = group-level agent.
	SubgroupID *string `json:"subgroupId,omitempty"`
	// Name is the agent's name. Required.
	Name string `json:"name"`
	// Status is the lifecycle status: "active" or "inactive" (defaults to "active").
	Status string `json:"status,omitempty"`
	// SystemPrompt is the base system prompt composed into every conversation.
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Greeting is the opening line the agent leads with.
	Greeting string `json:"greeting,omitempty"`
	// Guardrails are freeform guardrail instructions appended to the composed prompt.
	Guardrails string `json:"guardrails,omitempty"`
	// SendAuthority is the default send authority: "review" or "autopilot" (defaults to
	// "review"). Deployments may override it per subgroup.
	SendAuthority string `json:"sendAuthority,omitempty"`
}

// UpdateAgentProfileData represents the data for updating an agent profile. Every
// field is optional; the agent's scope (groupId and subgroupId) is immutable and
// therefore not updatable.
type UpdateAgentProfileData struct {
	// Name is the agent's name.
	Name string `json:"name,omitempty"`
	// Status is the lifecycle status: "active" or "inactive".
	Status string `json:"status,omitempty"`
	// SystemPrompt is the base system prompt composed into every conversation.
	SystemPrompt string `json:"systemPrompt,omitempty"`
	// Greeting is the opening line the agent leads with.
	Greeting string `json:"greeting,omitempty"`
	// Guardrails are freeform guardrail instructions appended to the composed prompt.
	Guardrails string `json:"guardrails,omitempty"`
	// SendAuthority is the default send authority: "review" or "autopilot". Deployments may
	// override it per subgroup. The read-only publishStatus and publishedAt fields change only
	// through PublishAgentProfile and UnpublishAgentProfile.
	SendAuthority string `json:"sendAuthority,omitempty"`
}

// GetAgentProfilesParams are the parameters for the GetAgentProfiles method.
type GetAgentProfilesParams struct {
	// GroupID is the group whose profiles to list. Required.
	GroupID string
	// SubgroupID narrows to the agents this subgroup can use — its own, plus the
	// group-level agents shared with every subgroup.
	SubgroupID string
	// Page is the page number for pagination.
	Page string
	// Limit is the number of items per page for pagination.
	Limit string
}

// GetAgentVoices lists the voices an agent can be configured to speak with.
//
// The catalog is platform-wide, not per-account, so this takes no parameters. The
// voiceId of an entry is the value to set on a spoken channel's setting.
// Roles: api, admin, developer, billing, user.
// GET /agent/voices
func (s *AgentsService) GetAgentVoices(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/agent/voices", nil, opts)
}

// GetAgentProfiles lists the agent profiles under a group.
// Roles: api, admin, developer, billing, user.
// GET /agent/profiles
func (s *AgentsService) GetAgentProfiles(ctx context.Context, params GetAgentProfilesParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId":    params.GroupID,
		"subgroupId": params.SubgroupID,
		"page":       params.Page,
		"limit":      params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/agent/profiles"+qs, nil, opts)
}

// GetAgentProfileParams are the parameters for the GetAgentProfile method.
type GetAgentProfileParams struct {
	AgentProfileID string
}

// GetAgentProfile returns a single agent profile by id.
// Roles: api, admin, developer, billing, user.
// GET /agent/profiles/{agentProfileId}
func (s *AgentsService) GetAgentProfile(ctx context.Context, params GetAgentProfileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s", url.PathEscape(params.AgentProfileID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// CreateAgentProfileParams are the parameters for the CreateAgentProfile method.
type CreateAgentProfileParams struct {
	ProfileData CreateAgentProfileData
}

// CreateAgentProfile creates a new agent profile.
// Roles: api, admin, developer.
// POST /agent/profiles
func (s *AgentsService) CreateAgentProfile(ctx context.Context, params CreateAgentProfileParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/agent/profiles", params.ProfileData, opts)
}

// UpdateAgentProfileParams are the parameters for the UpdateAgentProfile method.
type UpdateAgentProfileParams struct {
	ID         string
	UpdateData UpdateAgentProfileData
}

// UpdateAgentProfile updates an existing agent profile. The agent's scope
// (groupId and subgroupId) is immutable and cannot be changed.
// Roles: api, admin, developer.
// PUT /agent/profiles/{id}
func (s *AgentsService) UpdateAgentProfile(ctx context.Context, params UpdateAgentProfileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// DeleteAgentProfileParams are the parameters for the DeleteAgentProfile method.
type DeleteAgentProfileParams struct {
	ID string
}

// DeleteAgentProfile permanently deletes an agent profile by its id.
//
// Permanently deletes the agent, its channel settings, and any tools and knowledge scoped to it, plus their provider-side counterparts. Tools and knowledge shared across the account are untouched. Conversation records survive — a transcript outlives the agent that produced it.
// Roles: api, admin, developer.
// DELETE /agent/profiles/{id}
func (s *AgentsService) DeleteAgentProfile(ctx context.Context, params DeleteAgentProfileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// SendAgentMessageParams are the parameters (and JSON body) for the SendAgentMessage
// method. AgentProfileID, Channel, and Message are required; the optional fields are
// omitted from the request body when empty.
type SendAgentMessageParams struct {
	// AgentProfileID is the agent to talk to. Required.
	AgentProfileID string `json:"agentProfileId"`
	// Channel is the conversation channel: "webchat", "sms", or "voice". Required.
	Channel string `json:"channel"`
	// Message is the user's message. Required.
	Message string `json:"message"`
	// ConversationID continues an existing conversation; empty starts a new one.
	ConversationID string `json:"conversationId,omitempty"`
	// ContactIdentifier is the far-end identifier (webchat visitor id, sender number).
	ContactIdentifier string `json:"contactIdentifier,omitempty"`
}

// SendAgentMessage sends a message to an agent and returns its reply — the app-native
// conversation endpoint (webchat / SMS). Omit ConversationID to start a new conversation,
// or set it to continue an existing one. The agent's LLM tool loop runs server-side and
// the response carries conversationId, conversationSessionId, reply, toolInvocations, and ok.
// Roles: api, admin, developer, billing, user.
// POST /agent/messages
func (s *AgentsService) SendAgentMessage(ctx context.Context, params SendAgentMessageParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"agentProfileId": params.AgentProfileID,
		"channel":        params.Channel,
		"message":        params.Message,
	}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/agent/messages", params, opts)
}

// StartConversationParams are the parameters (and JSON body) for StartConversation.
// AgentProfileID and Channel are required; the optional fields are omitted from the
// request body when empty.
type StartConversationParams struct {
	// AgentProfileID is the agent this conversation belongs to. Required.
	AgentProfileID string `json:"agentProfileId"`
	// Channel is the conversation channel: "webchat", "sms", or "voice". Required.
	Channel string `json:"channel"`
	// ContactIdentifier is the far-end identifier (visitor id, caller number).
	ContactIdentifier string `json:"contactIdentifier,omitempty"`
	// CallID is the telephony call id, for the voice channel.
	CallID string `json:"callId,omitempty"`
	// Metadata is arbitrary metadata stored with the conversation.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// StartConversation starts a conversation without sending a message.
//
// The counterpart to SendAgentMessage for callers that run the model themselves — a voice
// runtime, or your own LLM. Those record what was said; SendAgentMessage decides it. Both
// write the same conversation records.
// Roles: api, admin, developer, billing, user.
// POST /agent/conversations
func (s *AgentsService) StartConversation(ctx context.Context, params StartConversationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"agentProfileId": params.AgentProfileID,
		"channel":        params.Channel,
	}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/agent/conversations", params, opts)
}

// GetConversationParams are the parameters for GetConversation.
type GetConversationParams struct {
	ConversationID string
	// Page is the page number; empty defaults to 1.
	Page string
	// Limit is the messages per page; empty defaults to 100 and is capped at 500.
	Limit string
}

// GetConversation reads one conversation with a page of its messages, in order.
//
// Paged rather than whole: a transcript has no natural ceiling. Leaving Limit empty returns
// 100 messages, not all of them.
// Roles: api, admin, developer, billing, user.
// GET /agent/conversations/{conversationId}
func (s *AgentsService) GetConversation(ctx context.Context, params GetConversationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"conversationId": params.ConversationID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"page": params.Page, "limit": params.Limit})
	path := fmt.Sprintf("/agent/conversations/%s%s", url.PathEscape(params.ConversationID), qs)
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// AppendConversationMessageData is the JSON body for appending one message. Optional
// fields use pointers so a zero value (e.g. ttfbMs=0, bargeInOccurred=false) can be sent
// explicitly rather than omitted.
type AppendConversationMessageData struct {
	// Role is "user", "assistant", "system", or "tool". Required.
	Role string `json:"role"`
	// Content is what was said. Optional — a tool-only turn needs no prose.
	Content *string `json:"content,omitempty"`
	// ToolCalls are the tools invoked on this turn.
	ToolCalls *[]map[string]any `json:"toolCalls,omitempty"`
	// ToolResults are their results.
	ToolResults *[]map[string]any `json:"toolResults,omitempty"`
	// TTFBMs is time to first byte/audio, in milliseconds.
	TTFBMs *int `json:"ttfbMs,omitempty"`
	// LatencyMs is total turn latency, in milliseconds.
	LatencyMs *int `json:"latencyMs,omitempty"`
	// BargeInOccurred is whether the far end talked over this turn.
	BargeInOccurred *bool `json:"bargeInOccurred,omitempty"`
}

// AppendConversationMessageParams are the parameters for AppendConversationMessage.
type AppendConversationMessageParams struct {
	ConversationID string
	Data           AppendConversationMessageData
}

// AppendConversationMessage appends one message to a conversation.
//
// Records a turn rather than generating one, so Role is explicit — an assistant turn your
// own runtime produced is the normal case here.
// Roles: api, admin, developer, billing, user.
// POST /agent/conversations/{conversationId}/messages
func (s *AgentsService) AppendConversationMessage(ctx context.Context, params AppendConversationMessageParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"conversationId": params.ConversationID,
		"role":           params.Data.Role,
	}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/conversations/%s/messages", url.PathEscape(params.ConversationID))
	return s.client.doJSON(ctx, "POST", path, params.Data, opts)
}

// EndConversationData is the JSON body for ending a conversation.
type EndConversationData struct {
	// Status records HOW it ended: "completed", "escalated", or "abandoned". Defaults to
	// "completed" server-side when empty.
	Status string `json:"status,omitempty"`
	// Metadata is merged onto the conversation.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EndConversationParams are the parameters for EndConversation.
type EndConversationParams struct {
	ConversationID string
	Data           EndConversationData
}

// EndConversation ends a conversation, closing its open session.
//
// Status records HOW it ended and cannot be recovered afterwards, so pass the one that
// actually happened. Ending an already-ended conversation is a no-op, so a retry after a
// dropped connection cannot overwrite it.
// Roles: api, admin, developer, billing, user.
// PUT /agent/conversations/{conversationId}
func (s *AgentsService) EndConversation(ctx context.Context, params EndConversationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"conversationId": params.ConversationID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/conversations/%s", url.PathEscape(params.ConversationID))
	return s.client.doJSON(ctx, "PUT", path, params.Data, opts)
}

// UpsertAgentChannelSettingData is the JSON body for an upsert of a per-channel setting.
// Optional fields use pointers so a zero value (e.g. enabled=false, empty tool list) can
// be sent explicitly rather than omitted: a nil field is dropped from the request, while a
// non-nil field — including &[]string{} to clear every tool — is sent as its JSON value.
type UpsertAgentChannelSettingData struct {
	// AllowedTools are the tools enabled on this channel (built-in keys or agentTool ids).
	// nil omits the field (leave unchanged); &[]string{} sends [] to revoke all tools.
	AllowedTools *[]string `json:"allowedTools,omitempty"`
	// Enabled controls whether the agent answers on this channel.
	Enabled *bool `json:"enabled,omitempty"`
	// ChannelPrompt is channel-specific prompt text appended to the agent's system prompt.
	ChannelPrompt *string `json:"channelPrompt,omitempty"`
	// Greeting is a channel-specific greeting override.
	Greeting *string `json:"greeting,omitempty"`
	// LLMProvider is the LLM backend for this channel: "bedrock", "openai", "anthropic",
	// or "groq" (defaults to "bedrock"). Model settings are per-channel because each
	// channel is served by a different runtime and the model catalogues do not overlap.
	LLMProvider *string `json:"llmProvider,omitempty"`
	// LLMModel is the concrete model id for the provider; nil = provider default.
	LLMModel *string `json:"llmModel,omitempty"`
	// Temperature is the sampling temperature (0-2).
	Temperature *float64 `json:"temperature,omitempty"`
	// VoiceID is the voice identifier for spoken channels; ignored off the voice channel.
	VoiceID *string `json:"voiceId,omitempty"`
	// Speed is the speaking rate multiplier (0.7-1.2); nil = platform default.
	Speed *float64 `json:"speed,omitempty"`
	// Stability is voice consistency (0-1); lower is more expressive and more variable.
	Stability *float64 `json:"stability,omitempty"`
	// SimilarityBoost is how closely synthesis tracks the source voice (0-1).
	SimilarityBoost *float64 `json:"similarityBoost,omitempty"`
	// SpeechModel is the speech synthesis model. Separate from LLMModel, which decides what
	// the agent says rather than how it is voiced; the two are chosen independently.
	SpeechModel *string `json:"speechModel,omitempty"`
	// TurnEagerness is how readily the agent decides the caller has finished speaking:
	// "patient", "normal", or "eager".
	TurnEagerness *string `json:"turnEagerness,omitempty"`
	// TurnTimeoutSeconds is the silence before the agent takes its turn: 1-300, or -1 for
	// no timeout. Values between 0 and 1 are rejected.
	TurnTimeoutSeconds *float64 `json:"turnTimeoutSeconds,omitempty"`
	// InitialWaitSeconds is the silence before the agent speaks first: 1-300, or -1 for
	// no timeout.
	InitialWaitSeconds *float64 `json:"initialWaitSeconds,omitempty"`
	// SilenceEndCallSeconds is the silence before the call is ended outright (10-7200).
	SilenceEndCallSeconds *float64 `json:"silenceEndCallSeconds,omitempty"`
	// MaxCallDurationSeconds is the hard ceiling on one conversation (60-7200).
	MaxCallDurationSeconds *float64 `json:"maxCallDurationSeconds,omitempty"`
	// AllowGreetingInterruption is whether a caller may talk over the greeting.
	AllowGreetingInterruption *bool `json:"allowGreetingInterruption,omitempty"`
	// Keyterms are words the transcriber is biased toward — names, jargon, unusual spellings.
	// A pointer, like AllowedTools and for the same reason: omitempty drops a zero-length slice,
	// so a bare []string could never send [] and a caller who had set keyterms would have no way
	// to clear them — an omitted field leaves the stored value unchanged.
	Keyterms *[]string `json:"keyterms,omitempty"`
	// BackgroundSound is the ambient audio bed; nil = silence.
	BackgroundSound *AgentBackgroundSound `json:"backgroundSound,omitempty"`
	// Workflow is the conversation workflow graph, in the voice runtime's own shape: "nodes"
	// and "edges" keyed maps, plus optional "subgraphs" and "prevent_subagent_loops". Node
	// bodies are defined by the runtime and are passed through as-is. Voice channel only —
	// sending one on webchat or sms is rejected.
	//
	// A pointer, like Keyterms and for the same reason: omitempty drops a nil map, so a bare
	// map[string]any could never send null and an agent that had a workflow could never have it
	// removed. Point at a nil map to clear one; leave this nil to leave the stored graph alone.
	Workflow *map[string]any `json:"workflow,omitempty"`
}

// AgentBackgroundSound is an ambient audio bed played under a call. Preset names come from a
// fixed set provided by the speech runtime; customer-uploaded audio is not supported.
type AgentBackgroundSound struct {
	// Preset is the ambient bed to play, e.g. "office1" or "restaurant".
	Preset string `json:"preset"`
	// Volume is the playback level (0.01-1).
	Volume *float64 `json:"volume,omitempty"`
}

// GetAgentChannelSettingsParams are the parameters for GetAgentChannelSettings.
type GetAgentChannelSettingsParams struct {
	AgentProfileID string
}

// GetAgentChannelSettings lists an agent's per-channel settings.
// Roles: api, admin, developer, billing, user.
// GET /agent/profiles/{agentProfileId}/channels
func (s *AgentsService) GetAgentChannelSettings(ctx context.Context, params GetAgentChannelSettingsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/channels", url.PathEscape(params.AgentProfileID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// GetAgentChannelSettingParams are the parameters for GetAgentChannelSetting.
type GetAgentChannelSettingParams struct {
	AgentProfileID string
	Channel        string
}

// GetAgentChannelSetting returns a single per-channel setting for an agent.
// Roles: api, admin, developer, billing, user.
// GET /agent/profiles/{agentProfileId}/channels/{channel}
func (s *AgentsService) GetAgentChannelSetting(ctx context.Context, params GetAgentChannelSettingParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID, "channel": params.Channel}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/channels/%s", url.PathEscape(params.AgentProfileID), url.PathEscape(params.Channel))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// UpsertAgentChannelSettingParams are the parameters for UpsertAgentChannelSetting.
type UpsertAgentChannelSettingParams struct {
	AgentProfileID string
	Channel        string
	SettingData    UpsertAgentChannelSettingData
}

// UpsertAgentChannelSetting creates or updates an agent's per-channel setting (one row per
// agent+channel; the channel comes from the path).
// Roles: api, admin, developer.
// PUT /agent/profiles/{agentProfileId}/channels/{channel}
func (s *AgentsService) UpsertAgentChannelSetting(ctx context.Context, params UpsertAgentChannelSettingParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID, "channel": params.Channel}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/channels/%s", url.PathEscape(params.AgentProfileID), url.PathEscape(params.Channel))
	return s.client.doJSON(ctx, "PUT", path, params.SettingData, opts)
}

// DeleteAgentChannelSettingParams are the parameters for DeleteAgentChannelSetting.
type DeleteAgentChannelSettingParams struct {
	AgentProfileID string
	Channel        string
}

// DeleteAgentChannelSetting deletes an agent's per-channel setting.
// Roles: api, admin, developer.
// DELETE /agent/profiles/{agentProfileId}/channels/{channel}
func (s *AgentsService) DeleteAgentChannelSetting(ctx context.Context, params DeleteAgentChannelSettingParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID, "channel": params.Channel}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/channels/%s", url.PathEscape(params.AgentProfileID), url.PathEscape(params.Channel))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// UpsertAgentDeploymentData is the JSON body for an upsert of an agent's deployment to a
// subgroup. A nil field is dropped from the request and leaves the stored value unchanged.
type UpsertAgentDeploymentData struct {
	// SendAuthorityOverride overrides the profile's default send authority for this
	// subgroup: "review" or "autopilot".
	SendAuthorityOverride *string
	// ClearSendAuthorityOverride sends sendAuthorityOverride as null, clearing the override so
	// the deployment inherits the profile default. It takes precedence over
	// SendAuthorityOverride.
	ClearSendAuthorityOverride bool
	// Enabled controls whether this deployment is live (defaults to true on create).
	Enabled *bool
}

// MarshalJSON encodes the deployment body, writing an explicit null for
// sendAuthorityOverride when ClearSendAuthorityOverride is set.
func (d UpsertAgentDeploymentData) MarshalJSON() ([]byte, error) {
	body := map[string]any{}
	if d.ClearSendAuthorityOverride {
		body["sendAuthorityOverride"] = nil
	} else if d.SendAuthorityOverride != nil {
		body["sendAuthorityOverride"] = *d.SendAuthorityOverride
	}
	if d.Enabled != nil {
		body["enabled"] = *d.Enabled
	}
	return json.Marshal(body)
}

// GetAgentDeploymentsParams are the parameters for GetAgentDeployments.
type GetAgentDeploymentsParams struct {
	AgentProfileID string
}

// GetAgentDeployments lists a group-level agent's deployments to subgroups. Each deployment
// carries agentDeploymentId, groupId, subgroupId, agentProfileId, sendAuthorityOverride,
// sendAuthorityChangedBy, sendAuthorityChangedAt, enabled, createdAt, and updatedAt.
// Roles: api, admin, developer, billing, user.
// GET /agent/profiles/{agentProfileId}/deployments
func (s *AgentsService) GetAgentDeployments(ctx context.Context, params GetAgentDeploymentsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/deployments", url.PathEscape(params.AgentProfileID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// UpsertAgentDeploymentParams are the parameters for UpsertAgentDeployment.
type UpsertAgentDeploymentParams struct {
	AgentProfileID string
	SubgroupID     string
	DeploymentData UpsertAgentDeploymentData
}

// UpsertAgentDeployment deploys a group-level agent to a subgroup, or changes that
// deployment (one row per agent+subgroup; the subgroup comes from the path).
// Roles: api, admin, developer.
// PUT /agent/profiles/{agentProfileId}/deployments/{subgroupId}
func (s *AgentsService) UpsertAgentDeployment(ctx context.Context, params UpsertAgentDeploymentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID, "subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/deployments/%s", url.PathEscape(params.AgentProfileID), url.PathEscape(params.SubgroupID))
	return s.client.doJSON(ctx, "PUT", path, params.DeploymentData, opts)
}

// DeleteAgentDeploymentParams are the parameters for DeleteAgentDeployment.
type DeleteAgentDeploymentParams struct {
	AgentProfileID string
	SubgroupID     string
}

// DeleteAgentDeployment removes an agent's deployment from a subgroup and returns the
// removed deployment.
// Roles: api, admin, developer.
// DELETE /agent/profiles/{agentProfileId}/deployments/{subgroupId}
func (s *AgentsService) DeleteAgentDeployment(ctx context.Context, params DeleteAgentDeploymentParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID, "subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/deployments/%s", url.PathEscape(params.AgentProfileID), url.PathEscape(params.SubgroupID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// GetAgentEndpointsParams are the parameters for GetAgentEndpoints.
type GetAgentEndpointsParams struct {
	// GroupID is the group whose bindings to list. Required.
	GroupID string
	// SubgroupID narrows to one subgroup's bindings.
	SubgroupID string
	// AgentProfileID narrows to one agent's bindings.
	AgentProfileID string
	// Channel narrows to one channel ("sms").
	Channel string
}

// GetAgentEndpoints lists the endpoint bindings (which agent answers which number) in a
// group, ordered by number. Each binding carries agentEndpointId, groupId, subgroupId,
// channel, endpointType, endpointValue, agentProfileId, answerMode, overrides, enabled,
// createdAt, and updatedAt.
// Roles: api, admin, developer, billing, user.
// GET /agent/endpoints
func (s *AgentsService) GetAgentEndpoints(ctx context.Context, params GetAgentEndpointsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"agentProfileId": params.AgentProfileID,
		"channel":        params.Channel,
	})
	return s.client.doJSON(ctx, "GET", "/agent/endpoints"+qs, nil, opts)
}

// UpsertAgentEndpointData is the JSON body for assigning an agent to a number.
type UpsertAgentEndpointData struct {
	// AgentProfileID is the agent that answers the number. Required. The agent must be
	// active, in the number's group, and either belong to the number's subgroup or be a
	// group-level agent with an enabled deployment there.
	AgentProfileID string `json:"agentProfileId"`
	// AnswerMode is which inbound messages the agent answers: "all_inbound" (the only
	// mode, and the default).
	AnswerMode string `json:"answerMode,omitempty"`
}

// UpsertAgentEndpointParams are the parameters for UpsertAgentEndpoint.
type UpsertAgentEndpointParams struct {
	// Channel is the channel to bind; must be "sms".
	Channel string
	// EndpointValue is the phone number: 10 to 15 digits, country code included, no "+".
	EndpointValue string
	// EndpointData is the agent to assign and its answer mode.
	EndpointData UpsertAgentEndpointData
}

// UpsertAgentEndpoint assigns an agent to answer a number (upsert keyed by channel +
// number). Assigning again replaces the previous agent, so a number has one agent per
// channel. Only SMS numbers can be bound.
// Roles: api, admin, developer.
// PUT /agent/endpoints/{channel}/{endpointValue}
func (s *AgentsService) UpsertAgentEndpoint(ctx context.Context, params UpsertAgentEndpointParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"channel": params.Channel, "endpointValue": params.EndpointValue, "agentProfileId": params.EndpointData.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/endpoints/%s/%s", url.PathEscape(params.Channel), url.PathEscape(params.EndpointValue))
	return s.client.doJSON(ctx, "PUT", path, params.EndpointData, opts)
}

// DeleteAgentEndpointParams are the parameters for DeleteAgentEndpoint.
type DeleteAgentEndpointParams struct {
	// Channel is the bound channel; must be "sms".
	Channel string
	// EndpointValue is the phone number: digits only, country code included, no "+".
	EndpointValue string
}

// DeleteAgentEndpoint removes the agent assigned to a number and returns the removed
// binding.
// Roles: api, admin, developer.
// DELETE /agent/endpoints/{channel}/{endpointValue}
func (s *AgentsService) DeleteAgentEndpoint(ctx context.Context, params DeleteAgentEndpointParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"channel": params.Channel, "endpointValue": params.EndpointValue}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/endpoints/%s/%s", url.PathEscape(params.Channel), url.PathEscape(params.EndpointValue))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// GetAgentReplyDraftsParams are the parameters for GetAgentReplyDrafts.
type GetAgentReplyDraftsParams struct {
	// GroupID is the group whose drafts to list. Required.
	GroupID string
	// SubgroupID narrows to one subgroup's drafts.
	SubgroupID string
	// AgentProfileID narrows to one agent's drafts.
	AgentProfileID string
	// Status narrows to one status: "pending", "sending", "sent", "rejected" or "failed".
	Status string
	// Page is the page number for pagination.
	Page string
	// Limit is the number of items per page (default 25, max 500).
	Limit string
}

// GetAgentReplyDrafts lists held agent replies (the Pending Replies queue) in a group,
// newest first. Each draft carries agentReplyDraftId, groupId, subgroupId, agentProfileId,
// conversationId, channel, endpointValue, contactPhoneNumber, inboundText, draftText,
// sentText, reason, status, decidedBy, decidedAt, sentMessageId, deliveredBy ("signalhouse" or
// "external"; null until decided), failureReason ("delivery_unconfirmed" or null), claimId (the
// current claim's id; null until claimed), createdAt, and updatedAt.
// Roles: api, admin, developer, billing, user.
// GET /agent/drafts
func (s *AgentsService) GetAgentReplyDrafts(ctx context.Context, params GetAgentReplyDraftsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"agentProfileId": params.AgentProfileID,
		"status":         params.Status,
		"page":           params.Page,
		"limit":          params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/agent/drafts"+qs, nil, opts)
}

// ApproveAgentReplyDraftData is the JSON body for approving a held agent reply.
type ApproveAgentReplyDraftData struct {
	// Text replaces the agent's draft when a person edited the reply (1 to 10000
	// characters). Empty sends the draft as the agent wrote it.
	Text string `json:"text,omitempty"`
}

// ApproveAgentReplyDraftParams are the parameters for ApproveAgentReplyDraft.
type ApproveAgentReplyDraftParams struct {
	// AgentReplyDraftID is the draft to approve. Required.
	AgentReplyDraftID string
	// ApprovalData carries the optional edited text.
	ApprovalData ApproveAgentReplyDraftData
}

// ApproveAgentReplyDraft approves a held agent reply and sends it from the agent's number,
// optionally with edited text. It goes through the normal SMS send path, so opt-out and
// billing apply; the draft comes back "failed" when the send path accepted nothing. A draft
// that is no longer pending returns 409.
// Roles: api, admin, developer, billing, user.
// POST /agent/drafts/{agentReplyDraftId}/approve
func (s *AgentsService) ApproveAgentReplyDraft(ctx context.Context, params ApproveAgentReplyDraftParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentReplyDraftId": params.AgentReplyDraftID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/drafts/%s/approve", url.PathEscape(params.AgentReplyDraftID))
	return s.client.doJSON(ctx, "POST", path, params.ApprovalData, opts)
}

// RejectAgentReplyDraftParams are the parameters for RejectAgentReplyDraft.
type RejectAgentReplyDraftParams struct {
	// AgentReplyDraftID is the draft to reject. Required.
	AgentReplyDraftID string
}

// RejectAgentReplyDraft rejects a held agent reply so it is never sent, and returns the
// rejected draft. A draft that is no longer pending returns 409.
// Roles: api, admin, developer, billing, user.
// POST /agent/drafts/{agentReplyDraftId}/reject
func (s *AgentsService) RejectAgentReplyDraft(ctx context.Context, params RejectAgentReplyDraftParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentReplyDraftId": params.AgentReplyDraftID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/drafts/%s/reject", url.PathEscape(params.AgentReplyDraftID))
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// ClaimAgentReplyDraftData is the JSON body for claiming a held agent reply.
type ClaimAgentReplyDraftData struct {
	// Text replaces the agent's draft when a person edited the reply (1 to 10000
	// characters). Empty claims the draft as the agent wrote it.
	Text string `json:"text,omitempty"`
}

// ClaimAgentReplyDraftParams are the parameters for ClaimAgentReplyDraft.
type ClaimAgentReplyDraftParams struct {
	// AgentReplyDraftID is the draft to claim. Required.
	AgentReplyDraftID string
	// ClaimData carries the optional edited text.
	ClaimData ClaimAgentReplyDraftData
}

// ClaimAgentReplyDraft claims a held agent reply for delivery through your own channel
// instead of Signal House's send path. Nothing is sent: the draft becomes "sending" with
// deliveredBy "external", sentText and a new claimId set, and you then report the outcome with
// CompleteAgentReplyDraft or ReleaseAgentReplyDraft, passing that claimId back. The claim
// means two approvals can never
// both deliver. A draft that is no longer pending returns 409; one whose number has left its
// subgroup returns 400 and is closed as failed.
// Roles: api, admin, developer, billing, user.
// POST /agent/drafts/{agentReplyDraftId}/claim
func (s *AgentsService) ClaimAgentReplyDraft(ctx context.Context, params ClaimAgentReplyDraftParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentReplyDraftId": params.AgentReplyDraftID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/drafts/%s/claim", url.PathEscape(params.AgentReplyDraftID))
	return s.client.doJSON(ctx, "POST", path, params.ClaimData, opts)
}

// CompleteAgentReplyDraftData is the JSON body for completing a claimed agent reply.
type CompleteAgentReplyDraftData struct {
	// ClaimID is the claim being completed, as returned by ClaimAgentReplyDraft. Required.
	ClaimID string `json:"claimId"`
	// ExternalMessageID is your channel's id for the sent message (up to 256 characters),
	// stored as sentMessageId. Empty omits it.
	ExternalMessageID string `json:"externalMessageId,omitempty"`
}

// CompleteAgentReplyDraftParams are the parameters for CompleteAgentReplyDraft.
type CompleteAgentReplyDraftParams struct {
	// AgentReplyDraftID is the claimed draft. Required.
	AgentReplyDraftID string
	// CompletionData carries the required claim id and the optional external message id.
	CompletionData CompleteAgentReplyDraftData
}

// CompleteAgentReplyDraft marks a claimed agent reply sent once your channel accepted it,
// and returns the sent draft. Only the claim named by ClaimID can be completed; anything else
// returns 409. It is also accepted after an unfinished claim was closed as "failed" with
// failureReason "delivery_unconfirmed", which it turns into "sent".
// Roles: api, admin, developer, billing, user.
// POST /agent/drafts/{agentReplyDraftId}/complete
func (s *AgentsService) CompleteAgentReplyDraft(ctx context.Context, params CompleteAgentReplyDraftParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentReplyDraftId": params.AgentReplyDraftID, "claimId": params.CompletionData.ClaimID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/drafts/%s/complete", url.PathEscape(params.AgentReplyDraftID))
	return s.client.doJSON(ctx, "POST", path, params.CompletionData, opts)
}

// ReleaseAgentReplyDraftData is the JSON body for releasing a claimed agent reply.
type ReleaseAgentReplyDraftData struct {
	// ClaimID is the claim being released, as returned by ClaimAgentReplyDraft. Required.
	ClaimID string `json:"claimId"`
	// Permanent, when true, closes the draft as failed instead of returning it to pending.
	// Nil omits it.
	Permanent *bool `json:"permanent,omitempty"`
}

// ReleaseAgentReplyDraftParams are the parameters for ReleaseAgentReplyDraft.
type ReleaseAgentReplyDraftParams struct {
	// AgentReplyDraftID is the claimed draft. Required.
	AgentReplyDraftID string
	// ReleaseData carries the required claim id and the optional permanent flag.
	ReleaseData ReleaseAgentReplyDraftData
}

// ReleaseAgentReplyDraft gives back a claim whose reply certainly did not go out: the draft
// returns to "pending", or with Permanent is closed as "failed". If you cannot tell whether
// it went out, do not release it; a claim left unfinished for 30 minutes is closed as failed
// with failureReason "delivery_unconfirmed". Only the claim named by ClaimID can be released;
// anything else returns 409.
// Roles: api, admin, developer, billing, user.
// POST /agent/drafts/{agentReplyDraftId}/release
func (s *AgentsService) ReleaseAgentReplyDraft(ctx context.Context, params ReleaseAgentReplyDraftParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentReplyDraftId": params.AgentReplyDraftID, "claimId": params.ReleaseData.ClaimID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/drafts/%s/release", url.PathEscape(params.AgentReplyDraftID))
	return s.client.doJSON(ctx, "POST", path, params.ReleaseData, opts)
}

// PublishAgentProfileParams are the parameters for PublishAgentProfile and
// UnpublishAgentProfile.
type PublishAgentProfileParams struct {
	AgentProfileID string
}

// PublishAgentProfile publishes an agent profile, setting publishStatus to "published" and
// stamping publishedAt. The server rejects the call with 400 when the agent has no enabled
// deployment, and with 409 when the agent is inactive.
// Roles: api, admin, developer.
// POST /agent/profiles/{agentProfileId}/publish
func (s *AgentsService) PublishAgentProfile(ctx context.Context, params PublishAgentProfileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/publish", url.PathEscape(params.AgentProfileID))
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// UnpublishAgentProfile returns an agent profile to draft (publishStatus "draft").
// Roles: api, admin, developer.
// POST /agent/profiles/{agentProfileId}/unpublish
func (s *AgentsService) UnpublishAgentProfile(ctx context.Context, params PublishAgentProfileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentProfileId": params.AgentProfileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/profiles/%s/unpublish", url.PathEscape(params.AgentProfileID))
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// BusinessHoursEntry is one day's open/close window in a tenant's business hours. Times are
// "HH:MM" (24h); Closed short-circuits open/close for that day.
type BusinessHoursEntry struct {
	// Day is the weekday: "monday".."sunday".
	Day string `json:"day"`
	// Closed, when true, marks the day fully closed (open/close ignored).
	Closed *bool `json:"closed,omitempty"`
	// Open is the local opening time "HH:MM".
	Open string `json:"open,omitempty"`
	// Close is the local closing time "HH:MM".
	Close string `json:"close,omitempty"`
}

// UpsertTenantAiSettingsData is the JSON body for an upsert of a scope's tenant AI settings.
// Optional scalar fields use pointers so a zero value can be sent explicitly rather than omitted.
type UpsertTenantAiSettingsData struct {
	// BrandName is the brand/business name the agent represents.
	BrandName *string `json:"brandName,omitempty"`
	// Tone is a freeform tone descriptor (e.g. "professional", "friendly").
	Tone *string `json:"tone,omitempty"`
	// Timezone is the IANA timezone business hours are evaluated in.
	Timezone *string `json:"timezone,omitempty"`
	// BusinessHours is the per-day open/close windows. nil omits the field (leave unchanged);
	// &[]BusinessHoursEntry{} sends [] to clear all hours (the agent is then always open).
	BusinessHours *[]BusinessHoursEntry `json:"businessHours,omitempty"`
	// AfterHoursMessage is the message used outside business hours.
	AfterHoursMessage *string `json:"afterHoursMessage,omitempty"`
}

// GetTenantAiSettingsParams are the parameters for GetTenantAiSettings.
type GetTenantAiSettingsParams struct {
	GroupID    string
	SubgroupID string
}

// GetTenantAiSettings reads the tenant AI settings for a scope (group, or a subgroup within
// it). Returns the scope's own row only — no group-level fallback.
// Roles: api, admin, developer, billing, user.
// GET /agent/tenant-settings
func (s *AgentsService) GetTenantAiSettings(ctx context.Context, params GetTenantAiSettingsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": params.GroupID, "subgroupId": params.SubgroupID})
	return s.client.doJSON(ctx, "GET", "/agent/tenant-settings"+qs, nil, opts)
}

// UpsertTenantAiSettingsParams are the parameters for UpsertTenantAiSettings.
type UpsertTenantAiSettingsParams struct {
	GroupID      string
	SubgroupID   string
	SettingsData UpsertTenantAiSettingsData
}

// UpsertTenantAiSettings creates or updates the tenant AI settings for a scope (one row per
// group/subgroup; the scope is passed as query params).
// Roles: api, admin, developer.
// PUT /agent/tenant-settings
func (s *AgentsService) UpsertTenantAiSettings(ctx context.Context, params UpsertTenantAiSettingsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": params.GroupID, "subgroupId": params.SubgroupID})
	return s.client.doJSON(ctx, "PUT", "/agent/tenant-settings"+qs, params.SettingsData, opts)
}

// DeleteTenantAiSettingsParams are the parameters for DeleteTenantAiSettings.
type DeleteTenantAiSettingsParams struct {
	GroupID    string
	SubgroupID string
}

// DeleteTenantAiSettings deletes the tenant AI settings for a scope.
// Roles: api, admin, developer.
// DELETE /agent/tenant-settings
func (s *AgentsService) DeleteTenantAiSettings(ctx context.Context, params DeleteTenantAiSettingsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": params.GroupID, "subgroupId": params.SubgroupID})
	return s.client.doJSON(ctx, "DELETE", "/agent/tenant-settings"+qs, nil, opts)
}

// CreateKnowledgeItemData is the JSON body for creating a knowledge item. GroupID and Text
// are required; the raw Text is stored and Atlas Vector Search owns the embedding.
type CreateKnowledgeItemData struct {
	// GroupID is the owning group (starts with 'G'). Required.
	GroupID string `json:"groupId"`
	// SubgroupID scopes the item to a subgroup (starts with 'S'); nil = group-level.
	SubgroupID *string `json:"subgroupId,omitempty"`
	// AgentProfileID restricts the item to one agent; nil = shared across the tenant's agents.
	AgentProfileID *string `json:"agentProfileId,omitempty"`
	// Title is a short human label.
	Title string `json:"title,omitempty"`
	// Text is the raw content Atlas auto-embeds and retrieves against. Required.
	Text string `json:"text"`
	// Source is how the item was added: "manual", "url", "document", or "import".
	Source string `json:"source,omitempty"`
	// Enabled controls whether the item is included in retrieval.
	Enabled *bool `json:"enabled,omitempty"`
	// Metadata is free-form per-item metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// UpdateKnowledgeItemData is the JSON body for updating a knowledge item. Every field is
// optional; the item's scope (group/subgroup/agent) is immutable and absent here.
type UpdateKnowledgeItemData struct {
	// Title is a short human label.
	Title *string `json:"title,omitempty"`
	// Text is the raw content Atlas auto-embeds and retrieves against.
	Text *string `json:"text,omitempty"`
	// Source is how the item was added: "manual", "url", "document", or "import".
	Source *string `json:"source,omitempty"`
	// Enabled controls whether the item is included in retrieval.
	Enabled *bool `json:"enabled,omitempty"`
	// Metadata is free-form per-item metadata.
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// GetKnowledgeItemsParams are the parameters for GetKnowledgeItems.
type GetKnowledgeItemsParams struct {
	// GroupID is the group whose knowledge to list. Required.
	GroupID string
	// SubgroupID narrows the results to a subgroup.
	SubgroupID string
	// AgentProfileID narrows the results to one agent.
	AgentProfileID string
	// Page is the page number (1-based) for pagination.
	Page string
	// Limit is the items per page (max 100; server default 25).
	Limit string
}

// GetKnowledgeItems lists the knowledge-base items an agent can answer from (RAG), scoped to
// a group and optionally narrowed to a subgroup and/or a single agent. Results are paginated
// (server default 25 per page, max 100).
// Roles: api, admin, developer, billing, user.
// GET /agent/knowledge
func (s *AgentsService) GetKnowledgeItems(ctx context.Context, params GetKnowledgeItemsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"agentProfileId": params.AgentProfileID,
		"page":           params.Page,
		"limit":          params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/agent/knowledge"+qs, nil, opts)
}

// GetKnowledgeItemParams are the parameters for GetKnowledgeItem.
type GetKnowledgeItemParams struct {
	KnowledgeItemID string
}

// GetKnowledgeItem returns a single knowledge item by id.
// Roles: api, admin, developer, billing, user.
// GET /agent/knowledge/{knowledgeItemId}
func (s *AgentsService) GetKnowledgeItem(ctx context.Context, params GetKnowledgeItemParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"knowledgeItemId": params.KnowledgeItemID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/knowledge/%s", url.PathEscape(params.KnowledgeItemID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// CreateKnowledgeItemParams are the parameters for CreateKnowledgeItem.
type CreateKnowledgeItemParams struct {
	ItemData CreateKnowledgeItemData
}

// CreateKnowledgeItem creates a knowledge item (raw text stored; Atlas owns the embedding).
// Roles: api, admin, developer.
// POST /agent/knowledge
func (s *AgentsService) CreateKnowledgeItem(ctx context.Context, params CreateKnowledgeItemParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/agent/knowledge", params.ItemData, opts)
}

// UpdateKnowledgeItemParams are the parameters for UpdateKnowledgeItem.
type UpdateKnowledgeItemParams struct {
	ID         string
	UpdateData UpdateKnowledgeItemData
}

// UpdateKnowledgeItem updates a knowledge item. The item's scope is immutable.
// Roles: api, admin, developer.
// PUT /agent/knowledge/{id}
func (s *AgentsService) UpdateKnowledgeItem(ctx context.Context, params UpdateKnowledgeItemParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/knowledge/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// DeleteKnowledgeItemParams are the parameters for DeleteKnowledgeItem.
type DeleteKnowledgeItemParams struct {
	ID string
}

// DeleteKnowledgeItem deletes a knowledge item by its id.
// Roles: api, admin, developer.
// DELETE /agent/knowledge/{id}
func (s *AgentsService) DeleteKnowledgeItem(ctx context.Context, params DeleteKnowledgeItemParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/knowledge/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// CreateAgentToolData is the JSON body for creating an agent tool. GroupID, Name, and
// Description are required; type-specific fields (WebhookURL / BuiltinKey / MCPServerURL) are
// required by the server for their respective Type. A webhook WebhookURL must be a public https URL.
type CreateAgentToolData struct {
	// GroupID is the owning group (starts with 'G'). Required.
	GroupID string `json:"groupId"`
	// SubgroupID scopes the tool to a subgroup (starts with 'S'); nil = group-level.
	SubgroupID *string `json:"subgroupId,omitempty"`
	// AgentProfileID restricts the tool to one agent; nil = shared across the tenant's agents.
	AgentProfileID *string `json:"agentProfileId,omitempty"`
	// Name is the tool name exposed to the LLM. Required.
	Name string `json:"name"`
	// Description tells the LLM what the tool does. Required.
	Description string `json:"description"`
	// Type is "builtin", "webhook", or "mcp" (defaults to "builtin").
	Type string `json:"type,omitempty"`
	// BuiltinKey is the built-in this configures (for Type "builtin").
	BuiltinKey *string `json:"builtinKey,omitempty"`
	// ParametersSchema is the JSON Schema for the tool's parameters.
	ParametersSchema map[string]interface{} `json:"parametersSchema,omitempty"`
	// Config is per-tool configuration.
	Config map[string]interface{} `json:"config,omitempty"`
	// WebhookURL is the customer endpoint the call is relayed to (for Type "webhook"); must be public https.
	WebhookURL *string `json:"webhookUrl,omitempty"`
	// WebhookAuth is auth config for the webhook call; stored, never echoed back. It MUST be of the
	// shape {"headers"?: {name: value}, "signingSecret"?: string} — any other shape (e.g.
	// {"header","value"}) is rejected with a 400. "headers" are sent verbatim on every call
	// (reserved/framing and x-signalhouse-* names are rejected); "signingSecret" makes us send
	// X-SignalHouse-Signature: sha256=HMAC-SHA256(`${X-SignalHouse-Timestamp}.${rawBody}`) so the
	// endpoint can verify the call and reject replays.
	WebhookAuth map[string]interface{} `json:"webhookAuth,omitempty"`
	// MCPServerURL is the MCP server URL (for Type "mcp").
	MCPServerURL *string `json:"mcpServerUrl,omitempty"`
	// Channels are the channels this tool is offered on (subset of the agent channels).
	Channels []string `json:"channels,omitempty"`
	// Enabled controls whether the tool is available.
	Enabled *bool `json:"enabled,omitempty"`
}

// UpdateAgentToolData is the JSON body for updating an agent tool. Every field is optional;
// the tool's scope (group/subgroup/agent) is immutable and absent here.
type UpdateAgentToolData struct {
	Name             *string                `json:"name,omitempty"`
	Description      *string                `json:"description,omitempty"`
	Type             *string                `json:"type,omitempty"`
	BuiltinKey       *string                `json:"builtinKey,omitempty"`
	ParametersSchema map[string]interface{} `json:"parametersSchema,omitempty"`
	Config           map[string]interface{} `json:"config,omitempty"`
	WebhookURL       *string                `json:"webhookUrl,omitempty"`
	// WebhookAuth MUST be {"headers"?: {name: value}, "signingSecret"?: string}; other shapes 400.
	// See CreateAgentToolData.WebhookAuth for the header constraints and signing scheme.
	WebhookAuth  map[string]interface{} `json:"webhookAuth,omitempty"`
	MCPServerURL *string                `json:"mcpServerUrl,omitempty"`
	Channels     []string               `json:"channels,omitempty"`
	Enabled      *bool                  `json:"enabled,omitempty"`
}

// GetAgentToolsParams are the parameters for GetAgentTools.
type GetAgentToolsParams struct {
	// GroupID is the group whose tools to list. Required.
	GroupID string
	// SubgroupID narrows the results to a subgroup.
	SubgroupID string
	// AgentProfileID narrows the results to one agent.
	AgentProfileID string
	// Type narrows the results to a tool type ("builtin", "webhook", "mcp").
	Type string
	// Page is the page number (1-based) for pagination.
	Page string
	// Limit is the items per page (max 100; server default 25).
	Limit string
}

// GetAgentTools lists the tools defined under a group (webhook / configured builtin / mcp),
// with webhookAuth secrets redacted. Paginated (server default 25 per page, max 100).
// Roles: api, admin, developer, billing, user.
// GET /agent/tools
func (s *AgentsService) GetAgentTools(ctx context.Context, params GetAgentToolsParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"agentProfileId": params.AgentProfileID,
		"type":           params.Type,
		"page":           params.Page,
		"limit":          params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/agent/tools"+qs, nil, opts)
}

// GetAgentToolParams are the parameters for GetAgentTool.
type GetAgentToolParams struct {
	AgentToolID string
}

// GetAgentTool returns a single agent tool by id (webhookAuth redacted).
// Roles: api, admin, developer, billing, user.
// GET /agent/tools/{agentToolId}
func (s *AgentsService) GetAgentTool(ctx context.Context, params GetAgentToolParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"agentToolId": params.AgentToolID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/tools/%s", url.PathEscape(params.AgentToolID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// CreateAgentToolParams are the parameters for CreateAgentTool.
type CreateAgentToolParams struct {
	ToolData CreateAgentToolData
}

// CreateAgentTool creates an agent tool. A webhook tool's WebhookURL must be a public https URL.
// Roles: api, admin, developer.
// POST /agent/tools
func (s *AgentsService) CreateAgentTool(ctx context.Context, params CreateAgentToolParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/agent/tools", params.ToolData, opts)
}

// UpdateAgentToolParams are the parameters for UpdateAgentTool.
type UpdateAgentToolParams struct {
	ID         string
	UpdateData UpdateAgentToolData
}

// UpdateAgentTool updates an agent tool. The tool's scope is immutable.
// Roles: api, admin, developer.
// PUT /agent/tools/{id}
func (s *AgentsService) UpdateAgentTool(ctx context.Context, params UpdateAgentToolParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/tools/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// DeleteAgentToolParams are the parameters for DeleteAgentTool.
type DeleteAgentToolParams struct {
	ID string
}

// DeleteAgentTool deletes an agent tool by its id.
// Roles: api, admin, developer.
// DELETE /agent/tools/{id}
func (s *AgentsService) DeleteAgentTool(ctx context.Context, params DeleteAgentToolParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/agent/tools/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}
