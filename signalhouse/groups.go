package signalhouse

import (
	"context"
	"fmt"
	"net/url"
)

// GroupsService handles group-related API calls.
type GroupsService struct {
	client      *Client
	enableAdmin bool

	// Admin contains admin-only group methods. Only populated when EnableAdmin is true.
	Admin *GroupsAdminService
}

// GroupsAdminService contains admin-only group methods.
type GroupsAdminService struct {
	client *Client
}

// AdminGetGroupCountsParams are the optional filters for the admin GetGroupCounts method.
type AdminGetGroupCountsParams struct {
	GroupID string
	Limit   string
	Offset  string
}

// GetGroupCounts returns resource counts grouped by group ID. Only signalhouse_admin can query all
// groups; every other allowed role is restricted to its JWT group scope.
func (s *GroupsAdminService) GetGroupCounts(ctx context.Context, params AdminGetGroupCountsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId": params.GroupID,
		"limit":   params.Limit,
		"offset":  params.Offset,
	})
	return s.client.doJSON(ctx, "GET", "/group/counts"+qs, nil, opts)
}

// GetGroupParams are the parameters for the GetGroup method.
type GetGroupParams struct {
	ID string
}

// GetGroup returns details of a group by its ID.
func (s *GroupsService) GetGroup(ctx context.Context, params GetGroupParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"id": params.ID,
	})
	return s.client.doJSON(ctx, "GET", "/group"+qs, nil, opts)
}

// UpdateGroupParams are the parameters for the UpdateGroup method.
// GroupData may include optional CNP fields: cspId (string|nil), defaultCnpSubgroupId (string|nil).
type UpdateGroupParams struct {
	ID        string
	GroupData map[string]interface{}
}

// UpdateGroup updates a group.
func (s *GroupsService) UpdateGroup(ctx context.Context, params UpdateGroupParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"id": params.ID}); err != nil {
		return nil, err
	}
	if err := requireNonNil(map[string]interface{}{"groupData": params.GroupData}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/%s", url.PathEscape(params.ID))
	return s.client.doJSON(ctx, "PUT", path, params.GroupData, opts)
}

// AdminGetGroupsParams are the parameters for the admin GetGroups method.
type AdminGetGroupsParams struct {
	Page  string
	Limit string
}

// GetGroups returns a list of all groups. Admin only.
func (s *GroupsAdminService) GetGroups(ctx context.Context, params AdminGetGroupsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"page":  params.Page,
		"limit": params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/group"+qs, nil, opts)
}

// AdminCreateGroupParams are the parameters for the admin CreateGroup method.
type AdminCreateGroupParams struct {
	GroupData map[string]interface{}
}

// CreateGroup creates a new group. Admin only.
// GroupData optionally accepts psPartnerKey and ownerEmail to credit a referring partner.
func (s *GroupsAdminService) CreateGroup(ctx context.Context, params AdminCreateGroupParams, opts *RequestOptions) (*Response, error) {
	if err := requireNonNil(map[string]interface{}{"groupData": params.GroupData}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/group", params.GroupData, opts)
}

// AdminDeleteGroupParams are the parameters for the admin DeleteGroup method.
type AdminDeleteGroupParams struct {
	GroupID string
}

// DeleteGroup deletes a group. Admin only.
func (s *GroupsAdminService) DeleteGroup(ctx context.Context, params AdminDeleteGroupParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/%s", url.PathEscape(params.GroupID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// AdminUpdateComplianceStatusParams are the parameters for the admin UpdateComplianceStatus method.
type AdminUpdateComplianceStatusParams struct {
	GroupID          string
	ComplianceStatus string
}

// UpdateComplianceStatus sets a group's KYC compliance status. Only flips the field —
// it does not trigger any lockout side effects (halted sends, revoked API keys, etc.) for
// "rejected". Admin only.
func (s *GroupsAdminService) UpdateComplianceStatus(ctx context.Context, params AdminUpdateComplianceStatusParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "complianceStatus": params.ComplianceStatus}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/%s/compliance-status", url.PathEscape(params.GroupID))
	body := map[string]interface{}{"complianceStatus": params.ComplianceStatus}
	return s.client.doJSON(ctx, "PUT", path, body, opts)
}

// AdminLinkExternalParams are the parameters for the admin LinkExternal method.
// ExistingGroupID is optional; it is only sent when non-empty.
type AdminLinkExternalParams struct {
	LinkToken       string `json:"linkToken"`
	ExternalSystem  string `json:"externalSystem"`
	ExternalID      string `json:"externalId"`
	ExistingGroupID string `json:"existingGroupId,omitempty"`
}

// LinkExternal links an external tenant (GHL/Shopify) to a V2 group (server-to-server). It
// exchanges a single-use link token for a canonical group, adopting an empty portal group,
// repointing to an existing group, or flagging for manual review. Admin only.
func (s *GroupsAdminService) LinkExternal(ctx context.Context, params AdminLinkExternalParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"linkToken":      params.LinkToken,
		"externalSystem": params.ExternalSystem,
		"externalId":     params.ExternalID,
	}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/group/link-external", params, opts)
}

