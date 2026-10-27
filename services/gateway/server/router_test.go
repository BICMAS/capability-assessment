package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/bicmas-academy/learn/services/gateway/auth"
	"github.com/bicmas-academy/learn/services/gateway/config"
)

func TestDevLoginEnabledRegistersRoute(t *testing.T) {
	cfg := mustConfig(t, map[string]string{
		config.EnvAppEnv:          "development",
		config.EnvDevLoginEnabled: "true",
		config.EnvDevJWTSecret:    "dev-only-not-a-production-secret",
	})
	materials, err := auth.NewMaterials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	publisher := &recordingPublisher{}
	handler := NewRouter(Options{Verifier: materials.Verifier, Publisher: publisher, DevLogin: materials.Issuer})

	body, _ := json.Marshal(map[string]string{"person": "ada"})
	request := httptest.NewRequest(http.MethodPost, "/dev/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	if len(publisher.events) != 1 || publisher.events[0].PersonID != auth.AdaPersonID {
		t.Fatalf("events = %#v", publisher.events)
	}
}

func TestDevLoginDisabledReturns404(t *testing.T) {
	cfg := mustConfig(t, map[string]string{
		config.EnvAppEnv:          "development",
		config.EnvDevLoginEnabled: "false",
	})
	materials, err := auth.NewMaterials(cfg)
	if err != nil {
		t.Fatal(err)
	}
	handler := NewRouter(Options{Verifier: materials.Verifier, DevLogin: materials.Issuer})

	body, _ := json.Marshal(map[string]string{"person": "ada"})
	request := httptest.NewRequest(http.MethodPost, "/dev/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("status = %d", response.Code)
	}
}

func TestUnknownDemoIdentityRejected(t *testing.T) {
	dev := testDev(t)
	handler := NewRouter(Options{Verifier: dev, Publisher: &recordingPublisher{}, DevLogin: dev})
	body, _ := json.Marshal(map[string]string{"person": "root"})
	request := httptest.NewRequest(http.MethodPost, "/dev/login", bytes.NewReader(body))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", response.Code)
	}
}

func mustConfig(t *testing.T, values map[string]string) config.Config {
	t.Helper()
	cfg, err := config.LoadFrom(func(key string) string { return values[key] })
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func testDev(t *testing.T) *auth.DevHMAC {
	t.Helper()
	dev, err := auth.NewDevHMAC([]byte("dev-only-not-a-production-secret"), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	return dev
}

type recordingPublisher struct {
	events []auth.PersonSignedIn
}

func (p *recordingPublisher) PublishPersonSignedIn(_ context.Context, event auth.PersonSignedIn) error {
	p.events = append(p.events, event)
	return nil
}
