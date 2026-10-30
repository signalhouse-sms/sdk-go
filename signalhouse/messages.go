package signalhouse

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/url"
)

// MessagesService handles message-related API calls.
//
// Date-range parameters (StartDate, EndDate) on list and analytics methods
// accept ISO-8601 date or timestamp strings, bound per DateBounds. A bare date is
// always that whole UTC day. A timestamp is widened to its whole UTC day in "day"
// mode and bound as the instant it names, in whatever
// offset it was written ("2026-09-14T00:00:00-04:00" is 04:00Z), in "exact" mode,
// where day buckets also align to the offset EndDate carries. Omitted, DateBounds is "day".
// Business-number filters and SMS/MMS senders accept digits-only 5-6 digit
// Short Codes or 10+ digit long numbers. Recipient numbers remain 10+ digits.
type MessagesService struct {
	client *Client
}

// GetMessagesParams are the parameters for the GetMessages method.
type GetMessagesParams struct {
	ID         string
	CampaignID string
	// RegistrationID filters to messages attributed to these registrations. Cannot be combined with
	// CampaignID; each id must belong to the group. Single value or comma-separated list.
	RegistrationID string
	BrandID        string
	SubgroupID     string
	GroupID        string
	PhoneNumber    string
	// SenderPhoneNumber accepts a plain digits-only exact value, or a value containing % as a
	// wildcard for a partial match (e.g. "%1234" matches numbers ending in 1234). 3-15 digits
	// required alongside any %.
	SenderPhoneNumber string
	// RecipientPhoneNumber accepts a plain digits-only exact value, or a value containing % as a
	// wildcard for a partial match (e.g. "%1234" matches numbers ending in 1234). 3-15 digits
	// required alongside any %.
	RecipientPhoneNumber string
	Status               string
	Direction            string
	// MessageType accepts a single value (e.g. "SMS") or a comma-separated list (e.g. "SMS,MMS").
	// Allowed values: SMS, MMS, RCS, WHATSAPP, VIBER, P2P. Empty defaults to all non-P2P types.
	MessageType string
	// Channel filters by "tenDLC", "virtualLongCode", "tollFree", "shortCode", or "p2p" (single value or comma-separated list).
	// Older messages without a stored channel count as tenDLC.
	Channel string
	Carrier string
	// SentimentLabel filters by sentiment bucket: "positive", "neutral", or "negative" (single value or
	// comma-separated list). Applied as the sentimentScore range the label threshold produces, so unscored
	// messages match no bucket and outbound messages (never scored) are excluded.
	SentimentLabel string
	// ErrorCode filters FAILED messages by Signal House error code (single value or comma-separated list).
	// "OUT" is the campaign opt-out, displayed as RECIPIENT_OPTED_OUT. Carrier codes are not filterable
	// here; they live inside successOrFailureReason.
	ErrorCode string
	StartDate string
	EndDate   string
	// SortField is one of "createdAt", "segmentCount", "status", or "sentimentScore".
	SortField string
	SortOrder string
	Page      string
	Limit     string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list).
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing.
	RegionScopes string
}

// GetMessages returns a list of messages with optional filters.
//
// Inbound messages in the response also carry sentimentScore (-100..100, null when unscored),
// sentimentLabel ("positive", "neutral" or "negative", null when sentimentScore is null) and
// sentimentScoredAt. Outbound messages are never scored.
func (s *MessagesService) GetMessages(ctx context.Context, params GetMessagesParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"id":                   params.ID,
		"campaignId":           params.CampaignID,
		"registrationId":       params.RegistrationID,
		"brandId":              params.BrandID,
		"subgroupId":           params.SubgroupID,
		"groupId":              params.GroupID,
		"phoneNumber":          params.PhoneNumber,
		"senderPhoneNumber":    params.SenderPhoneNumber,
		"recipientPhoneNumber": params.RecipientPhoneNumber,
		"status":               params.Status,
		"direction":            params.Direction,
		"messageType":          params.MessageType,
		"channel":              params.Channel,
		"region":               params.Region,
		"regionScopes":         params.RegionScopes,
		"carrier":              params.Carrier,
		"sentimentLabel":       params.SentimentLabel,
		"errorCode":            params.ErrorCode,
		"startDate":            params.StartDate,
		"endDate":              params.EndDate,
		"sortField":            params.SortField,
		"sortOrder":            params.SortOrder,
		"page":                 params.Page,
		"limit":                params.Limit,
		"dateBounds":           params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message"+qs, nil, opts)
}

