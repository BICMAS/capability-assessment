package learn

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	assessment_gap "github.com/bicmas-academy/learn/services/assessment-gap/health"
	audit "github.com/bicmas-academy/learn/services/audit/health"
	capability_framework "github.com/bicmas-academy/learn/services/capability-framework/health"
	development "github.com/bicmas-academy/learn/services/development/health"
	gateway "github.com/bicmas-academy/learn/services/gateway/health"
	notify "github.com/bicmas-academy/learn/services/notify/health"
	org_directory "github.com/bicmas-academy/learn/services/org-directory/health"
)

func ready(context.Context) error { return nil }

func TestServiceHealth(t *testing.T) {
	services := []struct {
		name    string
		handler http.Handler
		service string
	}{
		{"gateway", gateway.NewRouter(), gateway.ServiceName},
		{"audit", audit.NewRouter(ready), audit.ServiceName},
		{"notify", notify.NewRouter(ready), notify.ServiceName},
		{"org-directory", org_directory.NewRouter(ready), org_directory.ServiceName},
		{"capability-framework", capability_framework.NewRouter(ready), capability_framework.ServiceName},
		{"assessment-gap", assessment_gap.NewRouter(ready), assessment_gap.ServiceName},
		{"development", development.NewRouter(ready), development.ServiceName},
	}

	for _, service := range services {
		t.Run(service.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			response := httptest.NewRecorder()
			service.handler.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d", response.Code)
			}

			var body map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			if body["status"] != "ok" || body["service"] != service.service {
				t.Fatalf("body = %#v", body)
			}
		})
	}
}
