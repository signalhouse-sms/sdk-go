package signalhouse

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// NumbersService handles phone number-related API calls.
type NumbersService struct {
	client      *Client
	enableAdmin bool

	// Admin contains admin-only number methods. Only populated when EnableAdmin is true.
	Admin *NumbersAdminService
}

// NumbersAdminService contains admin-only number methods.
type NumbersAdminService struct {
	client *Client
}

// GetPhoneNumbersParams are the parameters for the GetPhoneNumbers method.
type GetPhoneNumbersParams struct {
	PhoneNumber string
	CampaignID  string
	// RegistrationID filters to numbers provisioned under this registration.
	RegistrationID string
	BrandID        string
	SubgroupID     string
	GroupID        string
	Page           string
	Limit          string
	// CurrentTollFreeCarrier, when true with GroupID or SubgroupID, returns only the toll-free numbers a new toll-free
	// campaign in that group can use.
	CurrentTollFreeCarrier *bool
}

// GetPhoneNumbers returns a list of phone numbers with optional filters.
//
// Every record carries a `sentiment` object -- trailing 7-day and 30-day inbound reply sentiment,
// refreshed periodically from analytics rather than computed per request: `{ sevenDay, thirtyDay,
// updatedAt }`, each window being `{ score, label, scoredCount, positive, neutral, negative }`. The
// object is always present and fully populated, so it can be read without a presence check.
// `score` is a volume-weighted average in -100..100 and is null when nothing was scored in the
// window, which is not the same as a score of 0 -- null means nobody replied, 0 means replies
// averaged neutral. `updatedAt` is null until the first rollup writes the record, and it tracks when
// the figures last CHANGED, not when the job last ran.
//
// Every record also carries a `health` object -- trailing 7-day and 30-day delivery/opt-out health from
// the same periodic refresh: `{ sevenDay, thirtyDay, updatedAt }`, each window being `{ score,
// deliveryRate, optOutRate, deliveryScore, optOutScore, messagesSent, messagesDelivered, messagesFailed,
// optOuts }`. `score` is 1.0..10.0 (the mean of the two bucket scores) and is null when nothing was sent
// in the window; the volumes say how much traffic sits behind it. A campaign's figure is scored from the
// campaign's own summed counters, never averaged from its numbers. For a live figure use GET /number/health
// or GET /campaign/health.
func (s *NumbersService) GetPhoneNumbers(ctx context.Context, params GetPhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"phoneNumber":            params.PhoneNumber,
		"campaignId":             params.CampaignID,
		"registrationId":         params.RegistrationID,
		"brandId":                params.BrandID,
		"subgroupId":             params.SubgroupID,
		"groupId":                params.GroupID,
		"page":                   params.Page,
		"limit":                  params.Limit,
		"currentTollFreeCarrier": boolQueryValue(params.CurrentTollFreeCarrier),
	})
	return s.client.doJSON(ctx, "GET", "/number"+qs, nil, opts)
}

// GetNumberHealth returns the health score, the same one the portal shows, (1–10) for an owned phone number.
func (s *NumbersService) GetNumberHealth(ctx context.Context, phoneNumber string, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": phoneNumber}); err != nil {
		return nil, err
	}
	qs := buildQueryString(map[string]string{"phoneNumber": phoneNumber})
	return s.client.doJSON(ctx, "GET", "/number/health"+qs, nil, opts)
}

// GetAvailablePhoneNumbersParams are the parameters for the GetAvailablePhoneNumbers method.
type GetAvailablePhoneNumbersParams struct {
	SMSEnabled   string
	MMSEnabled   string
	VoiceEnabled string
	Country      string
	State        string
	City         string // City prefix; requires US country and state.
	NPA          string
	NXX          string
	PhoneNumber  string
	Limit        string
	Page         string
}

// GetAvailablePhoneNumbers returns available phone numbers for purchase.
func (s *NumbersService) GetAvailablePhoneNumbers(ctx context.Context, params GetAvailablePhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"smsEnabled":   params.SMSEnabled,
		"mmsEnabled":   params.MMSEnabled,
		"voiceEnabled": params.VoiceEnabled,
		"country":      params.Country,
		"state":        params.State,
		"city":         params.City,
		"npa":          params.NPA,
		"nxx":          params.NXX,
		"phoneNumber":  params.PhoneNumber,
		"limit":        params.Limit,
		"page":         params.Page,
	})
	return s.client.doJSON(ctx, "GET", "/number/available"+qs, nil, opts)
}