// GetAnalyticsParams are the parameters for the GetAnalytics method.
//
// SubgroupID, BrandID, CampaignID, PhoneNumber and Carrier each accept a single value or a
// comma-separated list of values. GroupID accepts a single value only.
type GetAnalyticsParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	// CarrierFamily filters by the recipient's resolved carrier family (Default, ATT, TMobile, Verizon, USCellular, GoogleVoice, ClearSky, Interop, RogueMobile, P2P, Standard, IceWireless, Bell, Rogers, Telus, Videotron, SaskTel, Eastlink, Fibernetics);
	// single value or comma-separated list. Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400 -- omit the parameter unless it is wanted.
	CarrierFamily string
	StartDate     string
	EndDate       string
	// Channel filters by "tenDLC", "virtualLongCode", "tollFree", "shortCode", or "p2p" (single value or comma-separated list).
	// A tenDLC selection also includes older messages with no channel, and p2p is matched by carrier.
	// MessageType scopes the SENTIMENT figures to "SMS", "MMS" or "P2P" (single value or
	// comma-separated list). Empty means all scored types. It does NOT narrow the message counts,
	// which are always returned split per type and have no filterable type dimension.
	MessageType string
	Channel     string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
}

// GetAnalytics returns aggregated analytics for messages.
//
// The response also includes inbound-message sentiment for the requested scope: sentimentScore
// (volume-weighted average of every scored inbound message in range, -100..100, null when nothing
// was scored), sentimentLabel, sentimentScoredCount, sentimentPositive, sentimentNeutral and
// sentimentNegative. sentimentScore is an average, so it must never be summed across responses.
func (s *MessagesService) GetAnalytics(ctx context.Context, params GetAnalyticsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"carrierFamily":  params.CarrierFamily,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"messageType":    params.MessageType,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics"+qs, nil, opts)
}

// GetAnalyticsDetailParams are the parameters for the GetAnalyticsDetail method.
//
// SubgroupID, BrandID, CampaignID, PhoneNumber and Carrier each accept a single value or a
// comma-separated list of values. GroupID accepts a single value only.
type GetAnalyticsDetailParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	// CarrierFamily filters by the recipient's resolved carrier family (Default, ATT, TMobile, Verizon, USCellular, GoogleVoice, ClearSky, Interop, RogueMobile, P2P, Standard, IceWireless, Bell, Rogers, Telus, Videotron, SaskTel, Eastlink, Fibernetics);
	// single value or comma-separated list. Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400 -- omit the parameter unless it is wanted.
	CarrierFamily string
	StartDate     string
	EndDate       string
	// Channel filters by "tenDLC", "virtualLongCode", "tollFree", "shortCode", or "p2p" (single value or comma-separated list).
	// A tenDLC selection also includes older messages with no channel, and p2p is a real channel here.
	// MessageType scopes the SENTIMENT figures to "SMS", "MMS" or "P2P" (single value or
	// comma-separated list). Empty means all scored types. It does NOT narrow the message counts,
	// which are always returned split per type and have no filterable type dimension.
	MessageType string
	Channel     string
	// Granularity is the time-bucket grain of the byDate rows: "day" or "hour". Empty means day (each
	// row's _id is a "YYYY-MM-DD" date). "hour" buckets by hour (each row's _id is a
	// "YYYY-MM-DD HH:MM:SS" UTC hour; StartDate is floored to its hour and EndDate honoured as given)
	// and is capped to a 7-day span. Only accepted where the environment serves hourly analytics;
	// elsewhere any value, including "day", is rejected with a 400 -- leave it empty unless hour
	// grain is wanted.
	Granularity string
	// Breakdown is "carrier" or "carrierFamily". It adds a byBreakdown time × dimension series to the
	// response ({dimension, rows}: each row is one byDate bucket (_id) for one carrier or carrier family
	// (key), top 50 keys by volume, excluding P2P) and caps the span at 31 days. Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400 -- omit the parameter unless it is wanted.
	Breakdown string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
}

// GetAnalyticsDetail returns detailed analytics snapshot records for charting and aggregation.
//
// cardTotals and every byDate/byPhoneNumber/byCarrier row also carry that row's own inbound-message
// sentiment: sentimentScore (volume-weighted average, -100..100, null when nothing in that row's
// scope was scored), sentimentLabel, sentimentScoredCount, sentimentPositive, sentimentNeutral and
// sentimentNegative. The fields are always present; sentimentScore is an average and must never be
// summed across rows or responses.
func (s *MessagesService) GetAnalyticsDetail(ctx context.Context, params GetAnalyticsDetailParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"carrierFamily":  params.CarrierFamily,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"messageType":    params.MessageType,
		"breakdown":      params.Breakdown,
		"granularity":    params.Granularity,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics/detail"+qs, nil, opts)
}

