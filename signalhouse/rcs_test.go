package signalhouse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
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
		{"POST", "/rcs/agent/" + rcsAgentID + "/archive", func() (*Response, error) {
			return agents.Archive(ctx, RcsAgentParams{AgentID: rcsAgentID}, nil)
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
