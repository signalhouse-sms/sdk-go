package signalhouse

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// ComplianceService handles the compliance cases under /compliance. Signal House opens a case when a carrier
// suspends one of your objects, a carrier escalates a message, or Signal House needs something from you. You read
// your cases, answer their requirements and upload files; only Signal House opens, works and closes them.
type ComplianceService struct {
	client *Client
}

// GetCasesParams are the parameters for the GetCases method. Empty fields are left out of the query.
type GetCasesParams struct {
	// GroupID is a group you can view; the API defaults to your active group.
	GroupID string
	// Status is OPEN, LIFT_REQUESTED or CLOSED.
	Status string
	// Stage is new, customer, ready, lift_requested or closed.
	Stage string
	// ObjectType is CAMPAIGN, BRAND, NUMBER, RCS_AGENT or SUBGROUP.
	ObjectType string
	// ObjectID is the object's ID.
	ObjectID string
	// SubgroupID returns only cases in this subgroup.
	SubgroupID string
	// Suspended set to true returns cases with an open suspension or hold, and to false the rest. nil omits it.
	Suspended *bool
	// From returns cases opened on or after this ISO 8601 time.
	From string
	// To returns cases opened on or before this ISO 8601 time.
	To string
	// Sort is createdAt (the API default), fixBy or severity.
	Sort string
	// Page is the page number, from 1.
	Page string
	// Limit is the cases per page, 1-200.
	Limit string
}

// GetCases lists the compliance cases you can see, newest first, as
// {data, total, page, limit, totalPages, counts: {open, waitingOnYou}}. counts.waitingOnYou is the open cases with
// requirements still to answer. Roles: api, admin, developer, billing, user.
func (s *ComplianceService) GetCases(ctx context.Context, params GetCasesParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":    params.GroupID,
		"status":     params.Status,
		"stage":      params.Stage,
		"objectType": params.ObjectType,
		"objectId":   params.ObjectID,
		"subgroupId": params.SubgroupID,
		"suspended":  boolQueryValue(params.Suspended),
		"from":       params.From,
		"to":         params.To,
		"sort":       params.Sort,
		"page":       params.Page,
		"limit":      params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/compliance/cases"+qs, nil, opts)
}

// GetCaseParams are the parameters for the GetCase method.
type GetCaseParams struct {
	// CaseID is the case ID (e.g. "CASE-…"). Required.
	CaseID string
}

// GetCase returns one compliance case: its stage, suspensions, tickets and requirements.
// Roles: api, admin, developer, billing, user.
func (s *ComplianceService) GetCase(ctx context.Context, params GetCaseParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"caseId": params.CaseID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/compliance/cases/%s", url.PathEscape(params.CaseID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// AnswerRequirementParams are the parameters for the AnswerRequirement method.
type AnswerRequirementParams struct {
	// CaseID is the case ID (e.g. "CASE-…"). Required.
	CaseID string
	// RequirementID is the requirement ID (e.g. "CASE-…-n"). Required.
	RequirementID string
	// Answer says what you did, up to 2,000 characters. Empty omits it.
	Answer string
	// FileIDs lists up to 10 fileIds from UploadFile, on this case. nil omits it.
	FileIDs []string
}

// AnswerRequirement answers one requirement with a note, files, or both, and returns the case. A RECORDS
// requirement needs a file or a note. Roles: api, admin, developer, user.
func (s *ComplianceService) AnswerRequirement(ctx context.Context, params AnswerRequirementParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"caseId": params.CaseID, "requirementId": params.RequirementID}); err != nil {
		return nil, err
	}
	body := map[string]any{}
	if params.Answer != "" {
		body["answer"] = params.Answer
	}
	if params.FileIDs != nil {
		body["fileIds"] = params.FileIDs
	}
	path := fmt.Sprintf("/compliance/cases/%s/requirements/%s", url.PathEscape(params.CaseID), url.PathEscape(params.RequirementID))
	return s.client.doJSON(ctx, "PUT", path, body, opts)
}

// UploadFileParams are the parameters for the UploadFile method.
type UploadFileParams struct {
	// CaseID is the case the file answers. Required.
	CaseID string
	// RequirementID is the requirement the file answers. Empty omits it.
	RequirementID string
	// File is the file to upload. Required.
	File *FileUpload
}

// UploadFile uploads a file to a compliance case (CSV, XLSX, PDF, PNG, JPG or TXT, up to 10 MB). Pass the
// returned fileId in FileIDs when you answer the requirement. Roles: api, admin, developer, user.
//
// Response: { fileId, fileName, mimeType, size }.
func (s *ComplianceService) UploadFile(ctx context.Context, params UploadFileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"caseId": params.CaseID}); err != nil {
		return nil, err
	}
	if params.File == nil || params.File.Reader == nil {
		return nil, &ValidationError{Field: "file"}
	}
	writeForm := func(w *multipart.Writer) error {
		if err := w.WriteField("caseId", params.CaseID); err != nil {
			return err
		}
		if params.RequirementID != "" {
			if err := w.WriteField("requirementId", params.RequirementID); err != nil {
				return err
			}
		}
		part, err := w.CreateFormFile("file", params.File.Filename)
		if err != nil {
			return err
		}
		if _, err := io.Copy(part, params.File.Reader); err != nil {
			return err
		}
		return nil
	}
	return s.client.doMultipart(ctx, "POST", "/compliance/files", writeForm, opts)
}

// GetFileParams are the parameters for the GetFile method.
type GetFileParams struct {
	// FileID is the fileId UploadFile returned. Required.
	FileID string
}

// GetFile downloads a compliance file. Roles: api, admin, developer, billing, user.
// The file bytes are raw content, not JSON: on success Response.Data holds them unchanged, so do not decode
// it as JSON or marshal the Response.
func (s *ComplianceService) GetFile(ctx context.Context, params GetFileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"fileId": params.FileID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/compliance/files/%s", url.PathEscape(params.FileID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}