// GetAnalyticsThroughputParams are the parameters for the GetAnalyticsThroughput method.
//
// GroupID, StartDate and EndDate are required. BrandID, CampaignID and CarrierFamily each accept a
// single value or a comma-separated list; CampaignID wins over BrandID.
type GetAnalyticsThroughputParams struct {
	GroupID    string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). Single value or comma-separated list.
	RegistrationID string
	// CarrierFamily filters by carrier family (ATT, TMobile, Verizon, USCellular, ...).
	CarrierFamily string
	// StartDate is ISO-8601, bound per dateBounds: a bare date is always that whole UTC day (inclusive); a timestamp is widened to its whole UTC day in "day" mode and bound as the instant it names, in whatever offset it was written, in "exact" mode,
	// then floored to the start of its hour at hour granularity and of its minute at day granularity
	// (the cube is keyed by minute).
	StartDate string
	// EndDate is ISO-8601. At hour granularity it is honoured as given; at day granularity it is
	// bound per dateBounds: a bare date is always that whole UTC day (inclusive); a timestamp is widened to its whole UTC day in "day" mode and bound as the instant it names, in whatever offset it was written, in "exact" mode. In "exact" mode day buckets align to the UTC offset this bound carries, so a client that sends its own midnights gets its own calendar dates back. The span is capped at 35 days.
	EndDate string
	// Granularity is the bucket grain of the byDate and byBreakdown rows: "hour" (default; span capped
	// at 7 days) or "day".
	Granularity string
	// Ceiling is a segments-per-minute ceiling. When set, every row also reports minutesAtCeiling
	// (sending minutes at or above it); when nil minutesAtCeiling is null.
	Ceiling *int
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
}

// GetAnalyticsThroughput returns per-minute send throughput (peak segments per minute, sending
// minutes, minutes at a ceiling) per time bucket and per carrier family. Measured from SENT
// transitions at campaign × carrier-family grain — the grain carrier ceilings are enforced at — so
// a peak here is the figure a ceiling is compared against. Only accepted where the environment
// serves hourly analytics; elsewhere the request is rejected with a 400.
//
// Response: totals, byDate, byCarrierFamily and byBreakdown ({dimension: "carrierFamily", rows}).
// Every row carries segments, messages, peakSegmentsPerMinute, peakMessagesPerMinute, peakMinute
// (UTC "YYYY-MM-DD HH:MM:SS", null when no traffic), sendingMinutes,
// averageSegmentsPerSendingMinute and minutesAtCeiling.
func (s *MessagesService) GetAnalyticsThroughput(ctx context.Context, params GetAnalyticsThroughputParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"groupId": params.GroupID, "startDate": params.StartDate, "endDate": params.EndDate}); err != nil {
		return nil, err
	}

	ceiling := ""
	if params.Ceiling != nil {
		ceiling = fmt.Sprintf("%d", *params.Ceiling)
	}

	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"carrierFamily":  params.CarrierFamily,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"granularity":    params.Granularity,
		"ceiling":        ceiling,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics/throughput"+qs, nil, opts)
}

// GetAnalyticsFilterOptionsParams are the parameters for the GetAnalyticsFilterOptions method.
type GetAnalyticsFilterOptionsParams struct {
	GroupID string
	// Region restricts the options to "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated
	// list). Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
}

// GetAnalyticsFilterOptions returns filter dropdown options (subgroups, brands, campaigns, phone numbers,
// carriers) for the Analytics page, scoped to a single group. Only items with at
// least one message are returned. Each phone number carries type (TOLL_FREE, SHORT_CODE, ALPHANUMERIC,
// VIRTUAL_LONG_NUMBER, ...) and country (ISO-2), null when unknown: TOLL_FREE, SHORT_CODE and ALPHANUMERIC
// are channels by type; any other US number is 10DLC and a non-US one is a Virtual Long Code. Carriers is
// a plain array of carrier name strings.
func (s *MessagesService) GetAnalyticsFilterOptions(ctx context.Context, params GetAnalyticsFilterOptionsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId": params.GroupID,
		"region":  params.Region,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics/filter-options"+qs, nil, opts)
}

