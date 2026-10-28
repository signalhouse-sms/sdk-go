package signalhouse

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// RegistrationsService handles sender registrations: the catalog, quotes, submission and status tracking for
// every jurisdiction.
type RegistrationsService struct {
	client      *Client
	enableAdmin bool

	// Admin contains admin-only registration methods. Only populated when EnableAdmin is true.
	Admin *RegistrationsAdminService
}

// RegistrationsAdminService contains admin-only registration methods. Enabling it grants no server permission.
type RegistrationsAdminService struct {
	client *Client
}

// GetCatalogParams are the parameters for the GetCatalog method.
type GetCatalogParams struct {
	// Region is the jurisdiction (US, GB, CA, AU; any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB).
	Region string
}

// GetCatalog returns the registration types a jurisdiction offers and whether each can be acquired right now:
// region, jurisdiction, sendingRequiresRegistration, registeredThrough, provisioning and types, each with type, kind, channel,
// bundles, quantity {min, max}, requiresParent and available.
func (s *RegistrationsService) GetCatalog(ctx context.Context, params GetCatalogParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"region": params.Region}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"region": params.Region})
	return s.client.doJSON(ctx, "GET", "/registration/catalog"+qs, nil, opts)
}

// GetQuotesParams are the parameters for the GetQuotes method.
type GetQuotesParams struct {
	// Region is the jurisdiction (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB).
	Region string
	// Type is the registration type, e.g. "VIRTUAL_LONG_CODE", or "ALPHANUMERIC_SENDER_ID" for GB (one ["SMS"]
	// bundle, 6000000 setup and 6000000 monthly by default).
	Type string
	// Quantity is how many senders the order covers (1-100). Zero omits it and the server defaults to 1.
	Quantity int
	// GroupID is staff only: quote on behalf of another group.
	GroupID string
}

// GetQuotes returns setup and monthly prices, in integer microdollars, for each capability bundle of a type. The
// data is one quote per bundle: capabilities, quantity, currency, setupAmount, monthlyAmount, totalSetupAmount,
// totalMonthlyAmount and etaDays. setupAmount/monthlyAmount are per sender; the totals cover the quantity.
func (s *RegistrationsService) GetQuotes(ctx context.Context, params GetQuotesParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"region": params.Region, "type": params.Type}); err != nil {
		return nil, err
	}
	quantity := ""
	if params.Quantity > 0 {
		quantity = strconv.Itoa(params.Quantity)
	}
	qs := buildQueryString(map[string]string{
		"region":   params.Region,
		"type":     params.Type,
		"quantity": quantity,
		"groupId":  params.GroupID,
	})
	return s.client.doJSON(ctx, "GET", "/registration/quotes"+qs, nil, opts)
}

// GetRegistrationsParams are the parameters for the GetRegistrations method.
type GetRegistrationsParams struct {
	// GroupID scopes the list; staff may omit it for the review queue.
	GroupID    string
	SubgroupID string
	// Region filters by one jurisdiction or a comma-separated list (e.g. "GB,CA").
	Region string
	Type   string
	// Kind is identity, program or sender.
	Kind string
	// Status filters by one registration status or a comma-separated list (e.g. "PENDING_PROVIDER,APPROVED"),
	// from DRAFT, SIGNAL_HOUSE_REVIEW, SIGNAL_HOUSE_APPROVED, SIGNAL_HOUSE_REJECTED, SUBMITTING,
	// PENDING_PROVIDER, NEEDS_INFORMATION, PAYMENT_REQUIRED, APPROVED, REJECTED, CANCELLED, PENDING_DELETE, DELETED,
	// EXPIRED or RECONCILIATION_REQUIRED.
	Status               string
	ParentRegistrationID string
	// PhoneNumber filters to registrations holding this phone number.
	PhoneNumber string
	// SortBy is createdAt (default), subgroupId, registrationId, region or status.
	SortBy string
	// SortOrder is asc or desc (default).
	SortOrder string
	Page      string
	// Limit is at most 100.
	Limit string
}

// GetRegistrations lists registrations with optional filters and pagination. The response carries data,
// totalCount, page and limit.
func (s *RegistrationsService) GetRegistrations(ctx context.Context, params GetRegistrationsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":              params.GroupID,
		"subgroupId":           params.SubgroupID,
		"region":               params.Region,
		"type":                 params.Type,
		"kind":                 params.Kind,
		"status":               params.Status,
		"parentRegistrationId": params.ParentRegistrationID,
		"phoneNumber":          params.PhoneNumber,
		"sortBy":               params.SortBy,
		"sortOrder":            params.SortOrder,
		"page":                 params.Page,
		"limit":                params.Limit,
	})
	return s.client.doJSON(ctx, "GET", "/registration"+qs, nil, opts)
}

// GetRegistrationParams are the parameters for the GetRegistration method.
type GetRegistrationParams struct {
	RegistrationID string
}

