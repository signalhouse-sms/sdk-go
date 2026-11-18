package signalhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// SubgroupsService handles subgroup-related API calls.
type SubgroupsService struct {
	client *Client
}

// GetSubgroupsParams are the parameters for the GetSubgroups method.
type GetSubgroupsParams struct {
	ID      string
	GroupID string
	Page    string
	Limit   string
}

// GetSubgroups returns a list of subgroups with optional filters.
func (s *SubgroupsService) GetSubgroups(ctx context.Context, params GetSubgroupsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"id":      params.ID,
		"groupId": params.GroupID,
		"page":    params.Page,
		"limit":   params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/subgroup"+qs, nil, opts)
}

// GetSubgroupCountsParams are the optional filters for the GetSubgroupCounts method.
type GetSubgroupCountsParams struct {
	GroupID    string
	SubgroupID string
	Limit      string
	Offset     string
}

// GetSubgroupCounts returns resource counts grouped by subgroup ID. Only signalhouse_admin can query
// all groups; every other allowed role is restricted to its JWT group scope.
func (s *SubgroupsService) GetSubgroupCounts(ctx context.Context, params GetSubgroupCountsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":    params.GroupID,
		"subgroupId": params.SubgroupID,
		"limit":      params.Limit,
		"offset":     params.Offset,
	})
	return s.client.doJSON(ctx, "GET", "/subgroup/counts"+qs, nil, opts)
}

// LookupGhlLocationParams are the parameters for the LookupGhlLocation method. Provide exactly one of
// GhlLocationID or SubgroupID.
type LookupGhlLocationParams struct {
	GroupID       string
	GhlLocationID string
	SubgroupID    string
}

// LookupGhlLocation resolves the mapping between a GoHighLevel location and a Signal House subgroup,
// in either direction. GroupID is optional - customer roles default to their own accessible group(s)
// and are rejected if they name a group they don't belong to; a SubgroupID/GhlLocationID that belongs
// to a different group than the caller's returns a 404, identical to a nonexistent one.
func (s *SubgroupsService) LookupGhlLocation(ctx context.Context, params LookupGhlLocationParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":       params.GroupID,
		"ghlLocationId": params.GhlLocationID,
		"subgroupId":    params.SubgroupID,
	})
	return s.client.doJSON(ctx, "GET", "/subgroup/ghl-location-lookup"+qs, nil, opts)
}

// LinkGhlLocationParams are the parameters for the LinkGhlLocation method. Requires signalhouse_api or
// signalhouse_admin role. GhlLocationID nil unlinks; a non-nil GhlLocationID links, stealing it from any
// other subgroup that currently holds it. GhlLocationName is omitted from the request entirely when nil,
// leaving the subgroup's currently stored name untouched (never cleared, and only when GhlLocationID is
// unchanged - otherwise it is cleared); it is ignored when unlinking. ClaimedAt is required when
// GhlLocationID is non-nil: it is when this link was locally claimed by the caller, used to detect and
// reject a delayed/out-of-order push whose claim has since been superseded by a newer one recorded
// elsewhere; ignored when unlinking.
type LinkGhlLocationParams struct {
	SubgroupID      string
	GhlLocationID   *string
	GhlLocationName *string
	ClaimedAt       *time.Time
}

// MarshalJSON encodes the link body, omitting ghlLocationName entirely when nil (so the stored name is
// preserved, not cleared), and encoding ghlLocationId as JSON null when nil (unlink).
func (p LinkGhlLocationParams) MarshalJSON() ([]byte, error) {
	body := struct {
		SubgroupID      string           `json:"subgroupId"`
		GhlLocationID   *string          `json:"ghlLocationId"`
		GhlLocationName *json.RawMessage `json:"ghlLocationName,omitempty"`
		ClaimedAt       *time.Time       `json:"claimedAt,omitempty"`
	}{SubgroupID: p.SubgroupID, GhlLocationID: p.GhlLocationID, ClaimedAt: p.ClaimedAt}
	if p.GhlLocationName != nil {
		raw, err := json.Marshal(*p.GhlLocationName)
		if err != nil {
			return nil, err
		}
		rawMsg := json.RawMessage(raw)
		body.GhlLocationName = &rawMsg
	}
	return json.Marshal(body)
}

// LinkGhlLocation links (or, with GhlLocationID nil, unlinks) a GoHighLevel location to a subgroup.
func (s *SubgroupsService) LinkGhlLocation(ctx context.Context, params LinkGhlLocationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	if params.GhlLocationID != nil && params.ClaimedAt == nil {
		return nil, &ValidationError{Field: "claimedAt"}
	}
	return s.client.doJSON(ctx, "PUT", "/subgroup/ghl-location-link", params, opts)
}

