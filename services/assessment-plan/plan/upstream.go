package plan

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type HTTPFramework struct {
	BaseURL string
	Client  *http.Client
}

func (h HTTPFramework) ReadVersion(ctx context.Context, authorization, versionID string) (FrameworkVersion, error) {
	endpoint, err := versionURL(h.BaseURL, versionID)
	if err != nil {
		return FrameworkVersion{}, err
	}
	body, status, err := getJSON(ctx, h.Client, endpoint, authorization)
	if err != nil {
		return FrameworkVersion{}, err
	}
	if status == http.StatusNotFound {
		return FrameworkVersion{}, ErrNotFound
	}
	if status != http.StatusOK {
		return FrameworkVersion{}, ErrUpstream
	}
	var payload struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Roles  []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"roles"`
		Capabilities []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Levels []struct {
				Level      int `json:"level"`
				Indicators []struct {
					Statement string `json:"statement"`
				} `json:"indicators"`
			} `json:"levels"`
		} `json:"capabilities"`
		Requirements []struct {
			ID            string `json:"id"`
			RoleID        string `json:"roleId"`
			CapabilityID  string `json:"capabilityId"`
			RequiredLevel int    `json:"requiredLevel"`
		} `json:"requirements"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return FrameworkVersion{}, ErrUpstream
	}
	roles := make([]FrameworkRole, 0, len(payload.Roles))
	for _, role := range payload.Roles {
		roles = append(roles, FrameworkRole{ID: role.ID, Name: role.Name})
	}
	capabilities := make([]FrameworkCapability, 0, len(payload.Capabilities))
	for _, capability := range payload.Capabilities {
		indicators := make([]BehaviouralIndicator, 0)
		for _, level := range capability.Levels {
			for _, indicator := range level.Indicators {
				indicators = append(indicators, BehaviouralIndicator{Level: level.Level, Statement: indicator.Statement})
			}
		}
		capabilities = append(capabilities, FrameworkCapability{ID: capability.ID, Name: capability.Name, Indicators: indicators})
	}
	requirements := make([]FrameworkRequirement, 0, len(payload.Requirements))
	for _, requirement := range payload.Requirements {
		requirements = append(requirements, FrameworkRequirement{
			ID:            requirement.ID,
			RoleID:        requirement.RoleID,
			CapabilityID:  requirement.CapabilityID,
			RequiredLevel: requirement.RequiredLevel,
		})
	}
	return FrameworkVersion{
		ID:           payload.ID,
		Status:       payload.Status,
		Roles:        roles,
		Capabilities: capabilities,
		Requirements: requirements,
	}, nil
}

type HTTPDirectory struct {
	BaseURL string
	Client  *http.Client
}

func (h HTTPDirectory) ReadPerson(ctx context.Context, authorization, personID string) (PersonProfile, error) {
	endpoint, err := personURL(h.BaseURL, personID)
	if err != nil {
		return PersonProfile{}, err
	}
	body, status, err := getJSON(ctx, h.Client, endpoint, authorization)
	if err != nil {
		return PersonProfile{}, err
	}
	if status == http.StatusNotFound {
		return PersonProfile{}, ErrPersonOutsideOrg
	}
	if status != http.StatusOK {
		return PersonProfile{}, ErrUpstream
	}
	var payload struct {
		ID    string `json:"id"`
		Roles []struct {
			Title string `json:"title"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return PersonProfile{}, ErrUpstream
	}
	titles := make([]string, 0, len(payload.Roles))
	for _, role := range payload.Roles {
		titles = append(titles, role.Title)
	}
	return PersonProfile{ID: payload.ID, RoleTitles: titles}, nil
}

func getJSON(ctx context.Context, client *http.Client, endpoint, authorization string) ([]byte, int, error) {
	if client == nil {
		client = http.DefaultClient
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	request.Header.Set("Authorization", authorization)
	response, err := client.Do(request)
	if err != nil {
		return nil, 0, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, response.StatusCode, fmt.Errorf("%w: %s", ErrUpstream, err.Error())
	}
	return body, response.StatusCode, nil
}

func versionURL(baseURL, versionID string) (string, error) {
	return joinURL(baseURL, "/framework-versions/"+url.PathEscape(versionID))
}

func personURL(baseURL, personID string) (string, error) {
	return joinURL(baseURL, "/people/"+url.PathEscape(personID))
}

func joinURL(baseURL, path string) (string, error) {
	trimmed := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if trimmed == "" {
		return "", ErrUpstream
	}
	return trimmed + path, nil
}