// GetRegistration returns a single registration with its type data, review outcome and status history.
func (s *RegistrationsService) GetRegistration(ctx context.Context, params GetRegistrationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// CancelRegistrationParams are the parameters for the CancelRegistration method.
type CancelRegistrationParams struct {
	RegistrationID string
}

// CancelRegistration cancels a registration. Allowed from SIGNAL_HOUSE_REVIEW, SIGNAL_HOUSE_APPROVED,
// PENDING_PROVIDER, NEEDS_INFORMATION, PAYMENT_REQUIRED and APPROVED (409 otherwise). Numbers already allocated
// stay on the account and stop sending.
func (s *RegistrationsService) CancelRegistration(ctx context.Context, params CancelRegistrationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// CreateRegistrationParams are the parameters for the CreateRegistration method.
type CreateRegistrationParams struct {
	// Region is the jurisdiction (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB).
	Region string `json:"region"`
	// Type selects the shape of Data, e.g. "VIRTUAL_LONG_CODE", or "ALPHANUMERIC_SENDER_ID" for GB.
	Type       string `json:"type"`
	SubgroupID string `json:"subgroupId"`
	// Capabilities is one of the type's bundles from the catalog, e.g. ["SMS"] or ["SMS", "VOICE"].
	Capabilities []string `json:"capabilities"`
	// Data is the type's submission; for VIRTUAL_LONG_CODE: quantity (1-100), useCases (1-10) and voice, required
	// exactly when Capabilities includes VOICE. For GB ALPHANUMERIC_SENDER_ID (Capabilities ["SMS"]):
	// requestedSenderId (3-11 letters/digits/space/dot/dash, at least one letter), trafficOrigin (LOCAL |
	// INTERNATIONAL), trafficType (TRANSACTIONAL | PROMOTIONAL), companyName, companyCountry, companyWebsite,
	// industry, messageExample and senderRelationship, required when requestedSenderId differs from companyName;
	// quantity is always 1 and data.phoneNumbers carries the granted sender (e.g. ["ACME"]) once APPROVED.
	Data map[string]any `json:"data"`
}

// CreateRegistration submits a registration for Signal House review and returns the new registration (HTTP 201).
func (s *RegistrationsService) CreateRegistration(ctx context.Context, params CreateRegistrationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"region": params.Region, "type": params.Type, "subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/registration", params, opts)
}

// ReviewRegistrationParams are the parameters for the ReviewRegistration method.
type ReviewRegistrationParams struct {
	RegistrationID string `json:"-"`
	// Action is "APPROVE" (submits the registration to the provider) or "REJECT".
	Action string `json:"action"`
	// Reason says why the registration was rejected; required for "REJECT".
	Reason string `json:"reason,omitempty"`
}

// ReviewRegistration approves or rejects a registration awaiting Signal House review. Admin only.
func (s *RegistrationsAdminService) ReviewRegistration(ctx context.Context, params ReviewRegistrationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID, "action": params.Action}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s/review", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// CorrectRegistrationParams are the parameters for the CorrectRegistration method.
type CorrectRegistrationParams struct {
	RegistrationID string `json:"-"`
	Reason         string `json:"reason"`
	// Data holds the type's correctable fields (VIRTUAL_LONG_CODE: useCases and, for SMS + Voice, voice). Owner,
	// quantity, capabilities and the provider order are fixed.
	Data map[string]any `json:"data"`
}

// CorrectRegistration saves corrections for a registration the provider sent back for more information. Admin only.
func (s *RegistrationsAdminService) CorrectRegistration(ctx context.Context, params CorrectRegistrationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID, "reason": params.Reason}); err != nil {
		return nil, err
	}
	if params.Data == nil {
		return nil, &ValidationError{Field: "data"}
	}
	path := fmt.Sprintf("/registration/%s/corrections", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// AddNumberParams are the parameters for the AddNumber method.
type AddNumberParams struct {
	RegistrationID string `json:"-"`
	// PhoneNumber is the allocated number in E.164 digits (10-15 digits, optional leading +).
	PhoneNumber string `json:"phoneNumber"`
}

// AddNumber allocates a provider-owned number to a registration through the staff recovery path. Admin only.
func (s *RegistrationsAdminService) AddNumber(ctx context.Context, params AddNumberParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID, "phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s/numbers", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// ReconcileParams are the parameters for the Reconcile method.
type ReconcileParams struct {
	RegistrationID string `json:"-"`
	// ExternalID is the provider's identifier for the existing order.
	ExternalID string `json:"externalId"`
}

// Reconcile links a RECONCILIATION_REQUIRED registration to the provider order that was actually placed. Never
// place a second order to retry an ambiguous submission. Admin only.
func (s *RegistrationsAdminService) Reconcile(ctx context.Context, params ReconcileParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID, "externalId": params.ExternalID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s/reconcile", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "POST", path, params, opts)
}

// GetApplicationParams are the parameters for the GetApplication method.
type GetApplicationParams struct {
	RegistrationID string
}

// GetApplication returns the completed carrier application as {filename, contentType, base64}; decode base64 to
// save the file. Admin only.
func (s *RegistrationsAdminService) GetApplication(ctx context.Context, params GetApplicationParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"registrationId": params.RegistrationID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/registration/%s/application?format=json", url.PathEscape(params.RegistrationID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}
