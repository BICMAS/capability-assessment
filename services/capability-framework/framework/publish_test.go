package framework

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/bicmas-academy/learn/services/capability-framework/blob"
	"github.com/bicmas-academy/learn/services/token"
)

func TestPublishFreezesSalesFrameworkAndRecordsAda(t *testing.T) {
	payload, err := os.ReadFile("../testdata/sales.csv")
	if err != nil {
		t.Fatal(err)
	}
	store := NewMemory()
	claims := token.Claims{
		PersonID:       "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11",
		OrganisationID: "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21",
		Roles:          []string{"admin"},
		ManagerOf:      []string{},
	}
	imported := postImport(t, store, blob.NewMemory(), claims, payload)
	decision := callProposal(t, DecideProposal(store), claims, imported.Version.Id, imported.Proposals[0].Id, `{"status":"rejected"}`)
	if decision.Code != http.StatusOK {
		t.Fatalf("reject = %d", decision.Code)
	}
	version, err := store.GetVersion(context.Background(), claims.OrganisationID, imported.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	indicatorID := negotiationIndicator(t, version)
	saved := callIndicator(t, UpdateIndicator(store), claims, version.ID, indicatorID, `{"statement":"Agrees the commercial terms with the buyer."}`)
	if saved.Code != http.StatusOK {
		t.Fatalf("save = %d body = %s", saved.Code, saved.Body.String())
	}

	approved := callVersion(t, Approve(store), http.MethodPost, claims, version.ID, "")
	if approved.Code != http.StatusOK {
		t.Fatalf("approve = %d body = %s", approved.Code, approved.Body.String())
	}
	publisher := &recordingPublisher{}
	published := callVersion(t, Publish(store, publisher), http.MethodPost, claims, version.ID, "")
	if published.Code != http.StatusOK {
		t.Fatalf("publish = %d body = %s", published.Code, published.Body.String())
	}
	if len(publisher.events) != 1 || publisher.events[0].ActorPersonID != claims.PersonID || publisher.events[0].Type != "framework.published" || publisher.events[0].FrameworkVersionID != version.ID {
		t.Fatalf("event = %#v", publisher.events)
	}

	second := callIndicator(t, UpdateIndicator(store), claims, version.ID, indicatorID, `{"statement":"A later edit."}`)
	if second.Code != http.StatusConflict {
		t.Fatalf("second save = %d", second.Code)
	}
	frozen, err := store.GetVersion(context.Background(), claims.OrganisationID, version.ID)
	if err != nil {
		t.Fatal(err)
	}
	if frozen.Status != StatusPublished || negotiationStatement(t, frozen) != "Agrees the commercial terms with the buyer." {
		t.Fatalf("frozen = %#v", frozen)
	}

	employee := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/published-frameworks", nil)
	request = request.WithContext(token.WithClaims(request.Context(), token.Claims{
		PersonID:       "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22",
		OrganisationID: claims.OrganisationID,
		Roles:          []string{"employee"},
		ManagerOf:      []string{},
	}))
	ListPublished(store).ServeHTTP(employee, request)
	if employee.Code != http.StatusOK {
		t.Fatalf("employee read = %d", employee.Code)
	}
	var visible []map[string]any
	if err := json.Unmarshal(employee.Body.Bytes(), &visible); err != nil {
		t.Fatal(err)
	}
	if len(visible) != 1 || visible[0]["frameworkName"] != "Sales Framework" || visible[0]["versionNumber"] != float64(1) {
		t.Fatalf("visible = %#v", visible)
	}

	other := httptest.NewRecorder()
	otherRequest := httptest.NewRequest(http.MethodGet, "/published-frameworks", nil)
	otherRequest = otherRequest.WithContext(token.WithClaims(otherRequest.Context(), token.Claims{
		PersonID:       "other",
		OrganisationID: "other-org",
		Roles:          []string{"employee"},
		ManagerOf:      []string{},
	}))
	ListPublished(store).ServeHTTP(other, otherRequest)
	if other.Body.String() != "[]\n" && other.Body.String() != "[]" {
		t.Fatalf("other organisation = %s", other.Body.String())
	}
}

func TestApproveRejectsAPendingProposal(t *testing.T) {
	store := NewMemory()
	version, err := store.CreateDraft(context.Background(), "org", "Sales Framework")
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.versions[0].Proposals = []Proposal{{ID: "proposal", Status: ProposalPending}}
	store.mu.Unlock()
	if _, err := store.Approve(context.Background(), "org", version.ID); !errors.Is(err, ErrPending) {
		t.Fatalf("approve error = %v", err)
	}
}

func negotiationIndicator(t *testing.T, version Version) string {
	t.Helper()
	for _, capability := range version.Capabilities {
		if capability.Name != "Negotiation" {
			continue
		}
		for _, level := range capability.Levels {
			if level.Level == 4 && len(level.Indicators) == 1 {
				return level.Indicators[0].ID
			}
		}
	}
	t.Fatal("level 4 negotiation indicator was not imported")
	return ""
}

func negotiationStatement(t *testing.T, version Version) string {
	t.Helper()
	for _, capability := range version.Capabilities {
		if capability.Name != "Negotiation" {
			continue
		}
		for _, level := range capability.Levels {
			if level.Level == 4 && len(level.Indicators) == 1 {
				return level.Indicators[0].Statement
			}
		}
	}
	t.Fatal("level 4 negotiation indicator is missing")
	return ""
}

func callIndicator(t *testing.T, handler http.Handler, claims token.Claims, versionID, indicatorID, payload string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(payload))
	request = withRoute(request, claims, "versionId", versionID, "indicatorId", indicatorID)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

type recordingPublisher struct {
	events []FrameworkPublished
}

func (r *recordingPublisher) PublishFrameworkPublished(_ context.Context, event FrameworkPublished) error {
	r.events = append(r.events, event)
	return nil
}