// PurchasePhoneNumberParams are the parameters for the PurchasePhoneNumber method.
type PurchasePhoneNumberParams struct {
	// Country is the ISO-2 purchase country; US (default).
	Country      string   `json:"country,omitempty"`
	PhoneNumbers []string `json:"phoneNumbers"`
	SubgroupID   string   `json:"subgroupId"`
}

// PurchasePhoneNumber purchases phone numbers and assigns them to a subgroup.
//
// Each requested number is queued and processed independently. Per-number success or failure is
// delivered via NUMBER_PURCHASE_SUCCESSFUL / NUMBER_PURCHASE_FAILED webhooks — not in this response.
func (s *NumbersService) PurchasePhoneNumber(ctx context.Context, params PurchasePhoneNumberParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/number", params, opts)
}

// PurchaseTollFreeNumbersParams are the parameters for the PurchaseTollFreeNumbers method.
type PurchaseTollFreeNumbersParams struct {
	// Country is the ISO-2 purchase country; US (default). Toll-Free uses the shared registration flow.
	Country string `json:"country,omitempty"`
	// Quantity is the number of Toll-Free numbers to purchase (1-10).
	Quantity   int    `json:"quantity"`
	SubgroupID string `json:"subgroupId"`
}

// PurchaseTollFreeNumbers purchases one or more Toll-Free numbers via the asynchronous
// resource-request flow.
//
// Toll-Free numbers are ordered by quantity (not picked individually) and provisioned
// asynchronously; per-number completion is delivered via webhook/polling, not in this response.
// Poll the returned orderId with GetTollFreeOrderStatus.
func (s *NumbersService) PurchaseTollFreeNumbers(ctx context.Context, params PurchaseTollFreeNumbersParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"subgroupId": params.SubgroupID}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/number/toll-free", params, opts)
}

// RequestShortCodeAcquisitionParams are the parameters for the RequestShortCodeAcquisition method.
type RequestShortCodeAcquisitionParams struct {
	SubgroupID          string `json:"subgroupId"`
	BrandID             string `json:"brandId"`
	CampaignID          string `json:"campaignId"`
	RequestType         string `json:"requestType"`
	RequestedVanityCode string `json:"requestedVanityCode,omitempty"`
	LeaseTermMonths     *int   `json:"leaseTermMonths,omitempty"`
	ActualCode          string `json:"actualCode,omitempty"`
	OrderID             string `json:"orderId,omitempty"`
	// LeaseEndDate is the future Registry lease end date. A YYYY-MM-DD value expires at 23:59:59 UTC.
	LeaseEndDate string `json:"leaseEndDate,omitempty"`
	// LeaseReceiptFile is the external lease receipt PNG, JPEG, or PDF, required for EXTERNAL_LEASE.
	LeaseReceiptFile *FileUpload `json:"-"`
}

// RequestShortCodeAcquisition requests a campaign-bound Short Code acquisition or registers a customer-owned
// Registry lease. Random and vanity requests are fulfilled by Signal House staff; an external lease creates a
// pending number immediately and becomes READY only when its campaign is ACTIVE.
// EXTERNAL_LEASE requests must include LeaseReceiptFile, which is uploaded as multipart form data.
func (s *NumbersService) RequestShortCodeAcquisition(ctx context.Context, params RequestShortCodeAcquisitionParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"subgroupId": params.SubgroupID, "brandId": params.BrandID, "campaignId": params.CampaignID, "requestType": params.RequestType}); err != nil {
		return nil, err
	}
	writeForm := func(w *multipart.Writer) error {
		if err := marshalToField(w, "acquisitionData", params); err != nil {
			return err
		}
		if params.LeaseReceiptFile != nil {
			receiptPart, err := w.CreateFormFile("leaseReceipt", params.LeaseReceiptFile.Filename)
			if err != nil {
				return err
			}
			if _, err := io.Copy(receiptPart, params.LeaseReceiptFile.Reader); err != nil {
				return err
			}
		}
		return nil
	}
	return s.client.doMultipart(ctx, "POST", "/number/short-code", writeForm, opts)
}

// GetTollFreeOrderStatusParams are the parameters for the GetTollFreeOrderStatus method.
type GetTollFreeOrderStatusParams struct {
	OrderID string
}

