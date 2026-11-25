package signalhouse

import (
	"context"
	"net/http"
	"reflect"
	"testing"
)

func TestListSearchQueries(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	search := []string{"Acme, Inc", "displayName:dental"}
	reads := []struct {
		name     string
		path     string
		searched func() (*Response, error)
		plain    func() (*Response, error)
		plainURI string
	}{
		{"GetSubgroups", "/subgroup",
			func() (*Response, error) {
				return client.Subgroups.GetSubgroups(ctx, GetSubgroupsParams{GroupID: "G1", Search: search}, nil)
			},
			func() (*Response, error) {
				return client.Subgroups.GetSubgroups(ctx, GetSubgroupsParams{GroupID: "G1", Page: "2"}, nil)
			},
			"/subgroup?groupId=G1&page=2"},
		{"GetBrands", "/brand",
			func() (*Response, error) {
				return client.Brands.GetBrands(ctx, GetBrandsParams{GroupID: "G1", Search: search}, nil)
			},
			func() (*Response, error) {
				return client.Brands.GetBrands(ctx, GetBrandsParams{GroupID: "G1", Page: "2"}, nil)
			},
			"/brand?groupId=G1&page=2"},
		{"GetCampaigns", "/campaign",
			func() (*Response, error) {
				return client.Campaigns.GetCampaigns(ctx, GetCampaignsParams{GroupID: "G1", Search: search}, nil)
			},
			func() (*Response, error) {
				return client.Campaigns.GetCampaigns(ctx, GetCampaignsParams{GroupID: "G1", Page: "2"}, nil)
			},
			"/campaign?groupId=G1&page=2"},
		{"GetPhoneNumbers", "/number",
			func() (*Response, error) {
				return client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{GroupID: "G1", Search: search}, nil)
			},
			func() (*Response, error) {
				return client.Numbers.GetPhoneNumbers(ctx, GetPhoneNumbersParams{GroupID: "G1", Page: "2"}, nil)
			},
			"/number?groupId=G1&page=2"},
	}
	for _, read := range reads {
		t.Run(read.name, func(t *testing.T) {
			if _, err := read.searched(); err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequest("GET", server.URL+last.uri, nil)
			if err != nil {
				t.Fatal(err)
			}
			if req.URL.Path != read.path {
				t.Fatalf("path %q, want %q", req.URL.Path, read.path)
			}
			query := req.URL.Query()
			if got := query["search"]; !reflect.DeepEqual(got, search) {
				t.Fatalf("search %q, want %q (uri %s)", got, search, last.uri)
			}
			if got := query.Get("groupId"); got != "G1" {
				t.Fatalf("groupId %q, want G1", got)
			}

			if _, err := read.plain(); err != nil {
				t.Fatal(err)
			}
			if last.uri != read.plainURI {
				t.Fatalf("uri %q, want %q", last.uri, read.plainURI)
			}
		})
	}
}
