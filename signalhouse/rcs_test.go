package signalhouse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const rcsAgentID = "RCSA7K2M1QZ8RP"

func TestRcsAgentTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	agents := client.Rcs.Agents

	if _, err := agents.Create(ctx, CreateRcsAgentParams{
		BrandID: "BXYZ123", CampaignID: "CABC123", Agent: RcsAgentProfile{Name: "Acme"}, UseCase: "TRANSACTIONAL",
	}, &RequestOptions{IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/rcs/agent" || last.header.Get("Idempotency-Key") != "k1" {
		t.Fatalf("create contract: %s %s", last.method, last.uri)
	}
	if string(mustJSON(t, last.body)) != `{"agent":{"name":"Acme"},"brandId":"BXYZ123","campaignId":"CABC123","useCase":"TRANSACTIONAL"}` {
		t.Fatalf("create body: %s", mustJSON(t, last.body))
	}
	if _, err := agents.Create(ctx, CreateRcsAgentParams{Agent: RcsAgentProfile{Name: "Acme"}}, nil); err == nil || !strings.Contains(err.Error(), "brandId") {
		t.Fatalf("missing brandId must fail before transport: %v", err)
	}

	if _, err := agents.List(ctx, ListRcsAgentsParams{BrandID: "BXYZ123", CampaignID: "CABC123", Status: "DRAFT", Page: "2", Limit: "50"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/rcs/agent?brandId=BXYZ123&campaignId=CABC123&limit=50&page=2&status=DRAFT" {
		t.Fatalf("list contract: %s", last.uri)
	}

	calls := []struct {
		method, uri string
		call        func() (*Response, error)
	}{
		{"GET", "/rcs/agent/" + rcsAgentID, func() (*Response, error) {
			return agents.Get(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"PUT", "/rcs/agent/" + rcsAgentID, func() (*Response, error) {
			return agents.Update(ctx, UpdateRcsAgentParams{AgentID: rcsAgentID, ClearCampaign: true, Agent: map[string]any{"description": "Alerts"}}, nil)
		}},
		{"POST", "/rcs/agent/" + rcsAgentID + "/validate", func() (*Response, error) {
			return agents.Validate(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"POST", "/rcs/agent/" + rcsAgentID + "/submit", func() (*Response, error) {
			return agents.Submit(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"POST", "/registration/" + rcsAgentID + "/launch", func() (*Response, error) {
			return agents.Launch(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"POST", "/registration/" + rcsAgentID + "/carrier-update", func() (*Response, error) {
			return agents.RequestCarrierUpdate(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"POST", "/rcs/agent/" + rcsAgentID + "/archive", func() (*Response, error) {
			return agents.Archive(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
		}},
		{"DELETE", "/rcs/agent/" + rcsAgentID + "/testers/15125550199", func() (*Response, error) {
			return agents.RemoveTester(ctx, RemoveRcsTesterParams{AgentID: rcsAgentID, PhoneNumber: "15125550199"}, nil)
		}},
		{"POST", "/rcs/agent/" + rcsAgentID + "/testers/15125550199/resend", func() (*Response, error) {
			return agents.ResendTesterInvite(ctx, ResendRcsTesterInviteParams{AgentID: rcsAgentID, PhoneNumber: "15125550199"}, nil)
		}},
	}
	for _, c := range calls {
		if _, err := c.call(); err != nil {
			t.Fatal(err)
		}
		if last.method != c.method || last.uri != c.uri {
			t.Fatalf("got %s %s, want %s %s", last.method, last.uri, c.method, c.uri)
		}
		if c.method == "PUT" && string(mustJSON(t, last.body)) != `{"agent":{"description":"Alerts"},"campaignId":null}` {
			t.Fatalf("update body: %s", mustJSON(t, last.body))
		}
	}
	if _, err := agents.Update(ctx, UpdateRcsAgentParams{AgentID: "RCSA/1", ExcludedCarriers: []string{}}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/rcs/agent/RCSA%2F1" || string(mustJSON(t, last.body)) != `{"excludedCarriers":[]}` {
		t.Fatalf("update escapes id and sends an empty list: %s %s", last.uri, mustJSON(t, last.body))
	}
	var noProfile *RcsAgentProfile
	if _, err := agents.Update(ctx, UpdateRcsAgentParams{AgentID: rcsAgentID, Agent: noProfile, Contact: map[string]any(nil), ExcludedCarriers: []string{}}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"excludedCarriers":[]}` {
		t.Fatalf("a typed nil profile or contact is omitted, not sent as null: %s", mustJSON(t, last.body))
	}
	if _, err := agents.Submit(ctx, RcsAgentParams{}, nil); err == nil || !strings.Contains(err.Error(), "agentId") {
		t.Fatalf("missing agentId must fail before transport: %v", err)
	}
	if _, err := agents.CreateEntryPoints(ctx, CreateRcsEntryPointsParams{AgentID: "RCSA/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/rcs/agent/RCSA%2F1/entry-points" || string(mustJSON(t, last.body)) != `{}` {
		t.Fatalf("entry points escape the id and omit an empty body: %s %s %s", last.method, last.uri, mustJSON(t, last.body))
	}
	if _, err := agents.CreateEntryPoints(ctx, CreateRcsEntryPointsParams{Body: "Hi there"}, nil); err == nil || !strings.Contains(err.Error(), "agentId") {
		t.Fatalf("missing agentId must fail before transport: %v", err)
	}
}

func TestRcsMessageTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	messages := client.Rcs.Messages
	recipients := []map[string]any{{"phoneNumber": "+15125550199", "variables": map[string]any{"name": "Ada"}}}
	yes, no := true, false

	if _, err := messages.Send(ctx, SendRcsMessageParams{
		AgentID: rcsAgentID, Recipients: recipients, Content: map[string]any{"blocks": []any{}}, ContentFormat: "content.v1",
		TrafficType: "TRANSACTION", TTLSeconds: 900, Fallback: map[string]any{"mode": "auto", "content": map[string]any{"text": "Hi"}},
		StatusCallbackURL: "https://example.com/cb", EnableShortlink: &no, IsTest: &yes, Simulate: "delivered",
	}, &RequestOptions{IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/message/rcs" || last.header.Get("Idempotency-Key") != "k1" {
		t.Fatalf("send contract: %s %s", last.method, last.uri)
	}
	want := `{"agentId":"RCSA7K2M1QZ8RP","content":{"blocks":[]},"contentFormat":"content.v1","enableShortlink":false,` +
		`"fallback":{"content":{"text":"Hi"},"mode":"auto"},"isTest":true,` +
		`"recipients":[{"phoneNumber":"+15125550199","variables":{"name":"Ada"}}],"simulate":"delivered",` +
		`"statusCallbackUrl":"https://example.com/cb","trafficType":"TRANSACTION","ttlSeconds":900}`
	if string(mustJSON(t, last.body)) != want {
		t.Fatalf("send body: %s", mustJSON(t, last.body))
	}
	if _, err := messages.Send(ctx, SendRcsMessageParams{Recipients: recipients}, nil); err == nil || !strings.Contains(err.Error(), "agentId") {
		t.Fatalf("missing agentId must fail before transport: %v", err)
	}
	if _, err := messages.Send(ctx, SendRcsMessageParams{AgentID: rcsAgentID}, nil); err == nil || !strings.Contains(err.Error(), "recipients") {
		t.Fatalf("missing recipients must fail before transport: %v", err)
	}

	if _, err := messages.SendBatch(ctx, SendRcsMessageParams{AgentID: rcsAgentID, Recipients: recipients, TemplateID: "T1"}, &RequestOptions{IdempotencyKey: "k2"}); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/message/rcs/batch" || last.header.Get("Idempotency-Key") != "k2" {
		t.Fatalf("send batch contract: %s %s", last.method, last.uri)
	}
	if _, err := messages.SendBatch(ctx, SendRcsMessageParams{AgentID: rcsAgentID}, nil); err == nil || !strings.Contains(err.Error(), "recipients") {
		t.Fatalf("missing recipients must fail before transport: %v", err)
	}

	if _, err := messages.Validate(ctx, SendRcsMessageParams{AgentID: rcsAgentID, Recipients: recipients, TemplateID: "T1", TemplateVersion: 2, ExpireTime: "2026-11-05T00:00:00Z"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/message/rcs/validate" {
		t.Fatalf("validate contract: %s %s", last.method, last.uri)
	}
	want = `{"agentId":"RCSA7K2M1QZ8RP","expireTime":"2026-11-05T00:00:00Z",` +
		`"recipients":[{"phoneNumber":"+15125550199","variables":{"name":"Ada"}}],"templateId":"T1","templateVersion":2}`
	if string(mustJSON(t, last.body)) != want {
		t.Fatalf("validate body: %s", mustJSON(t, last.body))
	}

	if _, err := messages.Estimate(ctx, EstimateRcsMessageParams{AgentID: rcsAgentID, RecipientCount: 500, TemplateID: "T1", TemplateVersion: 2, TrafficType: "PROMOTION", Fallback: map[string]any{"mode": "rcs_only"}}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/message/estimate" {
		t.Fatalf("estimate contract: %s %s", last.method, last.uri)
	}
	if string(mustJSON(t, last.body)) != `{"agentId":"RCSA7K2M1QZ8RP","channel":"rcs","fallback":{"mode":"rcs_only"},"recipientCount":500,"templateId":"T1","templateVersion":2,"trafficType":"PROMOTION"}` {
		t.Fatalf("estimate body: %s", mustJSON(t, last.body))
	}
	if _, err := messages.Estimate(ctx, EstimateRcsMessageParams{RecipientCount: 500}, nil); err == nil || !strings.Contains(err.Error(), "agentId") {
		t.Fatalf("missing agentId must fail before transport: %v", err)
	}

	if _, err := messages.Capabilities(ctx, RcsCapabilitiesParams{AgentID: rcsAgentID, PhoneNumbers: []string{"+15125550199"}}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/rcs/capabilities" || string(mustJSON(t, last.body)) != `{"agentId":"RCSA7K2M1QZ8RP","phoneNumbers":["+15125550199"]}` {
		t.Fatalf("capabilities contract: %s %s %s", last.method, last.uri, mustJSON(t, last.body))
	}
	latitude, longitude := 30.26, -97.74
	if _, err := messages.SimulateInbound(ctx, SimulateRcsInboundParams{AgentID: rcsAgentID, From: "+15125550199", Type: "LOCATION", Latitude: &latitude, Longitude: &longitude}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/message/rcs/simulate-inbound" || string(mustJSON(t, last.body)) != `{"agentId":"RCSA7K2M1QZ8RP","from":"+15125550199","latitude":30.26,"longitude":-97.74,"type":"LOCATION"}` {
		t.Fatalf("simulate contract: %s %s %s", last.method, last.uri, mustJSON(t, last.body))
	}
	if _, err := messages.SimulateInbound(ctx, SimulateRcsInboundParams{AgentID: rcsAgentID, Type: "TEXT"}, nil); err == nil {
		t.Fatal("missing from must fail before transport")
	}
	if _, err := messages.Capabilities(ctx, RcsCapabilitiesParams{AgentID: rcsAgentID}, nil); err == nil || !strings.Contains(err.Error(), "phoneNumbers") {
		t.Fatalf("missing phoneNumbers must fail before transport: %v", err)
	}

	if _, err := messages.Get(ctx, GetRcsMessageParams{MessageID: "a/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/message/a%2F1" {
		t.Fatalf("get contract: %s %s", last.method, last.uri)
	}
	if _, err := messages.Get(ctx, GetRcsMessageParams{}, nil); err == nil || !strings.Contains(err.Error(), "messageId") {
		t.Fatalf("missing messageId must fail before transport: %v", err)
	}

	if _, err := client.Messages.GetMessage(ctx, GetMessageParams{ID: "a/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/message/a%2F1" {
		t.Fatalf("getMessage contract: %s %s", last.method, last.uri)
	}
	if _, err := client.Messages.GetMessage(ctx, GetMessageParams{}, nil); err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("missing id must fail before transport: %v", err)
	}

	if _, err := messages.List(ctx, ListRcsMessagesParams{
		AgentID: rcsAgentID, From: "2026-11-01", To: "2026-11-02", GroupID: "G1", Status: "READ", Page: "2", Limit: "50",
		TemplateID: "T1", TemplateVersion: "3", BulkID: "B1", FallbackReason: "NOT_CAPABLE", BillingClass: "RICH_MEDIA",
		DeliveredChannel: "sms", IsTest: &no,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/message?billingClass=RICH_MEDIA&bulkId=B1&dateBounds=exact&deliveredChannel=sms&endDate=2026-11-02&fallbackReason=NOT_CAPABLE&groupId=G1&isTest=false&limit=50&messageType=RCS&page=2&registrationId="+rcsAgentID+"&startDate=2026-11-01&status=READ&templateId=T1&templateVersion=3" {
		t.Fatalf("list contract: %s", last.uri)
	}
	last.uri = ""
	for field, params := range map[string]ListRcsMessagesParams{
		"groupId": {AgentID: rcsAgentID, From: "2026-11-01", To: "2026-11-02"},
		"from":    {GroupID: "G1", To: "2026-11-02"},
		"to":      {GroupID: "G1", From: "2026-11-01"},
	} {
		if _, err := messages.List(ctx, params, nil); err == nil || err.Error() != "missing required parameter: "+field {
			t.Fatalf("missing %s must fail before transport: %v", field, err)
		}
	}
	if last.uri != "" {
		t.Fatalf("a refused list must send no request: %s", last.uri)
	}
}

func TestGetMessagesForwardsRcsFilters(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	no := false

	if _, err := client.Messages.GetMessages(context.Background(), GetMessagesParams{
		GroupID: "G1", StartDate: "2026-11-01", EndDate: "2026-11-02", TemplateID: "T1", TemplateVersion: "3", BulkID: "B1",
		FallbackReason: "TTL_EXPIRED", BillingClass: "RICH", DeliveredChannel: "sms", IsTest: &no,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/message?billingClass=RICH&bulkId=B1&deliveredChannel=sms&endDate=2026-11-02&fallbackReason=TTL_EXPIRED&groupId=G1&isTest=false&startDate=2026-11-01&templateId=T1&templateVersion=3" {
		t.Fatalf("getMessages contract: %s", last.uri)
	}
}

func TestRcsAgentAssetUpload(t *testing.T) {
	var method, path, kind, filename, content, contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, contentType = r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			kind = r.FormValue("kind")
			if f, header, err := r.FormFile("file"); err == nil {
				filename = header.Filename
				data, _ := io.ReadAll(f)
				content = string(data)
			}
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Rcs.Agents.UploadAsset(ctx, UploadRcsAgentAssetParams{
		AgentID: rcsAgentID, Kind: "logo", File: &FileUpload{Filename: "logo.png", Reader: strings.NewReader("PNG")},
	}, nil); err != nil {
		t.Fatal(err)
	}
	if method != "POST" || path != "/rcs/agent/"+rcsAgentID+"/asset" || !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Fatalf("upload contract: %s %s %s", method, path, contentType)
	}
	if kind != "logo" || filename != "logo.png" || content != "PNG" {
		t.Fatalf("upload form: %q %q %q", kind, filename, content)
	}
	for _, params := range []UploadRcsAgentAssetParams{
		{Kind: "logo", File: &FileUpload{Reader: strings.NewReader("PNG")}},
		{AgentID: rcsAgentID, File: &FileUpload{Reader: strings.NewReader("PNG")}},
		{AgentID: rcsAgentID, Kind: "hero"},
	} {
		if _, err := client.Rcs.Agents.UploadAsset(ctx, params, nil); err == nil {
			t.Fatalf("missing parameter must fail before transport: %+v", params)
		}
	}
}

func TestRegistrationDrafts(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reg := client.Registrations

	if _, err := reg.UpdateRegistration(ctx, UpdateRegistrationParams{RegistrationID: rcsAgentID, Data: map[string]any{"useCase": "OTP"}}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "PUT" || last.uri != "/registration/"+rcsAgentID || string(mustJSON(t, last.body)) != `{"useCase":"OTP"}` {
		t.Fatalf("update contract: %s %s %s", last.method, last.uri, mustJSON(t, last.body))
	}
	if _, err := reg.ValidateRegistration(ctx, ValidateRegistrationParams{RegistrationID: rcsAgentID}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/registration/"+rcsAgentID+"/validate" {
		t.Fatalf("validate contract: %s %s", last.method, last.uri)
	}
	if _, err := reg.SubmitRegistration(ctx, SubmitRegistrationParams{RegistrationID: rcsAgentID}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/registration/"+rcsAgentID+"/submit" {
		t.Fatalf("submit contract: %s %s", last.method, last.uri)
	}
	if _, err := reg.LaunchRegistration(ctx, LaunchRegistrationParams{RegistrationID: rcsAgentID}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/registration/"+rcsAgentID+"/launch" {
		t.Fatalf("launch contract: %s %s", last.method, last.uri)
	}
	if _, err := reg.LaunchRegistration(ctx, LaunchRegistrationParams{RegistrationID: rcsAgentID, AcknowledgeFees: true}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"acknowledgeFees":true}` {
		t.Fatalf("launch body: %s", mustJSON(t, last.body))
	}
	if _, err := reg.LaunchRegistration(ctx, LaunchRegistrationParams{RegistrationID: rcsAgentID, AcknowledgeFees: true, AcknowledgePhi: true}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"acknowledgeFees":true,"acknowledgePhi":true}` {
		t.Fatalf("launch body with the PHI acknowledgement: %s", mustJSON(t, last.body))
	}
	if _, err := client.Rcs.Agents.Launch(ctx, RcsAgentParams{AgentID: rcsAgentID, AcknowledgeFees: true}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"acknowledgeFees":true}` {
		t.Fatalf("agent launch body: %s", mustJSON(t, last.body))
	}
	if _, err := client.Billing.GetTransactionHistory(ctx, GetTransactionHistoryParams{GroupID: "G1", Category: "RCS_INBOUND,RCS_CARRIER_PASSTHROUGH", Limit: "1000"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/billing/wallet/transactionHistory?category=RCS_INBOUND%2CRCS_CARRIER_PASSTHROUGH&groupId=G1&limit=1000" {
		t.Fatalf("history contract: %s", last.uri)
	}
	if _, err := reg.LaunchRegistration(ctx, LaunchRegistrationParams{}, nil); err == nil {
		t.Fatal("missing registrationId must fail before transport")
	}
	if _, err := reg.GetRegistrations(ctx, GetRegistrationsParams{Type: "RCS_AGENT", BrandID: "BXYZ123", CampaignID: "CABC123"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration?brandId=BXYZ123&campaignId=CABC123&type=RCS_AGENT" {
		t.Fatalf("link filters: %s", last.uri)
	}
	if _, err := reg.UpdateRegistration(ctx, UpdateRegistrationParams{RegistrationID: rcsAgentID}, nil); err == nil || !strings.Contains(err.Error(), "data") {
		t.Fatalf("missing data must fail before transport: %v", err)
	}
	if _, err := reg.SubmitRegistration(ctx, SubmitRegistrationParams{}, nil); err == nil {
		t.Fatal("missing registrationId must fail before transport")
	}
}

func TestAgentWebhookTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Webhooks.GetWebhooks(ctx, GetWebhooksParams{GroupID: "G1", EndpointType: "Agent", AgentID: rcsAgentID}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/webhook?agentId="+rcsAgentID+"&endpointType=Agent&groupId=G1" {
		t.Fatalf("agent filter: %s", last.uri)
	}
	webhookData := map[string]interface{}{"groupId": "G1", "endpointType": "Agent", "agentId": rcsAgentID, "url": "https://a.example/rcs", "secondaryUrl": "https://b.example/rcs"}
	if _, err := client.Webhooks.CreateWebhook(ctx, CreateWebhookParams{WebhookData: webhookData}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"agentId":"`+rcsAgentID+`","endpointType":"Agent","groupId":"G1","secondaryUrl":"https://b.example/rcs","url":"https://a.example/rcs"}` {
		t.Fatalf("create body: %s", mustJSON(t, last.body))
	}
	if _, err := client.Webhooks.UpdateWebhook(ctx, UpdateWebhookParams{ID: "e1", UpdateData: map[string]interface{}{"secondaryUrl": nil}}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"secondaryUrl":null}` {
		t.Fatalf("update body: %s", mustJSON(t, last.body))
	}
}

func TestMessageInteractionsTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Messages.GetMessageInteractions(ctx, GetMessageInteractionsParams{ID: "m1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/message/m1/interactions" {
		t.Fatalf("interactions contract: %s %s", last.method, last.uri)
	}
	if _, err := client.Messages.GetMessageInteractions(ctx, GetMessageInteractionsParams{}, nil); err == nil {
		t.Fatal("missing id must fail before transport")
	}
}

// fixtureInput reads one corpus case's input as Go parameter values.
type fixtureInput map[string]any

func (in fixtureInput) str(key string) string {
	value, _ := in[key].(string)
	return value
}

func (in fixtureInput) number(key string) int {
	value, _ := in[key].(float64)
	return int(value)
}

func (in fixtureInput) text(key string) string {
	if value := in.number(key); value != 0 {
		return strconv.Itoa(value)
	}
	return ""
}

func (in fixtureInput) flag(key string) *bool {
	if value, ok := in[key].(bool); ok {
		return &value
	}
	return nil
}

func (in fixtureInput) strs(key string) []string {
	values, ok := in[key].([]any)
	if !ok {
		return nil
	}
	list := make([]string, len(values))
	for index, value := range values {
		list[index] = value.(string)
	}
	return list
}

func (in fixtureInput) autofill() map[string]RcsAgentAutofillMarker {
	values, ok := in["autofill"].(map[string]any)
	if !ok {
		return nil
	}
	markers := map[string]RcsAgentAutofillMarker{}
	for path, value := range values {
		marker := fixtureInput(value.(map[string]any))
		edited, _ := marker["edited"].(bool)
		markers[path] = RcsAgentAutofillMarker{Source: marker.str("source"), SourceID: marker.str("sourceId"), At: marker.str("at"), Edited: edited}
	}
	return markers
}

func (in fixtureInput) recipients() []map[string]any {
	values, _ := in["recipients"].([]any)
	list := make([]map[string]any, len(values))
	for index, value := range values {
		list[index] = value.(map[string]any)
	}
	return list
}

func (in fixtureInput) send() SendRcsMessageParams {
	return SendRcsMessageParams{
		AgentID: in.str("agentId"), Recipients: in.recipients(), TemplateID: in.str("templateId"), TemplateVersion: in.number("templateVersion"), Content: in["content"],
		ContentFormat: in.str("contentFormat"), TrafficType: in.str("trafficType"), TTLSeconds: in.number("ttlSeconds"), ExpireTime: in.str("expireTime"), Fallback: in["fallback"],
		StatusCallbackURL: in.str("statusCallbackUrl"), EnableShortlink: in.flag("enableShortlink"), IsTest: in.flag("isTest"), Simulate: in.str("simulate"),
	}
}

// runFixture calls the operation a corpus case names.
func runFixture(ctx context.Context, client *Client, operation string, in fixtureInput) error {
	agents, messages := client.Rcs.Agents, client.Rcs.Messages
	agent := RcsAgentParams{AgentID: in.str("agentId")}
	if acknowledgeFees := in.flag("acknowledgeFees"); acknowledgeFees != nil {
		agent.AcknowledgeFees = *acknowledgeFees
	}
	if acknowledgePhi := in.flag("acknowledgePhi"); acknowledgePhi != nil {
		agent.AcknowledgePhi = *acknowledgePhi
	}
	var err error
	switch operation {
	case "messages.send":
		_, err = messages.Send(ctx, in.send(), nil)
	case "messages.sendBatch":
		_, err = messages.SendBatch(ctx, in.send(), nil)
	case "messages.validate":
		_, err = messages.Validate(ctx, in.send(), nil)
	case "messages.estimate":
		_, err = messages.Estimate(ctx, EstimateRcsMessageParams{AgentID: in.str("agentId"), RecipientCount: in.number("recipientCount"), Content: in["content"]}, nil)
	case "messages.capabilities":
		_, err = messages.Capabilities(ctx, RcsCapabilitiesParams{AgentID: in.str("agentId"), PhoneNumbers: in.strs("phoneNumbers")}, nil)
	case "messages.get":
		_, err = messages.Get(ctx, GetRcsMessageParams{MessageID: in.str("messageId")}, nil)
	case "messages.list":
		_, err = messages.List(ctx, ListRcsMessagesParams{GroupID: in.str("groupId"), From: in.str("from"), To: in.str("to"), AgentID: in.str("agentId"), Status: in.str("status"), Page: in.text("page"), Limit: in.text("limit")}, nil)
	case "agents.create":
		_, err = agents.Create(ctx, CreateRcsAgentParams{BrandID: in.str("brandId"), CampaignID: in.str("campaignId"), Agent: in["agent"], UseCase: in.str("useCase"), DefaultFallback: in["defaultFallback"], ExcludedCarriers: in.strs("excludedCarriers"), Testers: in.strs("testers")}, nil)
	case "agents.list":
		_, err = agents.List(ctx, ListRcsAgentsParams{Status: in.str("status"), BrandID: in.str("brandId"), Page: in.text("page"), Limit: in.text("limit")}, nil)
	case "agents.get":
		_, err = agents.Get(ctx, agent, nil)
	case "agents.update":
		_, err = agents.Update(ctx, UpdateRcsAgentParams{AgentID: in.str("agentId"), Agent: in["agent"], Consent: in["consent"], Testers: in.strs("testers"), Autofill: in.autofill()}, nil)
	case "agents.validate":
		_, err = agents.Validate(ctx, agent, nil)
	case "agents.submit":
		_, err = agents.Submit(ctx, agent, nil)
	case "agents.launch":
		_, err = agents.Launch(ctx, agent, nil)
	case "agents.requestCarrierUpdate":
		_, err = agents.RequestCarrierUpdate(ctx, agent, nil)
	case "agents.archive":
		_, err = agents.Archive(ctx, agent, nil)
	case "agents.createEntryPoints":
		_, err = agents.CreateEntryPoints(ctx, CreateRcsEntryPointsParams{AgentID: in.str("agentId"), Body: in.str("body")}, nil)
	case "agents.resendTesterInvite":
		_, err = agents.ResendTesterInvite(ctx, ResendRcsTesterInviteParams{AgentID: in.str("agentId"), PhoneNumber: in.str("phoneNumber")}, nil)
	case "agents.removeTester":
		_, err = agents.RemoveTester(ctx, RemoveRcsTesterParams{AgentID: in.str("agentId"), PhoneNumber: in.str("phoneNumber")}, nil)
	default:
		panic("no adapter for " + operation)
	}
	return err
}

func TestSharedFixtureCorpus(t *testing.T) {
	raw, err := os.ReadFile("../../fixtures/rcs/requests.json")
	if os.IsNotExist(err) {
		t.Skip("the corpus lives in the monorepo")
	}
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Cases []struct {
			Name      string       `json:"name"`
			Operation string       `json:"operation"`
			Input     fixtureInput `json:"input"`
			Request   struct {
				Method string            `json:"method"`
				Path   string            `json:"path"`
				Query  map[string]string `json:"query"`
				Body   map[string]any    `json:"body"`
			} `json:"request"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range corpus.Cases {
		t.Run(test.Name, func(t *testing.T) {
			if err := runFixture(context.Background(), client, test.Operation, test.Input); err != nil {
				t.Fatal(err)
			}
			parsed, err := url.ParseRequestURI(last.uri)
			if err != nil {
				t.Fatal(err)
			}
			query := map[string]string{}
			for key, values := range parsed.Query() {
				query[key] = values[0]
			}
			if last.method != test.Request.Method || parsed.Path != test.Request.Path {
				t.Fatalf("request: %s %s", last.method, last.uri)
			}
			if len(query)+len(test.Request.Query) > 0 && !reflect.DeepEqual(query, test.Request.Query) {
				t.Fatalf("query: %v", query)
			}
			if len(last.body)+len(test.Request.Body) > 0 && !reflect.DeepEqual(last.body, test.Request.Body) {
				t.Fatalf("body: %v", last.body)
			}
		})
	}
}
