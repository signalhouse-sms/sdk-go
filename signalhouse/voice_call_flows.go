package signalhouse

import (
	"context"
	"net/url"
)

// VoiceCallFlowsService handles call flow API calls.
//
// A call flow is a customer-built call tree (greeting, menu, forward, voicemail,
// hang up) that runs when a number's route, or its Programmable Voice Profile,
// names it with RouteAction "CALL_FLOW". Each flow has one working draft, edited
// with SaveCallFlowDraft, and published versions: PublishCallFlow validates the
// draft and makes it the version calls run, while the draft stays editable
// without affecting calls.
type VoiceCallFlowsService struct {
	client *Client
}

// ListCallFlowsParams filters ListCallFlows. All fields are optional; Page and
// Limit are strings to match this SDK's query-param convention.
type ListCallFlowsParams struct {
	// Page is the page number (1-indexed) as a string. Defaults server-side to "1".
	Page string
	// Limit is the page size (1-100) as a string. Defaults server-side to "20".
	Limit string
	// Q matches against the flow name.
	Q string
	// SubgroupID returns only flows in this subgroup.
	SubgroupID string
	// IncludeDeleted, when "true", includes deleted flows.
	IncludeDeleted string
	// SortBy is the sort field: createdAt (default), updatedAt or name.
	SortBy string
	// SortOrder is the sort direction: asc or desc (default desc).
	SortOrder string
}

// CreateCallFlowParams is the body for CreateCallFlow. Both fields are required.
type CreateCallFlowParams struct {
	// Name is the flow name, unique within its subgroup.
	Name string `json:"name"`
	// SubgroupID is the subgroup the flow belongs to. It is fixed once created;
	// only numbers and Programmable Voice Profiles in it can run the flow.
	SubgroupID string `json:"subgroupId"`
}

// UpdateCallFlowParams is the body for UpdateCallFlow. All fields are optional.
type UpdateCallFlowParams struct {
	// Name renames the flow, unique within its subgroup.
	Name string `json:"name,omitempty"`
	// Status is "ACTIVE" or "DISABLED". A disabled flow does not run; numbers
	// routed to it fall through to their next routing layer.
	Status string `json:"status,omitempty"`
}

// CallFlowStep is one step of a call flow draft.
type CallFlowStep struct {
	// ID is a UUID you generate. It is kept as the step's id across saves.
	ID string `json:"id"`
	// Type is the step kind: "inbound", "answer", "play", "ivr", "fwd_phone",
	// "voicemail" or "hangup". GetCallFlowCatalog lists the kinds this service runs.
	Type string `json:"type"`
	// Name is the display name.
	Name string `json:"name"`
	// Config holds the step's settings, which depend on its type.
	Config map[string]interface{} `json:"config,omitempty"`
	// PositionX is the canvas x position, for a visual builder.
	PositionX int `json:"positionX"`
	// PositionY is the canvas y position, for a visual builder.
	PositionY int `json:"positionY"`
}

// CallFlowConnection connects one step's outcome to the next step.
type CallFlowConnection struct {
	// ID is a UUID you generate.
	ID string `json:"id"`
	// FromNodeID is the step the connection leaves.
	FromNodeID string `json:"fromNodeId"`
	// ToNodeID is the step it leads to.
	ToNodeID string `json:"toNodeId"`
	// Port is the source step's outcome, e.g. "next", "success", "no_input" or
	// "opt_<optionId>".
	Port string `json:"port"`
}

// CallFlowDraft is the body for SaveCallFlowDraft: every step and connection.
type CallFlowDraft struct {
	// Nodes is every step, including exactly one "inbound" step.
	Nodes []CallFlowStep `json:"nodes"`
	// Edges is every connection between steps.
	Edges []CallFlowConnection `json:"edges"`
}

// emptyCallFlowBody is sent where a call-flow endpoint takes no input. The client
// always declares JSON, and the Voice service rejects a JSON request with no body.
var emptyCallFlowBody = map[string]interface{}{}

// ListCallFlows returns the account's call flows, newest first by default.
// Returns { data: [...], meta: { total, page, limit, totalPages } }.
// GET /voice/api/v1/call-flows
func (s *VoiceCallFlowsService) ListCallFlows(ctx context.Context, params ListCallFlowsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"page":           params.Page,
		"limit":          params.Limit,
		"q":              params.Q,
		"subgroupId":     params.SubgroupID,
		"includeDeleted": params.IncludeDeleted,
		"sortBy":         params.SortBy,
		"sortOrder":      params.SortOrder,
	})
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/call-flows"+qs, nil, opts)
}

