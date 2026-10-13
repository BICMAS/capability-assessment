package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/bicmas-academy/learn/services/capability-framework/blob"
	"github.com/bicmas-academy/learn/services/capability-framework/csvimport"
	"github.com/bicmas-academy/learn/services/token"
)

func TestSalesImportBlocksReviewUntilTheDuplicateIsRejected(t *testing.T) {
	payload, err := os.ReadFile("../testdata/sales.csv")
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemory()
	files := blob.NewMemory()
	claims := token.Claims{
		PersonID:       "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11",
		OrganisationID: "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21",
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	}

	imported := postImport(t, store, files, claims, payload)
	if imported.Version == nil || imported.Version.Status != StatusDraft || imported.Version.FrameworkName != "Sales Framework" || imported.Version.VersionNumber != 1 {
		t.Fatalf("version = %#v", imported.Version)
	}
	if len(imported.Proposals) != 1 || imported.Proposals[0].Kind != csvimport.ProposalDuplicateCode || imported.Proposals[0].CapabilityCode != "NEG" || imported.Proposals[0].Status != ProposalPending {
		t.Fatalf("proposals = %#v", imported.Proposals)
	}
	if len(filesObjects(files)) != 1 {
		t.Fatal("uploaded file was not stored")
	}
	roleNames := map[string]bool{}
	for _, role := range imported.Version.Roles {
		roleNames[role.Name] = true
	}
	if !roleNames["Area Sales Manager"] {
		t.Fatalf("roles = %#v", imported.Version.Roles)
	}
	stored, err := store.GetVersion(context.Background(), claims.OrganisationID, imported.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Requirements) != 5 {
		t.Fatalf("requirements = %#v", stored.Requirements)
	}
	indicators := map[string]int{}
	for _, capability := range stored.Capabilities {
		count := 0
		for _, level := range capability.Levels {
			count += len(level.Indicators)
		}
		indicators[capability.Name] = count
	}
	if indicators["Negotiation"] != 5 || indicators["Sales Leadership"] != 0 || indicators["Customer Management"] != 0 || indicators["Commercial Acumen"] != 0 || indicators["Execution Excellence"] != 0 {
		t.Fatalf("indicators = %#v", indicators)
	}

	denied := callVersion(t, SubmitForReview(store), http.MethodPost, claims, imported.Version.Id, "")
	if denied.Code != http.StatusConflict {
		t.Fatalf("submit status = %d", denied.Code)
	}

	decision := callProposal(t, DecideProposal(store), claims, imported.Version.Id, imported.Proposals[0].Id, `{"status":"rejected"}`)
	if decision.Code != http.StatusOK {
		t.Fatalf("reject status = %d body = %s", decision.Code, decision.Body.String())
	}
	accepted := callVersion(t, SubmitForReview(store), http.MethodPost, claims, imported.Version.Id, "")
	if accepted.Code != http.StatusOK {
		t.Fatalf("submit after reject = %d body = %s", accepted.Code, accepted.Body.String())
	}
	var version map[string]any
	if err := json.Unmarshal(accepted.Body.Bytes(), &version); err != nil {
		t.Fatal(err)
	}
	if version["status"] != StatusInReview {
		t.Fatalf("status = %#v", version["status"])
	}
}

func TestMissingRequiredLevelIsAProposal(t *testing.T) {
	parsed, err := csvimport.Parse(bytes.NewReader([]byte(
		"framework version,department,role code,role name,capability code,capability name,capability type,definition,required level,proficiency levels,behavioural indicators,assessment methods,learning asset ids,practice scenario ids,workplace mission,weight,governance status\n" +
			"1,Sales,ASM,Area Sales Manager,DISC,Discovery,skill,Finds the buyer need,,,,manager assessment,asset-2,scenario-2,Run a discovery call,1,draft\n",
	)))
	if err != nil {
		t.Fatal(err)
	}
	if len(parsed.RowErrors) != 0 || len(parsed.Proposals) != 1 || parsed.Proposals[0].Kind != csvimport.ProposalMissingLevel {
		t.Fatalf("parsed = %#v", parsed)
	}
	store := NewMemory()
	result, err := store.ApplyImport(context.Background(), "org", "imports/discovery.csv", parsed)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SubmitForReview(context.Background(), "org", result.Version.ID); !errors.Is(err, ErrPending) {
		t.Fatalf("submit error = %v", err)
	}
}

func TestMissingColumnsDoNotCreateADraft(t *testing.T) {
	store := NewMemory()
	files := blob.NewMemory()
	response := postImport(t, store, files, token.Claims{
		PersonID:       "ada",
		OrganisationID: "org",
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	}, []byte("role name\nArea Sales Manager\n"))
	if response.Version != nil || len(response.MissingColumns) == 0 {
		t.Fatalf("response = %#v", response)
	}
	versions, err := store.ListVersions(context.Background(), "org")
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 || len(filesObjects(files)) != 1 {
		t.Fatalf("versions = %d objects = %d", len(versions), len(filesObjects(files)))
	}
}

type importPayload struct {
	Version *struct {
		Id            string
		Status        string
		FrameworkName string
		VersionNumber int
		Roles         []struct{ Name string }
	}
	Proposals      []struct{ Id, Kind, CapabilityCode, Status string }
	MissingColumns []string
}

func postImport(t *testing.T, store Store, files *blob.Memory, claims token.Claims, payload []byte) importPayload {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "sales.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/imports", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request = request.WithContext(token.WithClaims(request.Context(), claims))
	response := httptest.NewRecorder()
	ImportCSV(store, files).ServeHTTP(response, request)
	if response.Code != http.StatusCreated && response.Code != http.StatusBadRequest {
		t.Fatalf("import status = %d body = %s", response.Code, response.Body.String())
	}
	var imported importPayload
	if err := json.Unmarshal(response.Body.Bytes(), &imported); err != nil {
		t.Fatal(err)
	}
	return imported
}

func callVersion(t *testing.T, handler http.Handler, method string, claims token.Claims, versionID, proposalID string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, "/", nil)
	request = withRoute(request, claims, "versionId", versionID, "proposalId", proposalID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func callProposal(t *testing.T, handler http.Handler, claims token.Claims, versionID, proposalID, payload string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(payload))
	request = withRoute(request, claims, "versionId", versionID, "proposalId", proposalID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func withRoute(request *http.Request, claims token.Claims, pairs ...string) *http.Request {
	route := chi.NewRouteContext()
	for i := 0; i < len(pairs); i += 2 {
		route.URLParams.Add(pairs[i], pairs[i+1])
	}
	ctx := token.WithClaims(request.Context(), claims)
	ctx = context.WithValue(ctx, chi.RouteCtxKey, route)
	return request.WithContext(ctx)
}

func filesObjects(files *blob.Memory) map[string][]byte {
	return files.Objects()
}