// GetSsoConfiguration returns a group's Enterprise SSO configuration: whether SSO is available, the
// redirect URL to register at the identity provider, the saved provider (never its client secret), and
// each claimed domain with the DNS TXT record that verifies it
// ({ groupId, available, callbackUrl, connection, domains }).
func (s *GroupsService) GetSsoConfiguration(ctx context.Context, groupID string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": groupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": groupID})
	return s.client.doJSON(ctx, "GET", "/group/sso"+qs, nil, opts)
}

// SaveSsoConnectionParams are the parameters for the SaveSsoConnection method.
type SaveSsoConnectionParams struct {
	GroupID string `json:"groupId"`
	// Issuer is the provider's https issuer URL.
	Issuer string `json:"issuer"`
	// ClientID is the client ID registered at the provider.
	ClientID string `json:"clientId"`
	// ClientSecret is required the first time; leave empty to keep the saved one.
	ClientSecret string `json:"clientSecret,omitempty"`
	// Enabled turns sign-in through the provider on or off (default true when nil).
	Enabled *bool `json:"enabled,omitempty"`
}

// SaveSsoConnection creates or updates a group's OpenID Connect identity provider and returns the SSO
// configuration. Requires a fresh Auth.ConfirmTwoFactorIdentity from the same session.
func (s *GroupsService) SaveSsoConnection(ctx context.Context, params SaveSsoConnectionParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"groupId":  params.GroupID,
		"issuer":   params.Issuer,
		"clientId": params.ClientID,
	}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "PUT", "/group/sso", params, opts)
}

// DeleteSsoConnection removes a group's identity provider and its domains, and unlinks every account
// that signed in through it. Returns the (now empty) SSO configuration. Requires a fresh
// Auth.ConfirmTwoFactorIdentity from the same session.
func (s *GroupsService) DeleteSsoConnection(ctx context.Context, groupID string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": groupID}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": groupID})
	return s.client.doJSON(ctx, "DELETE", "/group/sso"+qs, nil, opts)
}

// SsoDomainParams are the parameters for the AddSsoDomain, VerifySsoDomain and RemoveSsoDomain methods.
type SsoDomainParams struct {
	GroupID string `json:"groupId"`
	// Domain is the email domain, e.g. "acme.com".
	Domain string `json:"domain"`
}

// AddSsoDomain claims an email domain for a group's SSO. Returns the SSO configuration, which includes
// the DNS TXT record that verifies the domain.
func (s *GroupsService) AddSsoDomain(ctx context.Context, params SsoDomainParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "domain": params.Domain}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/group/sso/domains", params, opts)
}

// VerifySsoDomain verifies a claimed domain by its DNS TXT record and returns the SSO configuration.
func (s *GroupsService) VerifySsoDomain(ctx context.Context, params SsoDomainParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "domain": params.Domain}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/group/sso/domains/%s/verify", url.PathEscape(params.Domain))
	body := map[string]string{"groupId": params.GroupID}
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// RemoveSsoDomain removes a domain from a group's SSO and returns the SSO configuration.
func (s *GroupsService) RemoveSsoDomain(ctx context.Context, params SsoDomainParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "domain": params.Domain}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"groupId": params.GroupID})
	path := fmt.Sprintf("/group/sso/domains/%s", url.PathEscape(params.Domain))
	return s.client.doJSON(ctx, "DELETE", path+qs, nil, opts)
}
