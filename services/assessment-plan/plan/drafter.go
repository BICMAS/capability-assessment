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

const (
	defaultModelName = "gpt-4.1-mini"
	defaultModelURL  = "https://api.openai.com/v1/chat/completions"
)

const draftPrompt = `Return JSON only. Do not use markdown.
Write instructions, assessment items, and rubric anchors for every requirement.
The recommendedMethod on each requirement is fixed. Copy it unchanged.
Use only the proficiencyLevels listed for that requirement.
Shape:
{"skills":[{"requirementId":"","recommendedMethod":"","instructions":"","assessmentItems":[""],"rubric":[{"level":0,"descriptor":""}]}]}`

type DraftRequest struct {
	PromptVersion string
	Requirements  []draftConstraint
}

type Drafter interface {
	ModelName() string
	Draft(ctx context.Context, request DraftRequest) ([]byte, error)
}

type HTTPDrafter struct {
	APIKey   string
	Model    string
	Endpoint string
	Client   *http.Client
}

func (d HTTPDrafter) ModelName() string {
	if name := strings.TrimSpace(d.Model); name != "" {
		return name
	}
	return defaultModelName
}

func (d HTTPDrafter) Draft(ctx context.Context, request DraftRequest) ([]byte, error) {
	payload, err := json.Marshal(request.Requirements)
	if err != nil {
		return nil, ErrInvalid
	}
	return d.complete(ctx, draftPrompt, payload)
}

func (d HTTPDrafter) Match(ctx context.Context, request MatchRequest) ([]byte, error) {
	payload, err := json.Marshal(request.Skills)
	if err != nil {
		return nil, ErrInvalid
	}
	return d.complete(ctx, matchPrompt, payload)
}

func (d HTTPDrafter) complete(ctx context.Context, prompt string, payload []byte) ([]byte, error) {
	key := strings.TrimSpace(d.APIKey)
	if key == "" {
		return nil, ErrProvider
	}
	endpoint := strings.TrimSpace(d.Endpoint)
	if endpoint == "" {
		endpoint = defaultModelURL
	}
	body, err := json.Marshal(map[string]any{
		"model":           d.ModelName(),
		"temperature":     0,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": prompt},
			{"role": "user", "content": string(payload)},
		},
	})
	if err != nil {
		return nil, ErrInvalid
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	httpRequest.Header.Set("Authorization", "Bearer "+key)
	httpRequest.Header.Set("Content-Type", "application/json")
	client := d.Client
	if client == nil {
		client = http.DefaultClient
	}
	response, err := client.Do(httpRequest)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	if response.StatusCode != http.StatusOK {
		return nil, ErrUpstream
	}
	var completion struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &completion); err != nil || len(completion.Choices) == 0 {
		return nil, ErrInvalid
	}
	content := strings.TrimSpace(completion.Choices[0].Message.Content)
	if content == "" || !json.Valid([]byte(content)) {
		return nil, ErrInvalid
	}
	return []byte(content), nil
}
