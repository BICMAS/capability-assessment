package plan

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPDrafterSendsTheMethodAsAFixedConstraint(t *testing.T) {
	var seenAuth string
	var seenBody string
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		seenBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"{\"skills\":[]}"}}]}`))
	}))
	defer provider.Close()

	drafter := HTTPDrafter{APIKey: "test-key", Model: "draft-model", Endpoint: provider.URL, Client: provider.Client()}
	raw, err := drafter.Draft(context.Background(), DraftRequest{PromptVersion: PromptVersion, Requirements: []draftConstraint{{
		RequirementID: "req-1", CapabilityName: "Negotiation", RequiredLevel: 4, RecommendedMethod: MethodAvatarRoleplay, ProficiencyLevels: []int{4},
	}}})
	if err != nil {
		t.Fatal(err)
	}
	if seenAuth != "Bearer test-key" {
		t.Fatalf("auth = %q", seenAuth)
	}
	if !strings.Contains(seenBody, MethodAvatarRoleplay) || !strings.Contains(seenBody, "Return JSON only") || !strings.Contains(seenBody, "draft-model") {
		t.Fatalf("body = %s", seenBody)
	}
	if strings.Contains(seenBody, "test-key") {
		t.Fatal("key was sent in the request body")
	}
	if !strings.Contains(string(raw), `"skills"`) {
		t.Fatalf("draft = %s", raw)
	}
}

func TestHTTPDrafterRejectsInvalidJSON(t *testing.T) {
	provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"not json"}}]}`))
	}))
	defer provider.Close()
	drafter := HTTPDrafter{APIKey: "test-key", Endpoint: provider.URL, Client: provider.Client()}
	if _, err := drafter.Draft(context.Background(), DraftRequest{}); err != ErrInvalid {
		t.Fatalf("err = %v", err)
	}
}