// GetAnalyticsBySubgroupParams are the parameters for the GetAnalyticsBySubgroup method.
// Limit is capped server-side at 50 per page.
// Channel: "both" (default when empty, all activity) or "tenDLC" / "virtualLongCode" / "tollFree" / "shortCode" / "p2p" (single value
// or comma-separated list) — scopes ORDER BY + row inclusion by the stored channel column so a
// single-channel caller doesn't get pages dominated by other channels; toll-free is kept separate
// from 10DLC.
type GetAnalyticsBySubgroupParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	// CarrierFamily filters by the recipient's resolved carrier family (Default, ATT, TMobile, Verizon, USCellular, GoogleVoice, ClearSky, Interop, RogueMobile, P2P, Standard, IceWireless, Bell, Rogers, Telus, Videotron, SaskTel, Eastlink, Fibernetics);
	// single value or comma-separated list. Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400 -- omit the parameter unless it is wanted.
	CarrierFamily string
	StartDate     string
	EndDate       string
	Page          string
	Limit         string
	Channel       string
	// MessageType scopes ONLY the sentiment figures (SMS, MMS or P2P; comma-separated for several).
	// Omit for all scored types. It does not narrow the message counts, which are always returned
	// split per type and have no filterable type dimension.
	MessageType string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
}

// GetAnalyticsBySubgroup returns a paginated breakdown of message metrics grouped by subgroup.
// Each row carries the full set of sms/mms/p2p metric columns plus smsOptOuts/mmsOptOuts (sourced
// from the DNC analytics MV — P2P has no opt-outs) so callers can apply channel toggles client-side.
// Each row also carries that subgroup's inbound-message sentiment: sentimentScore (volume-weighted,
// -100..100, null when nothing was scored), sentimentLabel, sentimentScoredCount and the
// positive/neutral/negative counts, plus the same six per message type under smsSentiment*,
// mmsSentiment* and p2pSentiment*. The score is derived server-side; combine counts across rows,
// never the scores.
func (s *MessagesService) GetAnalyticsBySubgroup(ctx context.Context, params GetAnalyticsBySubgroupParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"carrierFamily":  params.CarrierFamily,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"page":           params.Page,
		"limit":          params.Limit,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"messageType":    params.MessageType,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics/by-subgroup"+qs, nil, opts)
}

// GetAnalyticsByErrorCodeParams are the parameters for the GetAnalyticsByErrorCode method.
// Limit is capped server-side at 50 per page.
// Channel: "both" (default when empty, every code) or "tenDLC" / "virtualLongCode" / "tollFree" / "shortCode" / "p2p" (single value
// or comma-separated list) — scopes ORDER BY + totalCount by the stored channel column; when one
// channel is selected, totalErrors reflects only that channel.
type GetAnalyticsByErrorCodeParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	// CarrierFamily filters by the recipient's resolved carrier family (Default, ATT, TMobile, Verizon, USCellular, GoogleVoice, ClearSky, Interop, RogueMobile, P2P, Standard, IceWireless, Bell, Rogers, Telus, Videotron, SaskTel, Eastlink, Fibernetics);
	// single value or comma-separated list. Only accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400 -- omit the parameter unless it is wanted.
	CarrierFamily string
	StartDate     string
	EndDate       string
	Page          string
	Limit         string
	Channel       string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
	// ErrorCode narrows the breakdown to a single failure reason: a numeric Infobip carrier code (as a
	// string, e.g. "607"), the Signal House campaign-opt-out code "OUT", or a no-code failure class
	// (other, moderation, att_limit, p2p_stop, campaign_opt_out, dnc_blocked, bind_hold_expired). Empty:
	// every reason.
	ErrorCode string
}

// GetAnalyticsByErrorCode returns a paginated breakdown of failed messages grouped by error code.
// Each row contains per-channel (sms/mms/p2p) error counts plus an enriched description.
func (s *MessagesService) GetAnalyticsByErrorCode(ctx context.Context, params GetAnalyticsByErrorCodeParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"carrierFamily":  params.CarrierFamily,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"page":           params.Page,
		"limit":          params.Limit,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"dateBounds":     params.DateBounds,
		"errorCode":      params.ErrorCode,
	})
	return s.client.doJSON(ctx, "GET", "/message/analytics/by-error-code"+qs, nil, opts)
}

// GetDncAnalyticsParams are the parameters for the GetDncAnalytics method.
type GetDncAnalyticsParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	StartDate      string
	EndDate        string
	// Channel filters by "tenDLC", "virtualLongCode", "tollFree", or "shortCode" (single value or comma-separated list). Opt-outs are
	// A2P-only, so "p2p" applies no filter, and a tenDLC selection also includes older opt-outs with
	// no channel.
	Channel string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
}