// GetTollFreeOrderStatus reads the outcome of a Toll-Free number purchase by its order id. It
// reports per-number ready/provisioning/failed counts (plus failure reasons) and the provisioned
// numbers; numbers fills in as the order completes, so poll until the order reaches a terminal state.
func (s *NumbersService) GetTollFreeOrderStatus(ctx context.Context, params GetTollFreeOrderStatusParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"orderId": params.OrderID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/toll-free/order/%s", url.PathEscape(params.OrderID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// UpdatePhoneNumberParams are the parameters for the UpdatePhoneNumber method.
type UpdatePhoneNumberParams struct {
	PhoneNumber string
	UpdateData  map[string]interface{}
}

// UpdatePhoneNumber updates a phone number's details.
func (s *NumbersService) UpdatePhoneNumber(ctx context.Context, params UpdatePhoneNumberParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	if err := requireNonNil(map[string]interface{}{"updateData": params.UpdateData}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/%s", url.PathEscape(params.PhoneNumber))
	return s.client.doJSON(ctx, "PUT", path, params.UpdateData, opts)
}

// GetCallerNameParams are the parameters for the GetCallerName method.
type GetCallerNameParams struct {
	// PhoneNumber is the phone number (E.164, leading + optional).
	PhoneNumber string
}

// GetCallerName reads a phone number's outbound caller name (CNAM) and whether one can be set.
// Resolves to { phoneNumber, cnam, autoApply, eligible, ineligibleReason }.
// Roles: api, admin, developer, billing, user.
// GET /number/{phoneNumber}/cnam
func (s *NumbersService) GetCallerName(ctx context.Context, params GetCallerNameParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/%s/cnam", url.PathEscape(params.PhoneNumber))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// SetCallerNameParams are the parameters for the SetCallerName method.
type SetCallerNameParams struct {
	// PhoneNumber is the phone number (E.164, leading + optional).
	PhoneNumber string `json:"-"`
	// Name is the caller name to show. It may only remove characters from the registered name of
	// the number's brand, and must be 1-15 letters, digits, spaces, '-' or '_'.
	Name string `json:"name"`
	// BrandID is the verified brand the name comes from; only for numbers with no campaign.
	BrandID string `json:"brandId,omitempty"`
}

// SetCallerName sets a phone number's outbound caller name (CNAM). The name may only remove
// characters from the registered name of the number's brand (its campaign's brand, else BrandID,
// else the subgroup's default caller-ID brand).
// Resolves to { phoneNumber, cnam, autoApply, eligible, ineligibleReason }.
// Roles: api, admin, developer.
// PUT /number/{phoneNumber}/cnam
func (s *NumbersService) SetCallerName(ctx context.Context, params SetCallerNameParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber, "name": params.Name}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/%s/cnam", url.PathEscape(params.PhoneNumber))
	return s.client.doJSON(ctx, "PUT", path, params, opts)
}

// DeleteCallerNameParams are the parameters for the DeleteCallerName method.
type DeleteCallerNameParams struct {
	// PhoneNumber is the phone number (E.164, leading + optional).
	PhoneNumber string
}

// DeleteCallerName removes a phone number's own caller name (CNAM). If its subgroup has caller
// name auto-apply on, the number goes back to the automatic name.
// Resolves to { phoneNumber, cnam, autoApply, eligible, ineligibleReason }.
// Roles: api, admin, developer.
// DELETE /number/{phoneNumber}/cnam
func (s *NumbersService) DeleteCallerName(ctx context.Context, params DeleteCallerNameParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"phoneNumber": params.PhoneNumber}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/%s/cnam", url.PathEscape(params.PhoneNumber))
	return s.client.doJSON(ctx, "DELETE", path, nil, opts)
}

// PortInPhoneNumberParams are the parameters for the PortInPhoneNumber method.
type PortInPhoneNumberParams struct {
	NumberData map[string]interface{} `json:"-"`
}

// PortInPhoneNumber submits a port-in request for one or more phone numbers from another provider.
func (s *NumbersService) PortInPhoneNumber(ctx context.Context, params PortInPhoneNumberParams, opts *RequestOptions) (*Response, error) {
	if err := requireNonNil(map[string]interface{}{"numberData": params.NumberData}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/number/portin", params.NumberData, opts)
}

// GetPortRequestsParams are the parameters for the GetPortRequests method.
type GetPortRequestsParams struct {
	GroupID     string
	PhoneNumber string
	Status      string
	Page        string
	Limit       string
	// ExcludeDocuments, when "true", leaves signatureUrl and billingUrl out of each request.
	ExcludeDocuments string
}

// GetPortRequests retrieves port-in requests with optional filters.
// Signal House staff can view all requests; regular users see only their group's requests.
func (s *NumbersService) GetPortRequests(ctx context.Context, params GetPortRequestsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":          params.GroupID,
		"phoneNumber":      params.PhoneNumber,
		"status":           params.Status,
		"page":             params.Page,
		"limit":            params.Limit,
		"excludeDocuments": params.ExcludeDocuments,
	})
	return s.client.doJSON(ctx, "GET", "/number/portin"+qs, nil, opts)
}

