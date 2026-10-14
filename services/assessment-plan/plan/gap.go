package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type GapRecorder interface {
	Record(ctx context.Context, authorization string, attempt GapAttempt) (GapExplanation, error)
}

type GapAttempt struct {
	PersonID           string
	FrameworkVersionID string
	CapabilityCode     string
	Source             string
	Percentage         int
	EvidenceIDs        []string
	RequiredLevel      float64
}

type GapExplanation struct {
	CapabilityCode string
	CurrentLevel   string
	Gap            string
}

type HTTPGap struct {
	BaseURL string
	Client  *http.Client
}

func (g HTTPGap) Record(ctx context.Context, authorization string, attempt GapAttempt) (GapExplanation, error) {
	base := strings.TrimRight(strings.TrimSpace(g.BaseURL), "/")
	if base == "" || strings.TrimSpace(authorization) == "" {
		return GapExplanation{}, ErrProvider
	}
	body, err := json.Marshal(map[string]any{
		"personId":           attempt.PersonID,
		"frameworkVersionId": attempt.FrameworkVersionID,
		"capabilityCode":     attempt.CapabilityCode,
		"source":             attempt.Source,
		"percentage":         attempt.Percentage,
		"evidenceIds":        attempt.EvidenceIDs,
		"requiredLevel":      attempt.RequiredLevel,
	})
	if err != nil {
		return GapExplanation{}, ErrInvalid
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/attempts", bytes.NewReader(body))
	if err != nil {
		return GapExplanation{}, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	request.Header.Set("Authorization", authorization)
	request.Header.Set("Content-Type", "application/json")
	client := g.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(request)
	if err != nil {
		return GapExplanation{}, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return GapExplanation{}, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	if response.StatusCode != http.StatusCreated {
		return GapExplanation{}, ErrUpstream
	}
	var payload struct {
		CapabilityCode string `json:"capabilityCode"`
		CurrentLevel   string `json:"currentLevel"`
		Gap            string `json:"gap"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil || payload.CurrentLevel == "" || payload.Gap == "" {
		return GapExplanation{}, ErrInvalid
	}
	return GapExplanation{
		CapabilityCode: payload.CapabilityCode,
		CurrentLevel:   payload.CurrentLevel,
		Gap:            payload.Gap,
	}, nil
}