// GetDncAnalytics returns aggregated DNC (Do Not Contact) opt-out analytics: totals, byDate,
// byPhoneNumber, bySenderNumber, byCampaign, bySubgroup, byType, byCarrier, byKeyword. bySenderNumber
// ({senderPhoneNumber, total, sms, mms}) attributes each opt-out to the first business number that
// received it, so it is unchanged by later campaign moves and START replies. byCampaign counts
// campaign-scoped opt-outs only, by the campaign at the time of the STOP; subgroup-scoped opt-outs
// (non-US recipients) are never attributed to a campaign and appear under bySubgroup instead. byType
// splits SMS/MMS. byKeyword breaks opt-outs down by the normalized keyword
// that revoked consent ({optOutKeyword, total, sms, mms}); opt-outs recorded before keyword capture
// shipped appear as a single optOutKeyword: null bucket ("not recorded"). The distinct keyword set
// is open-ended (campaigns register their own keywords beyond the mandatory floor) — render top-N
// plus Other, never a fixed list.
func (s *MessagesService) GetDncAnalytics(ctx context.Context, params GetDncAnalyticsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/dnc/analytics"+qs, nil, opts)
}

// GetDncRecordsParams are the parameters for the GetDncRecords method.
type GetDncRecordsParams struct {
	GroupID    string
	SubgroupID string
	BrandID    string
	CampaignID string
	// RegistrationID filters to traffic attributed to these registrations (hourly analytics only; a 400
	// while hourly reads are off). A PhoneNumber filter takes precedence. Single value or comma-separated list.
	RegistrationID string
	PhoneNumber    string
	Carrier        string
	StartDate      string
	EndDate        string
	Page           string
	Limit          string
	SortField      string
	SortOrder      string
	// Channel filters by "tenDLC", "virtualLongCode", "tollFree", or "shortCode" (single value or comma-separated list). Opt-outs are
	// A2P-only, so "p2p" applies no filter, and a tenDLC selection also includes older opt-outs with
	// no channel.
	Channel string
	// DateBounds is which contract binds StartDate/EndDate. "day": each bound is widened to its whole
	// UTC day. "exact": a timestamp is bound as the instant it names and a
	// bare date is its whole UTC day. Empty: "day".
	DateBounds string
	// Region filters by "US", "GB", or "CA" (any case of "UK"/"GB"/"GBR"/"United Kingdom" is accepted for GB; single value or comma-separated list). Only
	// accepted where the environment serves hourly analytics; elsewhere any value is rejected with a 400.
	Region string
	// RegionScopes is a JSON array of OR-ed branches {region, channels, brandId?, campaignId?, phoneNumber?}
	// (channels: tenDLC, virtualLongCode, tollFree, shortCode, p2p); "[]" matches nothing. Same hourly-analytics
	// gate as Region.
	RegionScopes string
}

// GetDncRecords returns paginated Do Not Call records. Each record carries senderPhoneNumber — the
// first business number that received the opt-out (set once, never changed by a later STOP or a
// campaign move; the same value as phoneNumber), campaignId (null on subgroup-scoped records for
// non-US recipients), optOutKeyword — the
// normalized keyword that revoked consent (e.g. "stop", "opt out"; a repeat opt-out records the
// most recent word), null for opt-outs recorded before keyword capture shipped — and carrierFamily,
// the resolved carrier family of the recipient number (Default, ATT, TMobile, Verizon, USCellular,
// GoogleVoice, ClearSky, Interop, RogueMobile, Standard, IceWireless, Bell, Rogers, Telus, Videotron, SaskTel, Eastlink, Fibernetics), copied from the inbound message that triggered the
// opt-out.
func (s *MessagesService) GetDncRecords(ctx context.Context, params GetDncRecordsParams, opts *RequestOptions) (*Response, error) {
	qs := buildQueryString(map[string]string{
		"groupId":        params.GroupID,
		"subgroupId":     params.SubgroupID,
		"brandId":        params.BrandID,
		"campaignId":     params.CampaignID,
		"registrationId": params.RegistrationID,
		"phoneNumber":    params.PhoneNumber,
		"carrier":        params.Carrier,
		"startDate":      params.StartDate,
		"endDate":        params.EndDate,
		"page":           params.Page,
		"limit":          params.Limit,
		"sortField":      params.SortField,
		"sortOrder":      params.SortOrder,
		"channel":        params.Channel,
		"region":         params.Region,
		"regionScopes":   params.RegionScopes,
		"dateBounds":     params.DateBounds,
	})
	return s.client.doJSON(ctx, "GET", "/message/dnc"+qs, nil, opts)
}

