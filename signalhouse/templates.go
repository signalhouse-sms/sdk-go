package signalhouse

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
	"strconv"
)

// TemplatesService handles versioned message templates under /template. Every call
// is 404 while the API's RCS kill switch is off.
// Writes apply directly (last write wins); 409 codes are TEMPLATE_ARCHIVED, TEMPLATE_NAME_TAKEN, NO_DRAFT (publish
// with no draft) and DRAFT_CHANGED (the draft changed while it was being published; publish again).
type TemplatesService struct {
	client *Client
}

// CreateTemplateParams are the parameters for the Create method.
type CreateTemplateParams struct {
	GroupID string `json:"groupId"`
	// Name is 1-120 characters, unique per group and channel.
	Name string `json:"name"`
	// Tag is 1-60 characters; empty omits it.
	Tag string `json:"tag,omitempty"`
	// Body is the raw RcsTemplateBody (channel "rcs", contentFormat "rcs.blocks.v1", content.blocks, variables,
	// defaults, fallback), e.g. a json.RawMessage or map[string]any. It is sent verbatim.
	Body any `json:"body"`
}

// Create creates a template with draft version 1 and returns {head, version} (HTTP 201).
func (s *TemplatesService) Create(ctx context.Context, params CreateTemplateParams, opts *RequestOptions) (*Response, error) {
	if err := checkCreateTemplate(params); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/template", params, opts)
}

// GetTemplatesParams are the parameters for the List method.
type GetTemplatesParams struct {
	GroupID string
	// State is DRAFT, PUBLISHED or ARCHIVED.
	State string
	// Channel is "rcs".
	Channel string
	Tag     string
	Search  string
	// SortBy is name, createdAt or updatedAt.
	SortBy string
	// SortOrder is asc or desc.
	SortOrder string
	// Page defaults to 1.
	Page string
	// Limit defaults to 25, max 100.
	Limit string
}

// List lists template summaries as {data, total, page, limit}.
func (s *TemplatesService) List(ctx context.Context, params GetTemplatesParams, opts *RequestOptions) (*Response, error) {
	qs, err := templateListQuery(params)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/template"+qs, nil, opts)
}

// GetTemplateParams are the parameters for the Get method.
type GetTemplateParams struct {
	TemplateID string
	GroupID    string
	// Version is a positive integer, "draft" or "published". Empty returns the published version, else the draft.
	Version string
}

// Get returns {head, version} for one template version; an explicit Version never falls back (404).
func (s *TemplatesService) Get(ctx context.Context, params GetTemplateParams, opts *RequestOptions) (*Response, error) {
	qs, err := templateVersionQuery(params.TemplateID, params.GroupID, params.Version)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "GET", path+qs, nil, opts)
}

// UpdateTemplateParams are the parameters for the Update method.
type UpdateTemplateParams struct {
	TemplateID string
	GroupID    string
	// Name is omitted when empty.
	Name string
	// Tag sets the tag; nil leaves it unchanged unless ClearTag is set.
	Tag *string
	// ClearTag sends tag: null to clear it.
	ClearTag bool
	// Body is a full replacement RcsTemplateBody, sent verbatim; nil leaves it unchanged.
	Body any
}

// Update edits the draft (creating one from the published version if needed) and returns {head, version}.
func (s *TemplatesService) Update(ctx context.Context, params UpdateTemplateParams, opts *RequestOptions) (*Response, error) {
	body, err := templateUpdateBody(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "PUT", path, body, opts)
}

// PublishTemplateParams are the parameters for the Publish method.
type PublishTemplateParams struct {
	TemplateID string
	GroupID    string
}

