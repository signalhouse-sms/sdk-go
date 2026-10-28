package signalhouse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// captured is the last request a capture server saw.
type captured struct {
	method string
	uri    string
	body   map[string]interface{}
	header http.Header
}

// newCaptureServer records each request and answers with an empty JSON object.
func newCaptureServer(t *testing.T) (*httptest.Server, *captured) {
	t.Helper()
	last := &captured{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last.method, last.uri, last.body, last.header = r.Method, r.URL.RequestURI(), nil, r.Header.Clone()
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&last.body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(server.Close)
	return server, last
}

func TestRegionalReportQueries(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	scopes := `[{"region":"GB","channels":["virtualLongCode"],"phoneNumber":["447400123456"]}]`
	reads := []struct {
		name string
		path string
		call func() (*Response, error)
	}{
		{"GetMessages", "/message", func() (*Response, error) {
			return client.Messages.GetMessages(ctx, GetMessagesParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetAnalytics", "/message/analytics", func() (*Response, error) {
			return client.Messages.GetAnalytics(ctx, GetAnalyticsParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetAnalyticsDetail", "/message/analytics/detail", func() (*Response, error) {
			return client.Messages.GetAnalyticsDetail(ctx, GetAnalyticsDetailParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetAnalyticsBySubgroup", "/message/analytics/by-subgroup", func() (*Response, error) {
			return client.Messages.GetAnalyticsBySubgroup(ctx, GetAnalyticsBySubgroupParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetAnalyticsByErrorCode", "/message/analytics/by-error-code", func() (*Response, error) {
			return client.Messages.GetAnalyticsByErrorCode(ctx, GetAnalyticsByErrorCodeParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetDncAnalytics", "/message/dnc/analytics", func() (*Response, error) {
			return client.Messages.GetDncAnalytics(ctx, GetDncAnalyticsParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
		{"GetDncRecords", "/message/dnc", func() (*Response, error) {
			return client.Messages.GetDncRecords(ctx, GetDncRecordsParams{GroupID: "G1", Region: "US,GB", RegionScopes: scopes}, nil)
		}},
	}
	for _, read := range reads {
		t.Run(read.name, func(t *testing.T) {
			if _, err := read.call(); err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest("GET", server.URL+last.uri, nil)
			if err != nil {
				t.Fatal(err)
			}
			if req.URL.Path != read.path {
				t.Fatalf("path %q, want %q", req.URL.Path, read.path)
			}
			q := req.URL.Query()
			if q.Get("region") != "US,GB" || q.Get("regionScopes") != scopes || q.Get("groupId") != "G1" {
				t.Fatalf("regional query lost: %s", last.uri)
			}
		})
	}

	if _, err := client.Messages.GetMessages(ctx, GetMessagesParams{Region: "GB", RegionScopes: "[]"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/message?region=GB&regionScopes=%5B%5D" {
		t.Fatalf("empty scopes lost: %s", last.uri)
	}
	if _, err := client.Messages.GetDncRecords(ctx, GetDncRecordsParams{GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/message/dnc?groupId=G1" {
		t.Fatalf("region sent by default: %s", last.uri)
	}
	if _, err := client.Messages.GetAnalyticsFilterOptions(ctx, GetAnalyticsFilterOptionsParams{GroupID: "G1", Region: "GB"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/message/analytics/filter-options?groupId=G1&region=GB" {
		t.Fatalf("filter options region lost: %s", last.uri)
	}
}

func TestRegistrationTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL, EnableAdmin: true})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Registrations.GetCatalog(ctx, GetCatalogParams{Region: "GB"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/registration/catalog?region=GB" {
		t.Fatalf("catalog contract: %s %s", last.method, last.uri)
	}
	if _, err := client.Registrations.GetCatalog(ctx, GetCatalogParams{}, nil); err == nil {
		t.Fatal("missing region must fail before transport")
	}

	if _, err := client.Registrations.GetQuotes(ctx, GetQuotesParams{Region: "GB", Type: "VIRTUAL_LONG_CODE", Quantity: 2}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/registration/quotes?quantity=2&region=GB&type=VIRTUAL_LONG_CODE" {
		t.Fatalf("quotes contract: %s %s", last.method, last.uri)
	}
	if _, err := client.Registrations.GetQuotes(ctx, GetQuotesParams{Region: "UK", Type: "VIRTUAL_LONG_CODE", GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration/quotes?groupId=G1&region=UK&type=VIRTUAL_LONG_CODE" {
		t.Fatalf("staff quotes contract: %s", last.uri)
	}
	if _, err := client.Registrations.GetQuotes(ctx, GetQuotesParams{Region: "GB"}, nil); err == nil {
		t.Fatal("missing type must fail before transport")
	}

	if _, err := client.Registrations.GetRegistrations(ctx, GetRegistrationsParams{
		GroupID: "G1", SubgroupID: "S1", Region: "GB,CA", Type: "VIRTUAL_LONG_CODE", Kind: "sender",
		Status: "PENDING_PROVIDER,APPROVED", ParentRegistrationID: "REG0", PhoneNumber: "15551234567",
		SortBy: "status", SortOrder: "asc", Page: "2", Limit: "10",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration?groupId=G1&kind=sender&limit=10&page=2&parentRegistrationId=REG0&phoneNumber=15551234567&region=GB%2CCA&sortBy=status&sortOrder=asc&status=PENDING_PROVIDER%2CAPPROVED&subgroupId=S1&type=VIRTUAL_LONG_CODE" {
		t.Fatalf("list contract: %s", last.uri)
	}
	if _, err := client.Registrations.GetRegistrations(ctx, GetRegistrationsParams{}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration" {
		t.Fatalf("unfiltered list contract: %s", last.uri)
	}
	if _, err := client.Registrations.GetRegistration(ctx, GetRegistrationParams{RegistrationID: "REG 1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration/REG%201" {
		t.Fatalf("read contract: %s", last.uri)
	}
	if _, err := client.Registrations.CancelRegistration(ctx, CancelRegistrationParams{RegistrationID: "GB8T9GTD5AZY"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "DELETE" || last.uri != "/registration/GB8T9GTD5AZY" {
		t.Fatalf("cancel contract: %s %s", last.method, last.uri)
	}

	_, err = client.Registrations.CreateRegistration(ctx, CreateRegistrationParams{
		Region: "GB", Type: "VIRTUAL_LONG_CODE", SubgroupID: "S1",
		Capabilities: []string{"SMS"}, Data: map[string]any{"quantity": 1, "useCases": []string{"Support"}},
	}, nil)
	if err != nil || last.method != "POST" || last.uri != "/registration" {
		t.Fatalf("create contract: %s %s %v", last.method, last.uri, err)
	}
	if last.body["region"] != "GB" || last.body["type"] != "VIRTUAL_LONG_CODE" {
		t.Fatalf("create body: %v", last.body)
	}
	data, ok := last.body["data"].(map[string]interface{})
	if !ok || data["quantity"] != float64(1) {
		t.Fatalf("create data: %v", last.body["data"])
	}
	for _, legacy := range []string{"quoteVersion", "version", "country", "quantity", "clientRequestId"} {
		if _, present := last.body[legacy]; present {
			t.Fatalf("%s must not be sent at the top level", legacy)
		}
	}
	if _, err := client.Registrations.CreateRegistration(ctx, CreateRegistrationParams{Region: "GB", Type: "VIRTUAL_LONG_CODE"}, nil); err == nil {
		t.Fatal("missing subgroupId must fail before transport")
	}

	admin := client.Registrations.Admin
	if _, err := admin.ReviewRegistration(ctx, ReviewRegistrationParams{RegistrationID: "REG1", Action: "APPROVE"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/registration/REG1/review" || last.body["action"] != "APPROVE" {
		t.Fatalf("review contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, present := last.body["reason"]; present {
		t.Fatal("empty reason must be omitted")
	}
	if _, present := last.body["registrationId"]; present {
		t.Fatal("registrationId belongs in the path, not the body")
	}
	if _, err := admin.ReviewRegistration(ctx, ReviewRegistrationParams{RegistrationID: "REG1", Action: "REJECT", Reason: "Not a business use case"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.body["reason"] != "Not a business use case" {
		t.Fatalf("reject reason lost: %v", last.body)
	}
	if _, err := admin.CorrectRegistration(ctx, CorrectRegistrationParams{RegistrationID: "REG1", Reason: "Provider asked", Data: map[string]any{"useCases": []string{"Support"}}}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration/REG1/corrections" || last.body["reason"] != "Provider asked" {
		t.Fatalf("corrections contract: %s %v", last.uri, last.body)
	}
	if corrections, ok := last.body["data"].(map[string]interface{}); !ok || corrections["useCases"] == nil {
		t.Fatalf("corrections data lost: %v", last.body)
	}
	if _, present := last.body["useCases"]; present {
		t.Fatal("correctable fields belong under data")
	}
	if _, err := admin.CorrectRegistration(ctx, CorrectRegistrationParams{RegistrationID: "REG1", Reason: "Provider asked"}, nil); err == nil {
		t.Fatal("missing data must fail before transport")
	}
	if _, err := admin.AddNumber(ctx, AddNumberParams{RegistrationID: "REG1", PhoneNumber: "447400123456"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration/REG1/numbers" || last.body["phoneNumber"] != "447400123456" {
		t.Fatalf("add number contract: %s %v", last.uri, last.body)
	}
	if _, err := admin.Reconcile(ctx, ReconcileParams{RegistrationID: "REG1", ExternalID: "ORDER-123"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/registration/REG1/reconcile" || last.body["externalId"] != "ORDER-123" {
		t.Fatalf("reconcile contract: %s %v", last.uri, last.body)
	}
	if _, present := last.body["providerReference"]; present {
		t.Fatal("providerReference must not be sent")
	}
	if _, err := admin.GetApplication(ctx, GetApplicationParams{RegistrationID: "REG1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/registration/REG1/application?format=json" {
		t.Fatalf("application contract: %s %s", last.method, last.uri)
	}

	if _, err := client.Registrations.GetRegistration(ctx, GetRegistrationParams{}, nil); err == nil {
		t.Fatal("missing registration ID must fail before transport")
	}
	if _, err := client.Registrations.CancelRegistration(ctx, CancelRegistrationParams{}, nil); err == nil {
		t.Fatal("missing registration ID must fail before cancel transport")
	}
	customer, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if customer.Registrations.Admin != nil {
		t.Fatal("staff namespace exposed by default")
	}
}

func TestRegistrationIDFilters(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	reads := []struct {
		name string
		path string
		call func() (*Response, error)
	}{
		{"GetAnalytics", "/message/analytics", func() (*Response, error) {
			return client.Messages.GetAnalytics(ctx, GetAnalyticsParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetAnalyticsDetail", "/message/analytics/detail", func() (*Response, error) {
			return client.Messages.GetAnalyticsDetail(ctx, GetAnalyticsDetailParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetAnalyticsThroughput", "/message/analytics/throughput", func() (*Response, error) {
			return client.Messages.GetAnalyticsThroughput(ctx, GetAnalyticsThroughputParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2", StartDate: "2026-09-01", EndDate: "2026-09-02"}, nil)
		}},
		{"GetDncAnalytics", "/message/dnc/analytics", func() (*Response, error) {
			return client.Messages.GetDncAnalytics(ctx, GetDncAnalyticsParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetAnalyticsBySubgroup", "/message/analytics/by-subgroup", func() (*Response, error) {
			return client.Messages.GetAnalyticsBySubgroup(ctx, GetAnalyticsBySubgroupParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetAnalyticsByErrorCode", "/message/analytics/by-error-code", func() (*Response, error) {
			return client.Messages.GetAnalyticsByErrorCode(ctx, GetAnalyticsByErrorCodeParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetDncRecords", "/message/dnc", func() (*Response, error) {
			return client.Messages.GetDncRecords(ctx, GetDncRecordsParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
		{"GetMessages", "/message", func() (*Response, error) {
			return client.Messages.GetMessages(ctx, GetMessagesParams{GroupID: "G1", CampaignID: "C1", RegistrationID: "REG1,REG2"}, nil)
		}},
	}
	for _, read := range reads {
		t.Run(read.name, func(t *testing.T) {
			if _, err := read.call(); err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest("GET", server.URL+last.uri, nil)
			if err != nil {
				t.Fatal(err)
			}
			q := req.URL.Query()
			if req.URL.Path != read.path || q.Get("registrationId") != "REG1,REG2" || q.Get("campaignId") != "C1" {
				t.Fatalf("registrationId not forwarded: %s", last.uri)
			}
		})
	}

	if _, err := client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{CampaignID: "C1", RegistrationID: "REG1"}, nil); err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", server.URL+last.uri, nil)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Path != "/number" || req.URL.Query().Get("registrationId") != "REG1" {
		t.Fatalf("GetPhoneNumbers lost registrationId: %s", last.uri)
	}
}

func TestAlphanumericSenderRegistration(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"requestedSenderId": "ACME", "trafficOrigin": "LOCAL", "trafficType": "TRANSACTIONAL",
		"companyName": "Acme Ltd", "companyCountry": "United Kingdom", "companyWebsite": "https://acme.example",
		"industry": "Retail", "senderRelationship": "Trading name of Acme Ltd", "messageExample": "Your Acme order has shipped.",
	}
	if _, err := client.Registrations.CreateRegistration(context.Background(), CreateRegistrationParams{
		Region: "GB", Type: "ALPHANUMERIC_SENDER_ID", SubgroupID: "S1",
		Capabilities: []string{"SMS"}, Data: data,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/registration" || last.body["type"] != "ALPHANUMERIC_SENDER_ID" || last.body["region"] != "GB" {
		t.Fatalf("alphanumeric submission contract: %s %s %v", last.method, last.uri, last.body)
	}
	sent, _ := json.Marshal(last.body["data"])
	want, _ := json.Marshal(data)
	if string(sent) != string(want) {
		t.Fatalf("alphanumeric data changed in transit: %s", sent)
	}
}

func TestShortlinkOptOut(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Shortlinks.GetOptOut(ctx, OptOutParams{Code: "ABC/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/shortlink/optout/ABC%2F1" || last.body != nil {
		t.Fatalf("get opt-out contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, err := client.Shortlinks.ConfirmOptOut(ctx, OptOutParams{Code: "ABC/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/shortlink/optout/ABC%2F1" || last.body != nil {
		t.Fatalf("confirm opt-out contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, err := client.Shortlinks.GetOptOut(ctx, OptOutParams{}, nil); err == nil {
		t.Fatal("missing code must fail before transport")
	}
	if _, err := client.Shortlinks.ConfirmOptOut(ctx, OptOutParams{}, nil); err == nil {
		t.Fatal("missing code must fail before transport")
	}
}

func TestIdempotencyKeyHeader(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	params := CreateRegistrationParams{Region: "GB", Type: "VIRTUAL_LONG_CODE", SubgroupID: "SG1", Capabilities: []string{"SMS"}}
	if _, err := client.Registrations.CreateRegistration(context.Background(), params, &RequestOptions{IdempotencyKey: "key-1"}); err != nil {
		t.Fatal(err)
	}
	if got := last.header.Get("Idempotency-Key"); got != "key-1" {
		t.Fatalf("Idempotency-Key = %q, want key-1", got)
	}
	if _, err := client.Registrations.CreateRegistration(context.Background(), params, nil); err != nil {
		t.Fatal(err)
	}
	if got := last.header.Get("Idempotency-Key"); got != "" {
		t.Fatalf("Idempotency-Key = %q without the option, want none", got)
	}
}

func TestCurrentTollFreeCarrierFilter(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	on, off := true, false

	if _, err := client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{SubgroupID: "S1", CurrentTollFreeCarrier: &on}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/number?currentTollFreeCarrier=true&subgroupId=S1" {
		t.Fatalf("number list filter lost: %s %s", last.method, last.uri)
	}
	if _, err := client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{GroupID: "G1", CurrentTollFreeCarrier: &off}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/number?currentTollFreeCarrier=false&groupId=G1" {
		t.Fatalf("explicit false lost: %s", last.uri)
	}
	if _, err := client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/number?groupId=G1" {
		t.Fatalf("unset filter sent: %s", last.uri)
	}

	if _, err := client.Brands.GetBrands(ctx, GetBrandsParams{GroupID: "G1", RegistrationType: "TOLL_FREE", CurrentTollFreeCarrier: &on}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/brand?currentTollFreeCarrier=true&groupId=G1&registrationType=TOLL_FREE" {
		t.Fatalf("brand list filter lost: %s %s", last.method, last.uri)
	}
	if _, err := client.Brands.GetBrands(ctx, GetBrandsParams{SubgroupID: "S1", CurrentTollFreeCarrier: &off}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/brand?currentTollFreeCarrier=false&subgroupId=S1" {
		t.Fatalf("explicit false lost: %s", last.uri)
	}
	if _, err := client.Brands.GetBrands(ctx, GetBrandsParams{SubgroupID: "S1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/brand?subgroupId=S1" {
		t.Fatalf("unset filter sent: %s", last.uri)
	}
}

func TestBusinessVerification(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{GroupID: "G/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/brand/business-verification/G%2F1" || last.body != nil {
		t.Fatalf("list contract: %s %s %v", last.method, last.uri, last.body)
	}

	details := map[string]any{
		"businessName": "Acme Inc", "businessType": "CORPORATION", "industry": "RETAIL", "registrationType": "EIN",
		"registrationNumber": "12-3456789", "website": "https://acme.example", "street": "1 Main St", "city": "Austin",
		"state": "TX", "postalCode": "78701", "country": "US",
		"representative": map[string]any{
			"firstName": "Ada", "lastName": "Lee", "email": "ada@acme.example", "phone": "+15125550100",
			"jobTitle": "Owner", "jobLevel": "CEO",
		},
	}
	if _, err := client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{GroupID: "G/1", Details: details}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/brand/business-verification/G%2F1" {
		t.Fatalf("create contract: %s %s", last.method, last.uri)
	}
	sent, _ := json.Marshal(last.body)
	want, _ := json.Marshal(details)
	if string(sent) != string(want) {
		t.Fatalf("details must be the whole body: %s", sent)
	}

	if _, err := client.Brands.GetBusinessVerification(ctx, GetBusinessVerificationParams{GroupID: "G/1", BusinessVerificationID: "BV/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/brand/business-verification/G%2F1/BV%2F1" || last.body != nil {
		t.Fatalf("read contract: %s %s %v", last.method, last.uri, last.body)
	}

	if _, err := client.Brands.UpdateBusinessVerification(ctx, UpdateBusinessVerificationParams{GroupID: "G/1", BusinessVerificationID: "BV/1", Details: details}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "PUT" || last.uri != "/brand/business-verification/G%2F1/BV%2F1" {
		t.Fatalf("update contract: %s %s", last.method, last.uri)
	}
	sent, _ = json.Marshal(last.body)
	if string(sent) != string(want) {
		t.Fatalf("details must be the whole update body: %s", sent)
	}

	refused := []struct {
		name  string
		field string
		call  func() (*Response, error)
	}{
		{"list without groupId", "groupId", func() (*Response, error) {
			return client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{}, nil)
		}},
		{"create without groupId", "groupId", func() (*Response, error) {
			return client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{Details: details}, nil)
		}},
		{"create without details", "details", func() (*Response, error) {
			return client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{GroupID: "G1"}, nil)
		}},
		{"read without groupId", "groupId", func() (*Response, error) {
			return client.Brands.GetBusinessVerification(ctx, GetBusinessVerificationParams{BusinessVerificationID: "BV1"}, nil)
		}},
		{"read without businessVerificationId", "businessVerificationId", func() (*Response, error) {
			return client.Brands.GetBusinessVerification(ctx, GetBusinessVerificationParams{GroupID: "G1"}, nil)
		}},
		{"update without groupId", "groupId", func() (*Response, error) {
			return client.Brands.UpdateBusinessVerification(ctx, UpdateBusinessVerificationParams{BusinessVerificationID: "BV1", Details: details}, nil)
		}},
		{"update without businessVerificationId", "businessVerificationId", func() (*Response, error) {
			return client.Brands.UpdateBusinessVerification(ctx, UpdateBusinessVerificationParams{GroupID: "G1", Details: details}, nil)
		}},
		{"update without details", "details", func() (*Response, error) {
			return client.Brands.UpdateBusinessVerification(ctx, UpdateBusinessVerificationParams{GroupID: "G1", BusinessVerificationID: "BV1"}, nil)
		}},
	}
	for _, refusal := range refused {
		t.Run(refusal.name, func(t *testing.T) {
			*last = captured{}
			_, err := refusal.call()
			if validation, ok := err.(*ValidationError); !ok || validation.Field != refusal.field {
				t.Fatalf("got %v, want a missing %s error", err, refusal.field)
			}
			if last.method != "" {
				t.Fatalf("request sent: %s %s", last.method, last.uri)
			}
		})
	}
}

func TestBusinessVerificationSubgroup(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{GroupID: "G/1", SubgroupID: "S0000001"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/brand/business-verification/G%2F1?subgroupId=S0000001" || last.body != nil {
		t.Fatalf("subgroup list contract: %s %s %v", last.method, last.uri, last.body)
	}

	details := map[string]any{"businessName": "Acme Inc", "registrationType": "EIN", "registrationNumber": "12-3456789"}
	if _, err := client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{GroupID: "G/1", SubgroupID: "S0000001", Details: details}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/brand/business-verification/G%2F1?subgroupId=S0000001" {
		t.Fatalf("subgroup create contract: %s %s", last.method, last.uri)
	}
	sent, _ := json.Marshal(last.body)
	want, _ := json.Marshal(details)
	if string(sent) != string(want) {
		t.Fatalf("details must be the whole body, with no subgroupId: %s", sent)
	}

	// The subgroup ID is escaped as a query value.
	if _, err := client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{GroupID: "G1", SubgroupID: "S&1 2"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/brand/business-verification/G1?subgroupId=S%261+2" {
		t.Fatalf("subgroup not escaped: %s", last.uri)
	}

	// An empty subgroup ID leaves the URL exactly as it was.
	if _, err := client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{GroupID: "G1", SubgroupID: ""}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/brand/business-verification/G1" {
		t.Fatalf("unset subgroup sent on list: %s", last.uri)
	}
	if _, err := client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{GroupID: "G1", SubgroupID: "", Details: details}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/brand/business-verification/G1" {
		t.Fatalf("unset subgroup sent on create: %s", last.uri)
	}

	refused := []struct {
		name  string
		field string
		call  func() (*Response, error)
	}{
		{"list with a subgroup but no groupId", "groupId", func() (*Response, error) {
			return client.Brands.GetBusinessVerifications(ctx, GetBusinessVerificationsParams{SubgroupID: "S0000001"}, nil)
		}},
		{"create with a subgroup but no groupId", "groupId", func() (*Response, error) {
			return client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{SubgroupID: "S0000001", Details: details}, nil)
		}},
		{"create with a subgroup but no details", "details", func() (*Response, error) {
			return client.Brands.CreateBusinessVerification(ctx, CreateBusinessVerificationParams{GroupID: "G1", SubgroupID: "S0000001"}, nil)
		}},
	}
	for _, refusal := range refused {
		t.Run(refusal.name, func(t *testing.T) {
			*last = captured{}
			_, err := refusal.call()
			if validation, ok := err.(*ValidationError); !ok || validation.Field != refusal.field {
				t.Fatalf("got %v, want a missing %s error", err, refusal.field)
			}
			if last.method != "" {
				t.Fatalf("request sent: %s %s", last.method, last.uri)
			}
		})
	}
}

func TestCreateTollFreeBrandLeavesOutEmptyLegalFields(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	// A brand on a business verification sends only what the verification doesn't supply.
	verified := CreateTollFreeBrandParams{BrandData: CreateTollFreeBrandData{
		SubgroupID:  "S0000001",
		DisplayName: "Acme",
		Email:       "support@acme.example",
		Website:     "https://acme.example",
		TollFree: TollFreeBrandDetails{
			LegalEntityType:        "PRIVATE_COMPANY",
			SupportPhone:           "17025550100",
			BusinessVerificationID: "BV0000000001",
		},
	}}
	if _, err := client.Brands.CreateTollFreeBrand(ctx, verified, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/brand/toll-free" {
		t.Fatalf("create contract: %s %s", last.method, last.uri)
	}
	for _, field := range []string{"companyName", "street", "city", "state", "postalCode", "country", "firstName", "lastName"} {
		if _, sent := last.body[field]; sent {
			t.Fatalf("empty %s was sent: %v", field, last.body)
		}
	}
	tollFree, _ := last.body["tollFree"].(map[string]interface{})
	for _, field := range []string{"businessRegistrationType", "taxId", "countryCode", "taxIdIssuingCountry"} {
		if _, sent := tollFree[field]; sent {
			t.Fatalf("empty tollFree.%s was sent: %v", field, tollFree)
		}
	}
	if tollFree["businessVerificationId"] != "BV0000000001" || last.body["subgroupId"] != "S0000001" || last.body["email"] != "support@acme.example" {
		t.Fatalf("set fields lost: %v", last.body)
	}

	// A brand without one still sends every legal field it fills in, and no business verification ID.
	full := CreateTollFreeBrandParams{BrandData: CreateTollFreeBrandData{
		SubgroupID: "S0000001", DisplayName: "Acme", CompanyName: "Acme Inc", Street: "1 Main St", City: "Austin",
		State: "TX", PostalCode: "78701", Email: "support@acme.example", FirstName: "Ada", LastName: "Lee",
		Website: "https://acme.example",
		TollFree: TollFreeBrandDetails{
			BusinessRegistrationType: "EIN", LegalEntityType: "PRIVATE_COMPANY", TaxID: "12-3456789", CountryCode: "US",
			SupportPhone: "17025550100",
		},
	}}
	if _, err := client.Brands.CreateTollFreeBrand(ctx, full, nil); err != nil {
		t.Fatal(err)
	}
	legal := map[string]interface{}{"companyName": "Acme Inc", "street": "1 Main St", "city": "Austin", "state": "TX", "postalCode": "78701", "firstName": "Ada", "lastName": "Lee"}
	for field, value := range legal {
		if last.body[field] != value {
			t.Fatalf("%s = %v, want %v", field, last.body[field], value)
		}
	}
	tollFree, _ = last.body["tollFree"].(map[string]interface{})
	if _, sent := tollFree["businessVerificationId"]; sent || tollFree["taxId"] != "12-3456789" || tollFree["businessRegistrationType"] != "EIN" {
		t.Fatalf("tollFree = %v", tollFree)
	}
}

func TestTollFreeIsvAccount(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	if _, err := client.Brands.GetTollFreeIsvAccount(ctx, GetTollFreeIsvAccountParams{GroupID: "G/1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/brand/toll-free-isv-account/G%2F1" || last.body != nil {
		t.Fatalf("read contract: %s %s %v", last.method, last.uri, last.body)
	}

	account := SetTollFreeIsvAccountParams{
		GroupID:      "G/1",
		AccountSID:   "AC00000000000000000000000000000000",
		APIKeySID:    "SK11111111111111111111111111111111",
		APIKeySecret: "test-key-secret",
	}
	if _, err := client.Brands.SetTollFreeIsvAccount(ctx, account, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "PUT" || last.uri != "/brand/toll-free-isv-account/G%2F1" {
		t.Fatalf("set contract: %s %s", last.method, last.uri)
	}
	sent, _ := json.Marshal(last.body)
	want, _ := json.Marshal(map[string]string{"accountSid": account.AccountSID, "apiKeySid": account.APIKeySID, "apiKeySecret": account.APIKeySecret})
	if string(sent) != string(want) {
		t.Fatalf("body must be the account and its key, with no groupId: %s", sent)
	}

	refused := []struct {
		name  string
		field string
		call  func() (*Response, error)
	}{
		{"read without groupId", "groupId", func() (*Response, error) {
			return client.Brands.GetTollFreeIsvAccount(ctx, GetTollFreeIsvAccountParams{}, nil)
		}},
		{"set without groupId", "groupId", func() (*Response, error) {
			return client.Brands.SetTollFreeIsvAccount(ctx, SetTollFreeIsvAccountParams{AccountSID: account.AccountSID, APIKeySID: account.APIKeySID, APIKeySecret: account.APIKeySecret}, nil)
		}},
		{"set without accountSid", "accountSid", func() (*Response, error) {
			return client.Brands.SetTollFreeIsvAccount(ctx, SetTollFreeIsvAccountParams{GroupID: "G1", APIKeySID: account.APIKeySID, APIKeySecret: account.APIKeySecret}, nil)
		}},
		{"set without apiKeySid", "apiKeySid", func() (*Response, error) {
			return client.Brands.SetTollFreeIsvAccount(ctx, SetTollFreeIsvAccountParams{GroupID: "G1", AccountSID: account.AccountSID, APIKeySecret: account.APIKeySecret}, nil)
		}},
		{"set without apiKeySecret", "apiKeySecret", func() (*Response, error) {
			return client.Brands.SetTollFreeIsvAccount(ctx, SetTollFreeIsvAccountParams{GroupID: "G1", AccountSID: account.AccountSID, APIKeySID: account.APIKeySID}, nil)
		}},
	}
	for _, refusal := range refused {
		t.Run(refusal.name, func(t *testing.T) {
			*last = captured{}
			_, err := refusal.call()
			if validation, ok := err.(*ValidationError); !ok || validation.Field != refusal.field {
				t.Fatalf("got %v, want a missing %s error", err, refusal.field)
			}
			if last.method != "" {
				t.Fatalf("request sent: %s %s", last.method, last.uri)
			}
		})
	}
}