// SendSMSParams are the parameters for the SendSMS method.
type SendSMSParams struct {
	SenderPhoneNumber                 string   `json:"senderPhoneNumber"`
	RecipientPhoneNumbers             []string `json:"recipientPhoneNumber"`
	MessageBody                       string   `json:"messageBody"`
	StatusCallbackURL                 string   `json:"statusCallbackUrl,omitempty"`
	EnableShortlink                   bool     `json:"enableShortlink"`
	FilterLandlinesAndInactiveNumbers bool     `json:"filterLandlinesAndInactiveNumbers,omitempty"`
}

// SendSMS sends an SMS message to one or more recipients. SenderPhoneNumber accepts a digits-only
// 5-6 digit Short Code or 10+ digit long number, or a READY alphanumeric sender ID (e.g. "ACME"; UK
// recipients and SMS only). Every message from an alphanumeric sender gets "\nOpt out: sihou.io/o/{code}"
// appended by the server, counted in segments. A Short Code sender permits exactly one recipient,
// and it must be a +1 (US/Canada) number; a bare 10-digit number is treated as +1. The API rejects
// Short Code sends with multiple recipients or a non-+1 recipient.
//
// The response reports the send outcome via requestedRecipientCount, enqueuedCount, and failedCount.
// enqueuedCount is the number of recipients accepted for delivery and is 0 when the send was fully
// blocked (by the moderation filter or a campaign opt-out); the request still succeeds in that case,
// so treat enqueuedCount, not the HTTP status, as the signal that a message was accepted. Blocked
// recipients are returned in insertedMessages with status "FAILED"; recipients on the global Do Not
// Contact list are reported in dncBlockedNumbers with no message record. Each FAILED message carries
// errorCode, a stable machine-readable cause (campaign opt-out is "OUT"; null on success or on
// failures with no assigned code) that is also included on the FAILED status-callback payload.
func (s *MessagesService) SendSMS(ctx context.Context, params SendSMSParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"senderPhoneNumber": params.SenderPhoneNumber,
		"messageBody":       params.MessageBody,
	}); err != nil {
		return nil, err
	}
	if len(params.RecipientPhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "recipientPhoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/message/sms", params, opts)
}

// SendP2PParams are the parameters for the SendP2P method.
type SendP2PParams struct {
	RecipientPhoneNumbers []string `json:"recipientPhoneNumber"`
	MessageBody           string   `json:"messageBody"`
	StatusCallbackURL     string   `json:"statusCallbackUrl,omitempty"`
	// GroupID is optional. Explicit group to send from; defaults to the group in your auth context.
	GroupID string `json:"groupId,omitempty"`
	// SubgroupID is optional. Explicit subgroup to attribute the send (and its inbound reply) to. P2P has
	// no dedicated sender number, so pass this to record the send against a specific subgroup; it also
	// resolves the group. Defaults to the group's oldest subgroup.
	SubgroupID string `json:"subgroupId,omitempty"`
	// UseSignalHouseShortlinks is optional. When false, SignalHouse applies no
	// link-shortening or text-spin and your pre-spun links/content are sent verbatim
	// (bring-your-own spinner). Defaults to true. Pointer so the literal false is sent
	// while a nil pointer omits the field and preserves the server default.
	UseSignalHouseShortlinks *bool `json:"useSignalHouseShortlinks,omitempty"`
}

// SendP2P sends a P2P message via Rogue Mobile SMPP.
func (s *MessagesService) SendP2P(ctx context.Context, params SendP2PParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"messageBody": params.MessageBody,
	}); err != nil {
		return nil, err
	}
	if len(params.RecipientPhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "recipientPhoneNumbers"}
	}
	return s.client.doJSON(ctx, "POST", "/message/p2p", params, opts)
}

// GetP2PBatchParams are the parameters for the GetP2PBatch method.
type GetP2PBatchParams struct {
	BatchID string
}

// GetP2PBatch returns a P2P batch by its ID.
func (s *MessagesService) GetP2PBatch(ctx context.Context, params GetP2PBatchParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{"batchId": params.BatchID}); err != nil {
		return nil, err
	}
	path := fmt.Sprintf("/message/p2p/batches/%s", url.PathEscape(params.BatchID))
	return s.client.doJSON(ctx, "GET", path, nil, opts)
}

// GetDeliveryReportsParams are the parameters for the GetDeliveryReports method.
type GetDeliveryReportsParams struct {
	MessageIDs []string `json:"messageIds"`
}

