package signalhouse

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const rcsBody = `{"channel":"rcs","contentFormat":"rcs.blocks.v1","content":{"blocks":[{"id":"b1","kind":"text","text":"Hi {{first_name}}"}]},"variables":[{"key":"first_name","type":"text","required":true}],"defaults":{},"fallback":{}}`

func TestTemplateTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	mt := client.Templates

	_, err = mt.Create(ctx, CreateTemplateParams{
		GroupID: "G1", Name: "Welcome", Body: json.RawMessage(rcsBody),
	}, &RequestOptions{IdempotencyKey: "k1"})
	if err != nil || last.method != "POST" || last.uri != "/template" {
		t.Fatalf("create contract: %s %s %v", last.method, last.uri, err)
	}
	if last.header.Get("Idempotency-Key") != "k1" || last.body["groupId"] != "G1" || last.body["name"] != "Welcome" {
		t.Fatalf("create body: %v", last.body)
	}
	if _, present := last.body["tag"]; present {
		t.Fatal("empty tag must be omitted on create")
	}
	sent, _ := json.Marshal(last.body["body"])
	var want, got any
	_ = json.Unmarshal([]byte(rcsBody), &want)
	_ = json.Unmarshal(sent, &got)
	if string(mustJSON(t, want)) != string(mustJSON(t, got)) {
		t.Fatalf("body altered: %s", sent)
	}
	if _, err := mt.Create(ctx, CreateTemplateParams{GroupID: "G1", Name: "Welcome"}, nil); err == nil {
		t.Fatal("missing body must fail before transport")
	}

	if _, err := mt.List(ctx, GetTemplatesParams{
		GroupID: "G1", State: "DRAFT", Channel: "rcs", Tag: "Promo", Search: "wel",
		SortBy: "name", SortOrder: "asc", Page: "2", Limit: "10",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/template?channel=rcs&groupId=G1&limit=10&page=2&search=wel&sortBy=name&sortOrder=asc&state=DRAFT&tag=Promo" {
		t.Fatalf("list contract: %s", last.uri)
	}
	if _, err := mt.List(ctx, GetTemplatesParams{}, nil); err == nil {
		t.Fatal("missing groupId must fail before transport")
	}

	if _, err := mt.Get(ctx, GetTemplateParams{TemplateID: "TPL/1", GroupID: "G1", Version: "draft"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/template/TPL%2F1?groupId=G1&version=draft" {
		t.Fatalf("get contract: %s", last.uri)
	}
	if _, err := mt.Get(ctx, GetTemplateParams{GroupID: "G1"}, nil); err == nil {
		t.Fatal("missing templateId must fail before transport")
	}

	if _, err := mt.Update(ctx, UpdateTemplateParams{TemplateID: "TPL1", GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "PUT" || last.uri != "/template/TPL1" || len(last.body) != 1 || last.body["groupId"] != "G1" {
		t.Fatalf("update contract: %s %s %v", last.method, last.uri, last.body)
	}
	tag := "Promo"
	if _, err := mt.Update(ctx, UpdateTemplateParams{
		TemplateID: "TPL1", GroupID: "G1", Name: "Renamed", Tag: &tag, Body: json.RawMessage(rcsBody),
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.body["groupId"] != "G1" || last.body["name"] != "Renamed" || last.body["tag"] != "Promo" || last.body["body"] == nil {
		t.Fatalf("update fields lost: %v", last.body)
	}
	if _, err := mt.Update(ctx, UpdateTemplateParams{TemplateID: "TPL1", GroupID: "G1", ClearTag: true}, nil); err != nil {
		t.Fatal(err)
	}
	if v, present := last.body["tag"]; !present || v != nil {
		t.Fatalf("tag must be sent as null to clear: %v", last.body)
	}
	if _, err := mt.Update(ctx, UpdateTemplateParams{GroupID: "G1"}, nil); err == nil {
		t.Fatal("missing templateId must fail before transport")
	}
	if _, err := mt.Update(ctx, UpdateTemplateParams{TemplateID: "TPL1"}, nil); err == nil {
		t.Fatal("missing groupId must fail before transport")
	}

	if _, err := mt.Publish(ctx, PublishTemplateParams{TemplateID: "TPL1", GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "POST" || last.uri != "/template/TPL1/publish" || len(last.body) != 1 || last.body["groupId"] != "G1" {
		t.Fatalf("publish contract: %s %v", last.uri, last.body)
	}
	if _, err := mt.Publish(ctx, PublishTemplateParams{TemplateID: "TPL1"}, nil); err == nil {
		t.Fatal("missing groupId must fail before transport")
	}

	if _, err := mt.Archive(ctx, ArchiveTemplateParams{TemplateID: "TPL1", GroupID: "G1"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/template/TPL1/archive" || len(last.body) != 1 || last.body["groupId"] != "G1" {
		t.Fatalf("archive contract: %s %v", last.uri, last.body)
	}
	if _, err := mt.Archive(ctx, ArchiveTemplateParams{GroupID: "G1"}, nil); err == nil {
		t.Fatal("missing templateId must fail before transport")
	}

	if _, err := mt.Versions(ctx, GetVersionsParams{TemplateID: "TPL1", GroupID: "G1", Page: "1", Limit: "5"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/template/TPL1/versions?groupId=G1&limit=5&page=1" {
		t.Fatalf("versions contract: %s", last.uri)
	}

	if _, err := mt.Validate(ctx, ValidateTemplateParams{TemplateID: "TPL1", GroupID: "G1", Version: "3"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/template/TPL1/validate" || last.body["version"] != float64(3) || last.body["body"] != nil {
		t.Fatalf("validate version contract: %s %v", last.uri, last.body)
	}
	if _, err := mt.Validate(ctx, ValidateTemplateParams{TemplateID: "TPL1", GroupID: "G1", Version: "published"}, nil); err != nil || last.body["version"] != "published" {
		t.Fatalf("validate selector: %v %v", last.body, err)
	}
	if _, err := mt.Validate(ctx, ValidateTemplateParams{TemplateID: "TPL1", GroupID: "G1", Body: json.RawMessage(rcsBody)}, nil); err != nil {
		t.Fatal(err)
	}
	if _, present := last.body["version"]; present || last.body["body"] == nil {
		t.Fatalf("validate body contract: %v", last.body)
	}
	if _, err := mt.Validate(ctx, ValidateTemplateParams{TemplateID: "TPL1", GroupID: "G1"}, nil); err == nil {
		t.Fatal("neither body nor version must fail before transport")
	}
	if _, err := mt.Validate(ctx, ValidateTemplateParams{TemplateID: "TPL1", GroupID: "G1", Body: json.RawMessage(rcsBody), Version: "draft"}, nil); err == nil {
		t.Fatal("both body and version must fail before transport")
	}

	if _, err := mt.Capabilities(ctx, GetCapabilitiesParams{GroupID: "G1", Channel: "rcs", Provider: "sinch"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/template/capabilities?channel=rcs&groupId=G1&provider=sinch" {
		t.Fatalf("capabilities contract: %s", last.uri)
	}
}

func TestTemplateMediaUpload(t *testing.T) {
	var method, path, groupID, filename, content, contentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, contentType = r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type")
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			groupID = r.FormValue("groupId")
			if f, header, err := r.FormFile("file"); err == nil {
				filename = header.Filename
				data, _ := io.ReadAll(f)
				content = string(data)
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"mediaId":"MED0000000001"}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()

	for _, door := range []struct {
		path   string
		upload func(UploadMediaParams) (*Response, error)
	}{
		{"/template/media", func(p UploadMediaParams) (*Response, error) { return client.Templates.UploadMedia(ctx, p, nil) }},
	} {
		resp, err := door.upload(UploadMediaParams{GroupID: "G1", File: &FileUpload{Filename: "hero.png", Reader: strings.NewReader("PNG")}})
		if err != nil || resp.Status != http.StatusCreated {
			t.Fatalf("upload failed: %v %v", resp, err)
		}
		if method != "POST" || path != door.path || !strings.HasPrefix(contentType, "multipart/form-data") {
			t.Fatalf("upload contract: %s %s %s", method, path, contentType)
		}
		if groupID != "G1" || filename != "hero.png" || content != "PNG" {
			t.Fatalf("upload form: %q %q %q", groupID, filename, content)
		}
		if _, err := door.upload(UploadMediaParams{GroupID: "G1"}); err == nil {
			t.Fatal("missing file must fail before transport")
		}
	}
}

// mustJSON re-marshals v so two decoded documents compare by canonical key order.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