// CreateSubgroupData represents the data for creating a subgroup.
type CreateSubgroupData struct {
	GroupID             string `json:"groupId"`
	SubgroupName        string `json:"subgroupName"`
	ContactFirstName    string `json:"contactFirstName,omitempty"`
	ContactLastName     string `json:"contactLastName,omitempty"`
	ContactMiddleName   string `json:"contactMiddleName,omitempty"`
	SubgroupCompanyName string `json:"subgroupCompanyName,omitempty"`
	AddressLine1        string `json:"addressLine1,omitempty"`
	AddressLine2        string `json:"addressLine2,omitempty"`
	City                string `json:"city,omitempty"`
	State               string `json:"state,omitempty"`
	Country             string `json:"country,omitempty"`
	PostalCode          string `json:"postalCode,omitempty"`
	Phone               string `json:"phone,omitempty"`
	// CarrierIDFamily and CarrierIDRegion require signalhouse_api, signalhouse_admin,
	// signalhouse_user, api, admin, developer, or billing role when set.
	CarrierIDFamily string `json:"carrierIdFamily,omitempty"`
	CarrierIDRegion string `json:"carrierIdRegion,omitempty"`
}

// CreateSubgroupParams are the parameters for the CreateSubgroup method.
type CreateSubgroupParams struct {
	SubgroupData CreateSubgroupData
}

// CreateSubgroup creates a new subgroup.
func (s *SubgroupsService) CreateSubgroup(ctx context.Context, params CreateSubgroupParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/subgroup", params.SubgroupData, opts)
}

// UpdateSubgroupData represents the data for updating a subgroup.
type UpdateSubgroupData struct {
	SubgroupName        string `json:"subgroupName,omitempty"`
	ContactFirstName    string `json:"contactFirstName,omitempty"`
	ContactLastName     string `json:"contactLastName,omitempty"`
	ContactMiddleName   string `json:"contactMiddleName,omitempty"`
	SubgroupCompanyName string `json:"subgroupCompanyName,omitempty"`
	AddressLine1        string `json:"addressLine1,omitempty"`
	AddressLine2        string `json:"addressLine2,omitempty"`
	City                string `json:"city,omitempty"`
	State               string `json:"state,omitempty"`
	Country             string `json:"country,omitempty"`
	PostalCode          string `json:"postalCode,omitempty"`
	Phone               string `json:"phone,omitempty"`
	Status              string `json:"status,omitempty"`
	// CarrierIDFamily and CarrierIDRegion require signalhouse_api, signalhouse_admin,
	// signalhouse_user, api, admin, developer, or billing role when set.
	CarrierIDFamily string `json:"carrierIdFamily,omitempty"`
	CarrierIDRegion string `json:"carrierIdRegion,omitempty"`
	// CnamAutoApply turns caller name (CNAM) auto-apply on or off for the subgroup's numbers.
	CnamAutoApply *bool `json:"cnamAutoApply,omitempty"`
	// CnamBrandID is the verified brand caller names come from for numbers with no campaign.
	// Leave nil to leave the stored value alone.
	CnamBrandID *string `json:"-"`
	// ClearCnamBrandID sends cnamBrandId as null, clearing the subgroup's caller-name brand. It
	// takes precedence over CnamBrandID.
	ClearCnamBrandID bool `json:"-"`
}

// MarshalJSON encodes the update body, writing cnamBrandId when CnamBrandID is set and an
// explicit null when ClearCnamBrandID is set.
func (d UpdateSubgroupData) MarshalJSON() ([]byte, error) {
	type plain UpdateSubgroupData
	body := struct {
		plain
		CnamBrandID *json.RawMessage `json:"cnamBrandId,omitempty"`
	}{plain: plain(d)}
	if d.ClearCnamBrandID {
		null := json.RawMessage("null")
		body.CnamBrandID = &null
	} else if d.CnamBrandID != nil {
		raw, err := json.Marshal(*d.CnamBrandID)
		if err != nil {
			return nil, err
		}
		rawMsg := json.RawMessage(raw)
		body.CnamBrandID = &rawMsg
	}
	return json.Marshal(body)
}

// UpdateSubgroupParams are the parameters for the UpdateSubgroup method.
type UpdateSubgroupParams struct {
	ID         string
	UpdateData UpdateSubgroupData
}

// UpdateSubgroup updates an existing subgroup.
func (s *SubgroupsService) UpdateSubgroup(ctx context.Context, params UpdateSubgroupParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/subgroup/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// DeleteSubgroupParams are the parameters for the DeleteSubgroup method.
type DeleteSubgroupParams struct {
	ID string
}

// DeleteSubgroup deletes (marks as inactive) a subgroup.
func (s *SubgroupsService) DeleteSubgroup(ctx context.Context, params DeleteSubgroupParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/subgroup/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}