// GetDeliveryReports fetches the latest upstream carrier delivery report for a set of messages by ID,
// useful for reconciling messages stuck in a non-terminal state. Outbound SMS and MMS messages sent from
// numbers on supported carriers are looked up upstream (reports are retained about 48 hours);
// every other message, including P2P messages and messages sent from numbers on other carriers, is returned
// with its stored status and upstreamReportAvailable false. The request fails with a 500 when the sending
// numbers can't be read.
func (s *MessagesService) GetDeliveryReports(ctx context.Context, params GetDeliveryReportsParams, opts *RequestOptions) (*Response, error) {
	if len(params.MessageIDs) == 0 {
		return nil, &ValidationError{Field: "messageIds"}
	}
	return s.client.doJSON(ctx, "POST", "/message/delivery-report", params, opts)
}

// SendMMSParams are the parameters for the SendMMS method.
type SendMMSParams struct {
	SenderPhoneNumber                 string        `json:"-"`
	RecipientPhoneNumbers             []string      `json:"-"`
	MessageBody                       string        `json:"-"`
	MediaURLs                         []string      `json:"-"`
	StatusCallbackURL                 string        `json:"-"`
	EnableShortlink                   bool          `json:"-"`
	EnableCompression                 bool          `json:"-"`
	FilterLandlinesAndInactiveNumbers bool          `json:"-"`
	Images                            []*FileUpload `json:"-"`
}

// SendMMS sends an MMS message with optional media attachments. SenderPhoneNumber accepts a
// digits-only 5-6 digit Short Code or 10+ digit long number. A Short Code sender permits exactly
// one recipient, and it must be a +1 (US/Canada) number; a bare 10-digit number is treated as +1.
// The API rejects Short Code sends with multiple recipients or a non-+1 recipient.
//
// The response reports the send outcome via requestedRecipientCount, enqueuedCount, and failedCount.
// enqueuedCount is the number of recipients accepted for delivery and is 0 when the send was fully
// blocked (by the moderation filter or a campaign opt-out); the request still succeeds in that case,
// so treat enqueuedCount, not the HTTP status, as the signal that a message was accepted. Blocked
// recipients are returned in insertedMessages with status "FAILED"; recipients on the global Do Not
// Contact list are reported in dncBlockedNumbers with no message record. Each FAILED message carries
// errorCode, a stable machine-readable cause (campaign opt-out is "OUT"; null on success or on
// failures with no assigned code) that is also included on the FAILED status-callback payload.
func (s *MessagesService) SendMMS(ctx context.Context, params SendMMSParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"senderPhoneNumber": params.SenderPhoneNumber,
	}); err != nil {
		return nil, err
	}
	if len(params.RecipientPhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "recipientPhoneNumbers"}
	}

	writeForm := func(w *multipart.Writer) error {
		return writeMMSForm(w, params.SenderPhoneNumber, params.RecipientPhoneNumbers,
			params.MessageBody, params.MediaURLs, params.StatusCallbackURL,
			params.EnableShortlink, params.EnableCompression, params.FilterLandlinesAndInactiveNumbers, params.Images)
	}

	return s.client.doMultipart(ctx, "POST", "/message/mms", writeForm, opts)
}

// SendGroupMessageParams are the parameters for the SendGroupMessage method.
type SendGroupMessageParams struct {
	SenderPhoneNumber                 string        `json:"-"`
	RecipientPhoneNumbers             []string      `json:"-"`
	MessageBody                       string        `json:"-"`
	MediaURLs                         []string      `json:"-"`
	StatusCallbackURL                 string        `json:"-"`
	EnableShortlink                   bool          `json:"-"`
	EnableCompression                 bool          `json:"-"`
	FilterLandlinesAndInactiveNumbers bool          `json:"-"`
	Images                            []*FileUpload `json:"-"`
}

// SendGroupMessage sends a group MMS message with optional media attachments. Short Codes do not
// support group messaging; SenderPhoneNumber must be a long number.
//
// The response reports the send outcome via requestedRecipientCount, enqueuedCount, and failedCount.
// enqueuedCount is the number of recipients accepted for delivery and is 0 when the send was fully
// blocked (by the moderation filter or a campaign opt-out); the request still succeeds in that case,
// so treat enqueuedCount, not the HTTP status, as the signal that a message was accepted. Blocked
// recipients are returned in insertedMessages with status "FAILED"; recipients on the global Do Not
// Contact list are reported in dncBlockedNumbers with no message record. Each FAILED message carries
// errorCode, a stable machine-readable cause (campaign opt-out is "OUT"; null on success or on
// failures with no assigned code) that is also included on the FAILED status-callback payload.
func (s *MessagesService) SendGroupMessage(ctx context.Context, params SendGroupMessageParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"senderPhoneNumber": params.SenderPhoneNumber,
	}); err != nil {
		return nil, err
	}
	if len(params.RecipientPhoneNumbers) == 0 {
		return nil, &ValidationError{Field: "recipientPhoneNumbers"}
	}

	writeForm := func(w *multipart.Writer) error {
		return writeMMSForm(w, params.SenderPhoneNumber, params.RecipientPhoneNumbers,
			params.MessageBody, params.MediaURLs, params.StatusCallbackURL,
			params.EnableShortlink, params.EnableCompression, params.FilterLandlinesAndInactiveNumbers, params.Images)
	}

	return s.client.doMultipart(ctx, "POST", "/message/groupMessage", writeForm, opts)
}