// Publish publishes the current draft and returns {head, version}. An incomplete draft is a 400
// TEMPLATE_INVALID carrying errors and warnings; no draft is a 409 NO_DRAFT, and a draft that changed while it
// was being published is a 409 DRAFT_CHANGED (publish again).
func (s *TemplatesService) Publish(ctx context.Context, params PublishTemplateParams, opts *RequestOptions) (*Response, error) {
	body, err := templatePublishBody(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s/publish", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// ArchiveTemplateParams are the parameters for the Archive method.
type ArchiveTemplateParams struct {
	TemplateID string
	GroupID    string
}

// Archive archives a template and returns its head.
func (s *TemplatesService) Archive(ctx context.Context, params ArchiveTemplateParams, opts *RequestOptions) (*Response, error) {
	body, err := templateArchiveBody(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s/archive", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// GetVersionsParams are the parameters for the Versions method.
type GetVersionsParams struct {
	TemplateID string
	GroupID    string
	Page       string
	Limit      string
}

// Versions lists a template's version summaries (no bodies) as {data, total, page, limit}.
func (s *TemplatesService) Versions(ctx context.Context, params GetVersionsParams, opts *RequestOptions) (*Response, error) {
	qs, err := templateVersionsQuery(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s/versions", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "GET", path+qs, nil, opts)
}

// ValidateTemplateParams are the parameters for the Validate method. Set exactly one of Body or Version.
type ValidateTemplateParams struct {
	TemplateID string
	GroupID    string
	// Body is a raw RcsTemplateBody to validate, sent verbatim.
	Body any
	// Version is a positive integer (sent as a JSON number), "draft" or "published".
	Version string
}

// Validate validates a body or a stored version without writing and returns {valid, contentHash,
// validatorVersion, errors, warnings}.
func (s *TemplatesService) Validate(ctx context.Context, params ValidateTemplateParams, opts *RequestOptions) (*Response, error) {
	body, err := templateValidateBody(params)
	if err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/template/%s/validate", url.PathEscape(params.TemplateID))
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// GetCapabilitiesParams are the parameters for the Capabilities method.
type GetCapabilitiesParams struct {
	GroupID string
	// Channel is "rcs" (the server default).
	Channel string
	// Provider is "infobip" or "sinch"; empty returns provider: null.
	Provider string
}

// Capabilities returns the block kinds, action types, limits and media rules the validator enforces.
func (s *TemplatesService) Capabilities(ctx context.Context, params GetCapabilitiesParams, opts *RequestOptions) (*Response, error) {
	qs, err := templateCapabilitiesQuery(params.GroupID, params.Channel, params.Provider)
	if err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "GET", "/template/capabilities"+qs, nil, opts)
}

// UploadMediaParams are the parameters for the UploadMedia method.
type UploadMediaParams struct {
	GroupID string
	File    *FileUpload
}

// UploadMedia uploads one template media file and returns the MediaAsset with its public URL (HTTP 201).
func (s *TemplatesService) UploadMedia(ctx context.Context, params UploadMediaParams, opts *RequestOptions) (*Response, error) {
	writeForm, err := templateMediaForm(params)
	if err != nil {
		return nil, err
	}
	return s.client.doMultipart(ctx, "POST", "/template/media", writeForm, opts)
}

// requireTemplate validates the identifiers every per-template call needs.
func requireTemplate(templateID, groupID string) error {
	return require(map[string]string{"templateId": templateID, "groupId": groupID})
}

// checkCreateTemplate validates a create request.
func checkCreateTemplate(params CreateTemplateParams) error {
	if err := require(map[string]string{"groupId": params.GroupID, "name": params.Name}); err != nil {
		return err
	}
	return requireNonNil(map[string]interface{}{"body": params.Body})
}

// templateListQuery builds the list query string; empty filters are dropped.
func templateListQuery(params GetTemplatesParams) (string, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return "", err
	}
	return buildQueryString(map[string]string{
		"groupId":   params.GroupID,
		"state":     params.State,
		"channel":   params.Channel,
		"tag":       params.Tag,
		"search":    params.Search,
		"sortBy":    params.SortBy,
		"sortOrder": params.SortOrder,
		"page":      params.Page,
		"limit":     params.Limit,
	}), nil
}

// templateVersionQuery builds the groupId + version selector query for a per-template read.
func templateVersionQuery(templateID, groupID, version string) (string, error) {
	if err := requireTemplate(templateID, groupID); err != nil {
		return "", err
	}
	return buildQueryString(map[string]string{"groupId": groupID, "version": version}), nil
}

// templateVersionsQuery builds the version-list query string.
func templateVersionsQuery(params GetVersionsParams) (string, error) {
	if err := requireTemplate(params.TemplateID, params.GroupID); err != nil {
		return "", err
	}
	return buildQueryString(map[string]string{"groupId": params.GroupID, "page": params.Page, "limit": params.Limit}), nil
}

// templateUpdateBody builds the update body; unchanged fields are omitted.
func templateUpdateBody(params UpdateTemplateParams) (map[string]any, error) {
	if err := requireTemplate(params.TemplateID, params.GroupID); err != nil {
		return nil, err
	}
	body := map[string]any{"groupId": params.GroupID}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if params.ClearTag {
		body["tag"] = nil
	} else if params.Tag != nil {
		body["tag"] = *params.Tag
	}
	if params.Body != nil {
		body["body"] = params.Body
	}
	return body, nil
}

// templatePublishBody builds the publish body.
func templatePublishBody(params PublishTemplateParams) (map[string]any, error) {
	if err := requireTemplate(params.TemplateID, params.GroupID); err != nil {
		return nil, err
	}
	return map[string]any{"groupId": params.GroupID}, nil
}

// templateArchiveBody builds the archive body.
func templateArchiveBody(params ArchiveTemplateParams) (map[string]any, error) {
	if err := requireTemplate(params.TemplateID, params.GroupID); err != nil {
		return nil, err
	}
	return map[string]any{"groupId": params.GroupID}, nil
}

// templateValidateBody builds the validate body from exactly one of a body or a version selector.
func templateValidateBody(params ValidateTemplateParams) (map[string]any, error) {
	if err := requireTemplate(params.TemplateID, params.GroupID); err != nil {
		return nil, err
	}
	if (params.Body == nil) == (params.Version == "") {
		return nil, &ValidationError{Field: "exactly one of body or version"}
	}
	body := map[string]any{"groupId": params.GroupID}
	if params.Body != nil {
		body["body"] = params.Body
	} else if n, err := strconv.Atoi(params.Version); err == nil {
		body["version"] = n
	} else {
		body["version"] = params.Version
	}
	return body, nil
}

// templateCapabilitiesQuery builds the capabilities query string.
func templateCapabilitiesQuery(groupID, channel, provider string) (string, error) {
	if err := require(map[string]string{"groupId": groupID}); err != nil {
		return "", err
	}
	return buildQueryString(map[string]string{"groupId": groupID, "channel": channel, "provider": provider}), nil
}

// templateMediaForm returns a writer for the multipart file + groupId upload form.
func templateMediaForm(params UploadMediaParams) (func(w *multipart.Writer) error, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	if params.File == nil || params.File.Reader == nil {
		return nil, &ValidationError{Field: "file"}
	}
	return func(w *multipart.Writer) error {
		part, err := w.CreateFormFile("file", params.File.Filename)
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, params.File.Reader); err != nil {
			return err
		}
		return w.WriteField("groupId", params.GroupID)
	}, nil
}
