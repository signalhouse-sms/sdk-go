package signalhouse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestComplianceTransport(t *testing.T) {
	server, last := newCaptureServer(t)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	compliance := client.Compliance

	if _, err := compliance.GetCases(ctx, GetCasesParams{}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/compliance/cases" || last.body != nil {
		t.Fatalf("list contract: %s %s %v", last.method, last.uri, last.body)
	}
	suspended := true
	if _, err := compliance.GetCases(ctx, GetCasesParams{
		GroupID: "G1", Status: "OPEN", Stage: "customer", ObjectType: "CAMPAIGN", ObjectID: "C1", SubgroupID: "S1",
		Suspended: &suspended, From: "2026-11-01", To: "2026-11-13", Sort: "fixBy", Page: "2", Limit: "50",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/compliance/cases?from=2026-11-01&groupId=G1&limit=50&objectId=C1&objectType=CAMPAIGN&page=2&sort=fixBy&stage=customer&status=OPEN&subgroupId=S1&suspended=true&to=2026-11-13" {
		t.Fatalf("list query: %s", last.uri)
	}
	notSuspended := false
	if _, err := compliance.GetCases(ctx, GetCasesParams{Suspended: &notSuspended}, nil); err != nil {
		t.Fatal(err)
	}
	if last.uri != "/compliance/cases?suspended=false" {
		t.Fatalf("list query sends suspended=false: %s", last.uri)
	}

	if _, err := compliance.GetCase(ctx, GetCaseParams{CaseID: "CASE-1/2"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/compliance/cases/CASE-1%2F2" || last.body != nil {
		t.Fatalf("read contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, err := compliance.GetCase(ctx, GetCaseParams{}, nil); err == nil || !strings.Contains(err.Error(), "caseId") {
		t.Fatalf("missing caseId must fail before transport: %v", err)
	}

	if _, err := compliance.AnswerRequirement(ctx, AnswerRequirementParams{
		CaseID: "CASE-1", RequirementID: "CASE-1-1/x", Answer: "Removed the number", FileIDs: []string{"F1", "F2"},
	}, &RequestOptions{IdempotencyKey: "k1"}); err != nil {
		t.Fatal(err)
	}
	if last.method != "PUT" || last.uri != "/compliance/cases/CASE-1/requirements/CASE-1-1%2Fx" || last.header.Get("Idempotency-Key") != "k1" {
		t.Fatalf("answer contract: %s %s", last.method, last.uri)
	}
	if string(mustJSON(t, last.body)) != `{"answer":"Removed the number","fileIds":["F1","F2"]}` {
		t.Fatalf("answer body: %s", mustJSON(t, last.body))
	}
	if _, err := compliance.AnswerRequirement(ctx, AnswerRequirementParams{CaseID: "CASE-1", RequirementID: "CASE-1-1", Answer: "Done"}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"answer":"Done"}` {
		t.Fatalf("answer body omits unset fileIds: %s", mustJSON(t, last.body))
	}
	if _, err := compliance.AnswerRequirement(ctx, AnswerRequirementParams{CaseID: "CASE-1", RequirementID: "CASE-1-1", FileIDs: []string{"F1"}}, nil); err != nil {
		t.Fatal(err)
	}
	if string(mustJSON(t, last.body)) != `{"fileIds":["F1"]}` {
		t.Fatalf("answer body omits unset answer: %s", mustJSON(t, last.body))
	}
	if _, err := compliance.AnswerRequirement(ctx, AnswerRequirementParams{RequirementID: "CASE-1-1"}, nil); err == nil || !strings.Contains(err.Error(), "caseId") {
		t.Fatalf("missing caseId must fail before transport: %v", err)
	}
	if _, err := compliance.AnswerRequirement(ctx, AnswerRequirementParams{CaseID: "CASE-1"}, nil); err == nil || !strings.Contains(err.Error(), "requirementId") {
		t.Fatalf("missing requirementId must fail before transport: %v", err)
	}

	if _, err := compliance.GetFile(ctx, GetFileParams{FileID: "CF1/2"}, nil); err != nil {
		t.Fatal(err)
	}
	if last.method != "GET" || last.uri != "/compliance/files/CF1%2F2" || last.body != nil {
		t.Fatalf("download contract: %s %s %v", last.method, last.uri, last.body)
	}
	if _, err := compliance.GetFile(ctx, GetFileParams{}, nil); err == nil || !strings.Contains(err.Error(), "fileId") {
		t.Fatalf("missing fileId must fail before transport: %v", err)
	}
}

func TestComplianceFiles(t *testing.T) {
	var method, path, contentType, caseID, filename, content string
	var hasRequirementID bool
	var requirementID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, contentType = r.Method, r.URL.RequestURI(), r.Header.Get("Content-Type")
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/csv")
			_, _ = w.Write([]byte("phone,optIn\n+12025550123,yes\n"))
			return
		}
		caseID, requirementID, hasRequirementID, filename, content = "", "", false, "", ""
		if err := r.ParseMultipartForm(1 << 20); err == nil {
			caseID = r.FormValue("caseId")
			_, hasRequirementID = r.MultipartForm.Value["requirementId"]
			requirementID = r.FormValue("requirementId")
			if f, header, err := r.FormFile("file"); err == nil {
				filename = header.Filename
				data, _ := io.ReadAll(f)
				content = string(data)
			}
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"fileId":"CF1","fileName":"optins.csv","mimeType":"text/csv","size":3}`))
	}))
	t.Cleanup(server.Close)
	client, err := NewClient(Config{APIKey: "test", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	compliance := client.Compliance

	resp, err := compliance.UploadFile(ctx, UploadFileParams{
		CaseID: "CASE-1", RequirementID: "CASE-1-1", File: &FileUpload{Filename: "optins.csv", Reader: strings.NewReader("CSV")},
	}, nil)
	if err != nil || resp.Status != http.StatusCreated {
		t.Fatalf("upload failed: %v %v", resp, err)
	}
	if method != "POST" || path != "/compliance/files" || !strings.HasPrefix(contentType, "multipart/form-data") {
		t.Fatalf("upload contract: %s %s %s", method, path, contentType)
	}
	if caseID != "CASE-1" || requirementID != "CASE-1-1" || filename != "optins.csv" || content != "CSV" {
		t.Fatalf("upload form: %q %q %q %q", caseID, requirementID, filename, content)
	}
	if _, err := compliance.UploadFile(ctx, UploadFileParams{CaseID: "CASE-1", File: &FileUpload{Filename: "optins.csv", Reader: strings.NewReader("CSV")}}, nil); err != nil {
		t.Fatal(err)
	}
	if caseID != "CASE-1" || hasRequirementID {
		t.Fatalf("upload form omits unset requirementId: %q %v", caseID, hasRequirementID)
	}
	if _, err := compliance.UploadFile(ctx, UploadFileParams{File: &FileUpload{Filename: "optins.csv", Reader: strings.NewReader("CSV")}}, nil); err == nil || !strings.Contains(err.Error(), "caseId") {
		t.Fatalf("missing caseId must fail before transport: %v", err)
	}
	if _, err := compliance.UploadFile(ctx, UploadFileParams{CaseID: "CASE-1"}, nil); err == nil || !strings.Contains(err.Error(), "file") {
		t.Fatalf("missing file must fail before transport: %v", err)
	}
	if _, err := compliance.UploadFile(ctx, UploadFileParams{CaseID: "CASE-1", File: &FileUpload{Filename: "optins.csv"}}, nil); err == nil || !strings.Contains(err.Error(), "file") {
		t.Fatalf("missing reader must fail before transport: %v", err)
	}

	resp, err = compliance.GetFile(ctx, GetFileParams{FileID: "CF1"}, nil)
	if err != nil || !resp.Success || method != "GET" || path != "/compliance/files/CF1" {
		t.Fatalf("download contract: %v %v %s %s", resp, err, method, path)
	}
	if string(resp.Data) != "phone,optIn\n+12025550123,yes\n" {
		t.Fatalf("download bytes: %q", resp.Data)
	}
}