// GetCallFlowCatalog returns the step types this service can run, with the
// outcomes each one offers.
// Returns { data: { nodeTypes: [{ type, ports, configurable }] } }.
// GET /voice/api/v1/call-flows/catalog
func (s *VoiceCallFlowsService) GetCallFlowCatalog(ctx context.Context, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/call-flows/catalog", nil, opts)
}

// GetCallFlow returns a call flow with its working draft and the issues that
// stop the draft being published.
// Returns { data: { ...flow, draft: { versionId, nodes, edges, issues } } }.
// GET /voice/api/v1/call-flows/:id
func (s *VoiceCallFlowsService) GetCallFlow(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/voice/api/v1/call-flows/"+url.PathEscape(id), nil, opts)
}

// CreateCallFlow creates a call flow, with its working draft and the Inbound
// Call step every flow starts from. Returns { data: { ...flow, draft } }.
// POST /voice/api/v1/call-flows
func (s *VoiceCallFlowsService) CreateCallFlow(ctx context.Context, params CreateCallFlowParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"name": params.Name, "subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/call-flows", params, opts)
}

// UpdateCallFlow renames a call flow, or enables or disables it (partial
// update). Returns { data: { ...flow, draft } }.
// PATCH /voice/api/v1/call-flows/:id
func (s *VoiceCallFlowsService) UpdateCallFlow(ctx context.Context, id string, params UpdateCallFlowParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PATCH", "/voice/api/v1/call-flows/"+url.PathEscape(id), params, opts)
}

// DeleteCallFlow deletes a call flow. Reversible with RestoreCallFlow: while
// deleted, numbers and profiles routed to it fall through to their next routing
// layer. Returns { data: { ...flow } }.
// DELETE /voice/api/v1/call-flows/:id
func (s *VoiceCallFlowsService) DeleteCallFlow(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "DELETE", "/voice/api/v1/call-flows/"+url.PathEscape(id), nil, opts)
}

// RestoreCallFlow restores a deleted call flow. Returns { data: { ...flow, draft } }.
// POST /voice/api/v1/call-flows/:id/restore
func (s *VoiceCallFlowsService) RestoreCallFlow(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/call-flows/"+url.PathEscape(id)+"/restore", emptyCallFlowBody, opts)
}

// DuplicateCallFlow copies a call flow and its working draft into a new flow in
// the same subgroup. The copy is not published. An empty name defaults to
// "<name> (copy)". Returns { data: { ...flow, draft } }.
// POST /voice/api/v1/call-flows/:id/duplicate
func (s *VoiceCallFlowsService) DuplicateCallFlow(ctx context.Context, id string, name string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	body := map[string]interface{}{}
	if name != "" {
		body["name"] = name
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/call-flows/"+url.PathEscape(id)+"/duplicate", body, opts)
}

// SaveCallFlowDraft replaces the working draft. The whole draft is saved even
// when it is not finished; the issues that would stop it being published come
// back in the response. Returns { data: { nodes, edges, issues } }.
// PUT /voice/api/v1/call-flows/:id/draft
func (s *VoiceCallFlowsService) SaveCallFlowDraft(ctx context.Context, id string, draft CallFlowDraft, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	// A nil slice marshals to null; the service expects arrays.
	if draft.Nodes == nil {
		draft.Nodes = []CallFlowStep{}
	}
	if draft.Edges == nil {
		draft.Edges = []CallFlowConnection{}
	}
	return s.client.doJSON(ctx, "PUT", "/voice/api/v1/call-flows/"+url.PathEscape(id)+"/draft", draft, opts)
}

// PublishCallFlow publishes the working draft, making it the version calls run.
// A draft that is not ready returns 400 with error.issues listing each step that
// needs attention. Returns { data: { id, versionNumber, status, publishedAt } }.
// POST /voice/api/v1/call-flows/:id/publish
func (s *VoiceCallFlowsService) PublishCallFlow(ctx context.Context, id string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": id}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/voice/api/v1/call-flows/"+url.PathEscape(id)+"/publish", emptyCallFlowBody, opts)
}