// AssignPhoneNumberToCampaignParams are the parameters for the AssignPhoneNumberToCampaign method.
type AssignPhoneNumberToCampaignParams struct {
	CampaignID   string   `json:"-"`
	PhoneNumbers []string `json:"phoneNumbers"`
}

// AssignPhoneNumberToCampaign assigns phone numbers to a campaign. Numbers that are not found, already
// RELEASED, already assigned to a different campaign, on a mismatched DCA, or (for a 10DLC campaign
// without a number pool) beyond the remaining 49-number-per-campaign limit are not queued; the response
// Data holds {message, queued, skipped}, where skipped numbers were rejected individually for one of
// those reasons rather than failing the whole request. A Toll-Free campaign whose brand reports
// tollFree.maxNumbersPerCampaign as 1 holds one number: a request that would give it a second is
// refused with a 400 and nothing is queued.
func (s *NumbersService) AssignPhoneNumberToCampaign(ctx context.Context, params AssignPhoneNumberToCampaignParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"campaignId": params.CampaignID}); err != nil {
		return nil, err
	}
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	path := fmt.Sprintf("/number/assign/%s", url.PathEscape(params.CampaignID))
	body := map[string][]string{"phoneNumbers": params.PhoneNumbers}
	return s.client.doJSON(ctx, "POST", path, body, opts)
}

// UnassignPhoneNumbersFromCampaignsParams are the parameters for the UnassignPhoneNumbersFromCampaigns method.
type UnassignPhoneNumbersFromCampaignsParams struct {
	PhoneNumbers []string `json:"phoneNumbers"`
}

// UnassignPhoneNumbersFromCampaigns unassigns phone numbers from their campaigns.
func (s *NumbersService) UnassignPhoneNumbersFromCampaigns(ctx context.Context, params UnassignPhoneNumbersFromCampaignsParams, opts *RequestOptions) (*Response, error) {
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/number/unassign", params, opts)
}

// DeletePhoneNumbersParams are the parameters for the DeletePhoneNumbers method.
type DeletePhoneNumbersParams struct {
	PhoneNumbers []string `json:"phoneNumbers"`
}

// DeletePhoneNumbers releases phone numbers (marks as RELEASED). This is irreversible.
func (s *NumbersService) DeletePhoneNumbers(ctx context.Context, params DeletePhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "DELETE", "/number", params, opts)
}

// DeactivatePhoneNumbersParams are the parameters for the DeactivatePhoneNumbers method.
type DeactivatePhoneNumbersParams struct {
	PhoneNumbers []string `json:"phoneNumbers"`
}

// DeactivatePhoneNumbers deactivates one or more phone numbers. Numbers must be in READY status.
func (s *NumbersService) DeactivatePhoneNumbers(ctx context.Context, params DeactivatePhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/number/deactivate", params, opts)
}

// ReactivatePhoneNumbersParams are the parameters for the ReactivatePhoneNumbers method.
type ReactivatePhoneNumbersParams struct {
	PhoneNumbers []string `json:"phoneNumbers"`
}

// ReactivatePhoneNumbers reactivates one or more previously deactivated phone numbers.
func (s *NumbersService) ReactivatePhoneNumbers(ctx context.Context, params ReactivatePhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	if len(params.PhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "phoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/number/reactivate", params, opts)
}

// TransferPhoneNumbersParams are the parameters for the TransferPhoneNumbers method.
type TransferPhoneNumbersParams struct {
	PhoneNumbers  []string `json:"phoneNumbers"`
	NewSubgroupID string   `json:"newSubgroupId"`
}

// TransferPhoneNumbers transfers unassigned phone numbers to a different subgroup.
func (s *NumbersService) TransferPhoneNumbers(ctx context.Context, params TransferPhoneNumbersParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/number/transfer", params, opts)
}

// SearchNpaNxxParams are the parameters for the SearchNpaNxx method.
// At least one search parameter is required.
// Country may scope NPA/NXX filters; State and City cannot be combined with them.
type SearchNpaNxxParams struct {
	Country string
	State   string
	City    string
	NPA     string
	NXX     string
}

// SearchNpaNxx searches NPA/NXX lookup data.
func (s *NumbersService) SearchNpaNxx(ctx context.Context, params SearchNpaNxxParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"country": params.Country,
		"state":   params.State,
		"city":    params.City,
		"npa":     params.NPA,
		"nxx":     params.NXX,
	})
	return s.client.doJSON(ctx, "GET", "/number/lookup"+qs, nil, opts)
}

// LookupLocationsEntry represents a single NPA/NXX pair for batch location lookup.
type LookupLocationsEntry struct {
	NPA string `json:"npa"`
	NXX string `json:"nxx"`
}

