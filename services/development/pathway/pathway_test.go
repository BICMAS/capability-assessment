package pathway

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	eliID          = "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22"
	capabilityID   = "09ac779b-f2ba-4b78-83c2-4844b0a6e690"
	webhookSecret  = "dev-only-webhook-not-a-production-secret"
	lmsBase        = "https://lms.bicmas.example"
)

func TestLearnDeepLinkContainsThePlanItemID(t *testing.T) {
	store := negotiationPathway(t)
	got, err := store.GetPathway(context.Background(), organisationID, eliID, capabilityID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) < 2 || got.Items[0].Stage != StageLearn || got.Items[0].Title != "Negotiation Fundamentals" {
		t.Fatalf("first item = %#v", got.Items)
	}
	if !strings.Contains(got.Items[0].DeepLink, got.Items[0].ID) {
		t.Fatalf("deep link %s does not contain %s", got.Items[0].DeepLink, got.Items[0].ID)
	}
	for _, item := range got.Items[1:] {
		if item.Status != StatusUpcoming || item.DeepLink != "" {
			t.Fatalf("later stage = %#v", item)
		}
	}
}

func TestWebhookTwiceLeavesOneCompletion(t *testing.T) {
	store := negotiationPathway(t)
	got, err := store.GetPathway(context.Background(), organisationID, eliID, capabilityID)
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingPublisher{}
	handler := NewRouter(func(context.Context) error { return nil }, []byte("jwt-secret"), []byte(webhookSecret), store, publisher)
	body := []byte(`{"organisationId":"` + organisationID + `","personId":"` + eliID + `","planItemId":"` + got.Items[0].ID + `","completedAt":"2026-10-11T22:40:00Z"}`)
	first := postWebhook(t, handler, body)
	second := postWebhook(t, handler, body)
	if first.Code != http.StatusOK || second.Code != http.StatusOK {
		t.Fatalf("statuses = %d and %d", first.Code, second.Code)
	}
	if store.completionCount(got.Items[0].ID) != 1 {
		t.Fatalf("completions = %d", store.completionCount(got.Items[0].ID))
	}
	if publisher.calls != 1 || publisher.event.PlanItemID != got.Items[0].ID || publisher.event.Type != "learning.completed" || publisher.event.PersonID != eliID {
		t.Fatalf("event calls = %d %#v", publisher.calls, publisher.event)
	}
	reloaded, err := store.GetPathway(context.Background(), organisationID, eliID, capabilityID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Items[0].Status != StatusComplete {
		t.Fatalf("status = %s", reloaded.Items[0].Status)
	}
	for _, item := range reloaded.Items[1:] {
		if item.Status != StatusUpcoming {
			t.Fatalf("later stage = %#v", item)
		}
	}
}

func negotiationPathway(t *testing.T) *Memory {
	t.Helper()
	store := NewMemory(lmsBase)
	if _, err := store.MapCourse(context.Background(), MappingInput{
		OrganisationID:   organisationID,
		ExternalCourseID: "negotiation-fundamentals",
		CapabilityID:     capabilityID,
		TargetLevel:      4,
		Title:            "Negotiation Fundamentals",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreatePathway(context.Background(), PathwayInput{
		OrganisationID: organisationID,
		PersonID:       eliID,
		CapabilityID:   capabilityID,
		TargetLevel:    4,
	}); err != nil {
		t.Fatal(err)
	}
	return store
}

func postWebhook(t *testing.T, handler http.Handler, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/webhooks/completions", bytes.NewReader(body))
	request.Header.Set("X-Learning-Signature", signature(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func signature(body []byte) string {
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

type recordingPublisher struct {
	calls int
	event LearningCompleted
}

func (r *recordingPublisher) PublishLearningCompleted(_ context.Context, event LearningCompleted) error {
	r.calls++
	r.event = event
	return nil
}
