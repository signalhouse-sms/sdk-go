package signalhouse

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestPushCredentialRequests(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	creds := client.Voice.PushCredentials
	const id = "5a3c2f8e-1b7d-4c9a-9e2f-0d4b6a8c1e33"
	const base = "/voice/api/v1/push-credentials"
	apns := PushCredentialParams{
		Provider:   "apns",
		KeyID:      "ABC123DEFG",
		TeamID:     "TEAM123456",
		BundleID:   "com.example.app",
		PrivateKey: "p8-key-contents",
	}
	// body is the JSON the server decoded, re-marshaled with sorted keys: "null" when the
	// request had none.
	cases := []struct {
		name   string
		method string
		uri    string
		body   string
		call   func() (*Response, error)
	}{
		{"ListPushCredentials", "GET", base, "null", func() (*Response, error) {
			return creds.ListPushCredentials(ctx, nil)
		}},
		{"GetPushCredential", "GET", base + "/" + id, "null", func() (*Response, error) {
			return creds.GetPushCredential(ctx, id, nil)
		}},
		{"GetPushCredential encodes the id", "GET", base + "/a%2Fb", "null", func() (*Response, error) {
			return creds.GetPushCredential(ctx, "a/b", nil)
		}},
		{"CreatePushCredential apns", "POST", base, `{"bundleId":"com.example.app","keyId":"ABC123DEFG","privateKey":"p8-key-contents","provider":"apns","teamId":"TEAM123456"}`, func() (*Response, error) {
			return creds.CreatePushCredential(ctx, apns, nil)
		}},
		{"CreatePushCredential fcm with an object", "POST", base, `{"name":"Android","provider":"fcm","serviceAccount":{"project_id":"demo"}}`, func() (*Response, error) {
			return creds.CreatePushCredential(ctx, PushCredentialParams{Provider: "fcm", Name: "Android", ServiceAccount: map[string]interface{}{"project_id": "demo"}}, nil)
		}},
		{"UpdatePushCredential fcm with JSON text", "PUT", base + "/" + id, `{"provider":"fcm","serviceAccount":"{}"}`, func() (*Response, error) {
			return creds.UpdatePushCredential(ctx, id, PushCredentialParams{Provider: "fcm", ServiceAccount: "{}"}, nil)
		}},
		{"DeletePushCredential", "DELETE", base + "/" + id, "null", func() (*Response, error) {
			return creds.DeletePushCredential(ctx, id, nil)
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
}

func TestPushDeviceRequests(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	devices := client.Voice.PushDevices
	const base = "/voice/api/v1/push-devices"
	// The app calls these with its voice token, not the client's API key.
	voiceToken := &RequestOptions{Token: "voice-jwt"}
	cases := []struct {
		name   string
		method string
		uri    string
		body   string
		call   func() (*Response, error)
	}{
		{"RegisterPushDevice", "POST", base, `{"deviceToken":"fcm:tok/1","provider":"fcm"}`, func() (*Response, error) {
			return devices.RegisterPushDevice(ctx, RegisterPushDeviceParams{Provider: "fcm", DeviceToken: "fcm:tok/1"}, voiceToken)
		}},
		{"RegisterPushDevice apns sandbox", "POST", base, `{"appId":"com.example.app","deviceToken":"abc123","environment":"sandbox","provider":"apns"}`, func() (*Response, error) {
			return devices.RegisterPushDevice(ctx, RegisterPushDeviceParams{Provider: "apns", DeviceToken: "abc123", Environment: "sandbox", AppID: "com.example.app"}, voiceToken)
		}},
		// url.PathEscape keeps ":" (legal in a path segment) and escapes "/".
		{"UnregisterPushDevice escapes the device token", "DELETE", base + "/fcm:tok%2F1", "null", func() (*Response, error) {
			return devices.UnregisterPushDevice(ctx, "fcm:tok/1", voiceToken)
		}},
		{"ListPushDeliveries", "GET", base + "/deliveries", "null", func() (*Response, error) {
			return devices.ListPushDeliveries(ctx, ListPushDeliveriesParams{}, voiceToken)
		}},
		{"ListPushDeliveries limited", "GET", base + "/deliveries?limit=5", "null", func() (*Response, error) {
			return devices.ListPushDeliveries(ctx, ListPushDeliveriesParams{Limit: "5"}, voiceToken)
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
		if got := last.header.Get("Authorization"); got != "Bearer voice-jwt" {
			t.Errorf("%s: Authorization = %q, want the voice token", c.name, got)
		}
	}
}

func TestPushRequiredInputs(t *testing.T) {
	client, err := NewClient(Config{APIKey: "test", BaseURL: "http://127.0.0.1:1"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	creds := client.Voice.PushCredentials
	devices := client.Voice.PushDevices
	const id = "5a3c2f8e-1b7d-4c9a-9e2f-0d4b6a8c1e33"
	checks := []struct {
		name string
		call func() (*Response, error)
	}{
		{"GetPushCredential without an id", func() (*Response, error) {
			return creds.GetPushCredential(ctx, "", nil)
		}},
		{"CreatePushCredential without a provider", func() (*Response, error) {
			return creds.CreatePushCredential(ctx, PushCredentialParams{}, nil)
		}},
		{"UpdatePushCredential without an id", func() (*Response, error) {
			return creds.UpdatePushCredential(ctx, "", PushCredentialParams{Provider: "fcm"}, nil)
		}},
		{"UpdatePushCredential without a provider", func() (*Response, error) {
			return creds.UpdatePushCredential(ctx, id, PushCredentialParams{}, nil)
		}},
		{"DeletePushCredential without an id", func() (*Response, error) {
			return creds.DeletePushCredential(ctx, "", nil)
		}},
		{"RegisterPushDevice without a device token", func() (*Response, error) {
			return devices.RegisterPushDevice(ctx, RegisterPushDeviceParams{Provider: "fcm"}, nil)
		}},
		{"UnregisterPushDevice without a device token", func() (*Response, error) {
			return devices.UnregisterPushDevice(ctx, "", nil)
		}},
	}
	for _, c := range checks {
		var validation *ValidationError
		if _, err := c.call(); !errors.As(err, &validation) {
			t.Errorf("%s: got %v, want a ValidationError before sending", c.name, err)
		}
	}
}