// LookupLocationsParams are the parameters for the LookupLocations method.
type LookupLocationsParams struct {
	Entries []LookupLocationsEntry `json:"entries"`
}

// LookupLocations performs a batch lookup of city/state for NPA/NXX pairs (max 50).
func (s *NumbersService) LookupLocations(ctx context.Context, params LookupLocationsParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/number/lookup/location", params, opts)
}

// GetPortRequestByIdParams are the parameters for the GetPortRequestById method.
type GetPortRequestByIdParams struct {
	PortingID string
}

// GetPortRequestById retrieves a single port-in request by its ID.
func (s *NumbersService) GetPortRequestById(ctx context.Context, params GetPortRequestByIdParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"portingId": params.PortingID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/portin/%s", url.PathEscape(params.PortingID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// GetPortingRequestsParams are the parameters for the GetPortingRequests method.
type GetPortingRequestsParams struct {
	PortingID string
	Page      string
	Limit     string
}

// GetPortingRequests retrieves port-in requests. If PortingID is provided, retrieves a single request by ID.
// Otherwise, retrieves a paginated list of port-in requests.
func (s *NumbersService) GetPortingRequests(ctx context.Context, params GetPortingRequestsParams, opts *RequestOptions) (*Response, error) {
	if params.PortingID != "" {
		return s.GetPortRequestById(ctx, GetPortRequestByIdParams{PortingID: params.PortingID}, opts)
	}
	return s.GetPortRequests(ctx, GetPortRequestsParams{Page: params.Page, Limit: params.Limit}, opts)
}

// ChangePortingStatusParams are the parameters for the ChangePortingStatus admin method.
// Status is one of PENDING, IN_PROGRESS, ON_HOLD, REJECTED. OnHoldReason is required when
// Status is ON_HOLD (DOCUMENTATION_MISMATCH, ACCOUNT_INFO_INCORRECT, CARRIER_REJECTED,
// PENDING_CUSTOMER_ACTION, OTHER) — Infobip exposes no porting-status API or webhook, so this
// is always staff-entered.
type ChangePortingStatusParams struct {
	PortingID    string `json:"-"`
	Status       string `json:"status"`
	Description  string `json:"description"`
	OnHoldReason string `json:"onHoldReason,omitempty"`
}

// ChangePortingStatus changes the status of a port-in request. Admin only.
func (s *NumbersAdminService) ChangePortingStatus(ctx context.Context, params ChangePortingStatusParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"portingId": params.PortingID, "status": params.Status}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/portin/%s/status", url.PathEscape(params.PortingID))
	body := map[string]string{
		"status":      params.Status,
		"description": params.Description,
	}
	if params.OnHoldReason != "" {
		body["onHoldReason"] = params.OnHoldReason
	}
	return s.client.doJSON(ctx, "PUT", path, body, opts)
}

// ApprovePortRequestParams are the parameters for the ApprovePortRequest admin method.
type ApprovePortRequestParams struct {
	PortingID string
}

// ApprovePortRequest approves a port-in request. Admin only.
func (s *NumbersAdminService) ApprovePortRequest(ctx context.Context, params ApprovePortRequestParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"portingId": params.PortingID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/number/portin/%s/approve", url.PathEscape(params.PortingID))
	return s.client.doJSON(ctx, "POST", path, nil, opts)
}

// GetShortCodeAcquisitionRequestsParams are filters for staff-only Short Code acquisition requests.
type GetShortCodeAcquisitionRequestsParams struct {
	Page   string
	Limit  string
	Status string
	Search string
}

// GetShortCodeAcquisitionRequests lists campaign-bound Short Code acquisition requests. Admin only.
func (s *NumbersAdminService) GetShortCodeAcquisitionRequests(ctx context.Context, params GetShortCodeAcquisitionRequestsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{"page": params.Page, "limit": params.Limit, "status": params.Status, "search": params.Search})
	return s.client.doJSON(ctx, "GET", "/number/short-code/requests"+qs, nil, opts)
}

// MigratePhoneNumberToV2Params are the parameters for the MigratePhoneNumberToV2 method.
type MigratePhoneNumberToV2Params struct {
	PhoneNumber string `json:"phoneNumber,omitempty"`
	CampaignID  string `json:"campaignId,omitempty"`
	GroupID     string `json:"groupId,omitempty"`
}

// MigratePhoneNumberToV2 migrates a phone number (or all numbers in a group) from v1 to v2.
func (s *NumbersService) MigratePhoneNumberToV2(ctx context.Context, params MigratePhoneNumberToV2Params, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/number/migrate", params, opts)
}