// CarrierIdLookupParams are the parameters for the CarrierIdLookup method.
type CarrierIdLookupParams struct {
	PhoneNumber string `json:"phoneNumber"`
}

// CarrierIdLookup looks up the carrier for a phone number. Always performs a fresh lookup (not cached). Billed per request.
func (s *MessagesService) CarrierIdLookup(ctx context.Context, params CarrierIdLookupParams, opts *RequestOptions) (*Response, error) {
	if err := require(map[string]string{
		"phoneNumber": params.PhoneNumber,
	}); err != nil {
		return nil, err
	}
	return s.client.doJSON(ctx, "POST", "/message/carrier-id", params, opts)
}

// writeMMSForm writes MMS/GroupMessage form fields to a multipart writer.
func writeMMSForm(w *multipart.Writer, senderPhone string, recipientPhones []string,
	messageBody string, mediaURLs []string, statusCallbackURL string,
	enableShortlink, enableCompression, filterLandlinesAndInactiveNumbers bool, images []*FileUpload) error {

	// Add images
	for _, img := range images {
		if img != nil {
			part, err := w.CreateFormFile("images", img.Filename)
			if err != nil {
				return err
			}
			if _, err := io.Copy(part, img.Reader); err != nil {
				return err
			}
		}
	}

	// Add form fields
	if err := w.WriteField("senderPhoneNumber", senderPhone); err != nil {
		return err
	}

	recipientJSON, err := json.Marshal(recipientPhones)
	if err != nil {
		return err
	}
	if err := w.WriteField("recipientPhoneNumber", string(recipientJSON)); err != nil {
		return err
	}

	if err := w.WriteField("messageBody", messageBody); err != nil {
		return err
	}

	if len(mediaURLs) > 0 {
		urlsJSON, err := json.Marshal(mediaURLs)
		if err != nil {
			return err
		}
		if err := w.WriteField("mediaUrls", string(urlsJSON)); err != nil {
			return err
		}
	}

	if statusCallbackURL != "" {
		if err := w.WriteField("statusCallbackUrl", statusCallbackURL); err != nil {
			return err
		}
	}

	if err := w.WriteField("enableShortlink", fmt.Sprintf("%t", enableShortlink)); err != nil {
		return err
	}

	if err := w.WriteField("enableCompression", fmt.Sprintf("%t", enableCompression)); err != nil {
		return err
	}

	if filterLandlinesAndInactiveNumbers {
		if err := w.WriteField("filterLandlinesAndInactiveNumbers", "true"); err != nil {
			return err
		}
	}

	return nil
}

// EstimateMessageParams describes a Canadian or UK retail-cost estimate, bounded to 100 recipients. A Ready
// Canadian sender takes SMS or MMS to Canadian or US recipients; a UK (+44) virtual long code takes SMS only to
// GB mobile recipients (10-15 digits). The response region is "CA" or "GB". Unavailable unless that region is
// enabled for the environment. SenderPhoneNumber may also be a READY alphanumeric sender ID (e.g. "ACME"; UK
// recipients, SMS only); its estimate counts the "\nOpt out: sihou.io/o/{code}" footer the server appends.
type EstimateMessageParams struct {
	MessageType           string   `json:"messageType,omitempty"`
	SenderPhoneNumber     string   `json:"senderPhoneNumber"`
	RecipientPhoneNumbers []string `json:"recipientPhoneNumbers"`
	MessageBody           string   `json:"messageBody"`
}

// EstimateMessage returns retail rates and total in microdollars; it never sends or charges.
func (s *MessagesService) EstimateMessage(ctx context.Context, params EstimateMessageParams, opts *RequestOptions) (*Response, error) {
	return s.client.doJSON(ctx, "POST", "/message/estimate", params, opts)
}
