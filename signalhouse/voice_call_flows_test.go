package signalhouse

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestCallFlowRequests(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	flows := client.Voice.CallFlows
	const id = "0088947b-7821-4853-bfe6-744fce926de2"
	const base = "/voice/api/v1/call-flows"
	// body is the JSON the server decoded: "null" when the request had none. The input-less
	// POSTs must send {}, because the Voice service rejects a JSON request with no body.
	cases := []struct {
		name   string
		method string
		uri    string
		body   string
		call   func() (*Response, error)
	}{
		{"ListCallFlows", "GET", base, "null", func() (*Response, error) {
			return flows.ListCallFlows(ctx, ListCallFlowsParams{}, nil)
		}},
		{"ListCallFlows filtered", "GET", base + "?includeDeleted=true&page=2&subgroupId=S1", "null", func() (*Response, error) {
			return flows.ListCallFlows(ctx, ListCallFlowsParams{Page: "2", SubgroupID: "S1", IncludeDeleted: "true"}, nil)
		}},
		{"GetCallFlowCatalog", "GET", base + "/catalog", "null", func() (*Response, error) {
			return flows.GetCallFlowCatalog(ctx, nil)
		}},
		{"GetCallFlow", "GET", base + "/" + id, "null", func() (*Response, error) {
			return flows.GetCallFlow(ctx, id, nil)
		}},
		{"GetCallFlow encodes the id", "GET", base + "/a%2Fb", "null", func() (*Response, error) {
			return flows.GetCallFlow(ctx, "a/b", nil)
		}},
		{"CreateCallFlow", "POST", base, `{"name":"Main line","subgroupId":"S1"}`, func() (*Response, error) {
			return flows.CreateCallFlow(ctx, CreateCallFlowParams{Name: "Main line", SubgroupID: "S1"}, nil)
		}},
		{"UpdateCallFlow", "PATCH", base + "/" + id, `{"status":"DISABLED"}`, func() (*Response, error) {
			return flows.UpdateCallFlow(ctx, id, UpdateCallFlowParams{Status: "DISABLED"}, nil)
		}},
		{"SaveCallFlowDraft sends arrays, not null", "PUT", base + "/" + id + "/draft", `{"edges":[],"nodes":[]}`, func() (*Response, error) {
			return flows.SaveCallFlowDraft(ctx, id, CallFlowDraft{}, nil)
		}},
		{"DeleteCallFlow", "DELETE", base + "/" + id, "null", func() (*Response, error) {
			return flows.DeleteCallFlow(ctx, id, nil)
		}},
		{"PublishCallFlow", "POST", base + "/" + id + "/publish", "{}", func() (*Response, error) {
			return flows.PublishCallFlow(ctx, id, nil)
		}},
		{"RestoreCallFlow", "POST", base + "/" + id + "/restore", "{}", func() (*Response, error) {
			return flows.RestoreCallFlow(ctx, id, nil)
		}},
		{"DuplicateCallFlow", "POST", base + "/" + id + "/duplicate", "{}", func() (*Response, error) {
			return flows.DuplicateCallFlow(ctx, id, "", nil)
		}},
		{"DuplicateCallFlow named", "POST", base + "/" + id + "/duplicate", `{"name":"Weekend"}`, func() (*Response, error) {
			return flows.DuplicateCallFlow(ctx, id, "Weekend", nil)
		}},
	}
	for _, c := range cases {
		if _, err := c.call(); err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		body, _ := json.Marshal(last.body)
		if last.method != c.method || last.uri != c.uri || string(body) != c.body {
			t.Errorf("%s: got %s %s %s, want %s %s %s", c.name, last.method, last.uri, body, c.method, c.uri, c.body)
		}
	}

	if _, err := flows.PublishCallFlow(ctx, id, &RequestOptions{Token: "tok-1", IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if got := last.header.Get("Authorization"); got != "Bearer tok-1" {
		t.Errorf("Authorization = %q, want the per-call token", got)
	}
	if got := last.header.Get("Idempotency-Key"); got != "k1" {
		t.Errorf("Idempotency-Key = %q, want k1", got)
	}
}

func TestCallFlowRequiredInputs(t *testing.T) {
	client, err := NewClient(Config{APIKey: "test", BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	flows := client.Voice.CallFlows
	var validation *ValidationError
	if _, err := flows.GetCallFlow(ctx, "", nil); !errors.As(err, &validation) {
		t.Errorf("GetCallFlow: got %v, want a ValidationError before sending", err)
	}
	if _, err := flows.PublishCallFlow(ctx, "", nil); !errors.As(err, &validation) {
		t.Errorf("PublishCallFlow: got %v, want a ValidationError before sending", err)
	}
	if _, err := flows.SaveCallFlowDraft(ctx, "", CallFlowDraft{}, nil); !errors.As(err, &validation) {
		t.Errorf("SaveCallFlowDraft: got %v, want a ValidationError before sending", err)
	}
	if _, err := flows.CreateCallFlow(ctx, CreateCallFlowParams{Name: "Main line"}, nil); !errors.As(err, &validation) {
		t.Errorf("CreateCallFlow without a subgroup: got %v, want a ValidationError before sending", err)
	}
}
