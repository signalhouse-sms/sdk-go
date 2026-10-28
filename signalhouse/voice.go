package signalhouse

// VoiceService groups Voice service resources under one namespace. Access
// via client.Voice.SipTrunks, client.Voice.SipProfiles, client.Voice.Calls,
// and client.Voice.Tokens.
//
// Routes live under /voice/* on the Voice service.
type VoiceService struct {
	client *Client

	// SipTrunks — peer-to-peer SIP connections to a PBX or carrier.
	SipTrunks *VoiceSipTrunksService

	// SipProfiles — single registerable SIP endpoints (desk phones, softphones, apps).
	SipProfiles *VoiceSipProfilesService

	// ProgrammableVoiceProfiles — route a set of the account's numbers through Signal House's own voice servers.
	ProgrammableVoiceProfiles *VoiceProgrammableVoiceProfilesService

	// NumberRoutes — per-number inbound route (overrides the profile and subgroup settings).
	NumberRoutes *VoiceNumberRoutesService

	// Calls — outbound call origination and call log queries.
	Calls *VoiceCallsService

	// CallLogs — account-scoped call history: list/get + presigned recording.
	CallLogs *VoiceCallLogsService

	// Analytics — aggregated call metrics (summary tiles + byDate/byNumber/byCarrier breakdowns).
	Analytics *VoiceAnalyticsService

	// GlobalVoiceSettings — account-wide voice defaults (accepted regions, max spend per minute, E911).
	GlobalVoiceSettings *VoiceGlobalVoiceSettingsService

	// Tokens — mint ephemeral SIP credentials for the browser voice SDK.
	Tokens *VoiceTokensService
}
