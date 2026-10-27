package plan

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ProfileReader interface {
	ReadProfileVersion(ctx context.Context, authorization, profileID, versionID string) (ProfileSource, error)
}

type TenureReader interface {
	ListRoleTenures(ctx context.Context, authorization string) ([]RoleTenure, error)
}

type LevelView struct {
	Found        bool
	CurrentLevel string
	Gap          string
}

type LevelReader interface {
	ReadLevel(ctx context.Context, authorization, personID, versionID, skillID string) (LevelView, error)
}

type DirectoryPerson struct {
	ID             string
	DepartmentName string
}

type PeopleReader interface {
	ListPeople(ctx context.Context, authorization string) ([]DirectoryPerson, error)
}

type HTTPProfile struct {
	BaseURL string
	Client  *http.Client
}

func (h HTTPProfile) ReadProfileVersion(ctx context.Context, authorization, profileID, versionID string) (ProfileSource, error) {
	profileURL, err := joinURL(h.BaseURL, "/role-profiles/"+url.PathEscape(profileID))
	if err != nil {
		return ProfileSource{}, err
	}
	body, status, err := getJSON(ctx, h.Client, profileURL, authorization)
	if err != nil {
		return ProfileSource{}, err
	}
	if status == http.StatusNotFound {
		return ProfileSource{}, ErrNotFound
	}
	if status != http.StatusOK {
		return ProfileSource{}, ErrUpstream
	}
	var profile struct {
		ID             string `json:"id"`
		OrganisationID string `json:"organisationId"`
		RoleID         string `json:"roleId"`
		RoleName       string `json:"roleName"`
	}
	if err := json.Unmarshal(body, &profile); err != nil {
		return ProfileSource{}, ErrUpstream
	}
	versionsURL, err := joinURL(h.BaseURL, "/role-profiles/"+url.PathEscape(profileID)+"/versions")
	if err != nil {
		return ProfileSource{}, err
	}
	body, status, err = getJSON(ctx, h.Client, versionsURL, authorization)
	if err != nil {
		return ProfileSource{}, err
	}
	if status != http.StatusOK {
		return ProfileSource{}, ErrUpstream
	}
	var versions []struct {
		ID            string `json:"id"`
		VersionNumber int    `json:"versionNumber"`
		Status        string `json:"status"`
		Skills        []struct {
			ID                  string `json:"id"`
			SkillID             string `json:"skillId"`
			SkillName           string `json:"skillName"`
			Definition          string `json:"definition"`
			RequiredProficiency int    `json:"requiredProficiency"`
			Priority            string `json:"priority"`
			Rationale           string `json:"rationale"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(body, &versions); err != nil {
		return ProfileSource{}, ErrUpstream
	}
	for _, version := range versions {
		if version.ID != versionID {
			continue
		}
		source := ProfileSource{
			ProfileID: profile.ID, VersionID: version.ID, VersionNumber: version.VersionNumber,
			OrganisationID: profile.OrganisationID, RoleID: profile.RoleID, RoleName: profile.RoleName, Status: version.Status,
		}
		for _, skill := range version.Skills {
			source.Skills = append(source.Skills, ProfileSkillSnapshot{
				AssignmentID: skill.ID, SkillID: skill.SkillID, Name: skill.SkillName, Definition: skill.Definition,
				RequiredProficiency: skill.RequiredProficiency, Priority: skill.Priority, Rationale: skill.Rationale,
			})
		}
		return source, nil
	}
	return ProfileSource{}, ErrNotFound
}

type HTTPLevels struct {
	BaseURL string
	Client  *http.Client
}

func (h HTTPLevels) ReadLevel(ctx context.Context, authorization, personID, versionID, skillID string) (LevelView, error) {
	if strings.TrimSpace(h.BaseURL) == "" {
		return LevelView{}, ErrProvider
	}
	endpoint, err := joinURL(h.BaseURL, "/levels")
	if err != nil {
		return LevelView{}, err
	}
	endpoint += "?personId=" + url.QueryEscape(personID) + "&frameworkVersionId=" + url.QueryEscape(versionID) + "&capabilityCode=" + url.QueryEscape(skillID)
	body, status, err := getJSON(ctx, h.Client, endpoint, authorization)
	if err != nil {
		return LevelView{}, err
	}
	if status == http.StatusNotFound {
		return LevelView{}, nil
	}
	if status != http.StatusOK {
		return LevelView{}, ErrUpstream
	}
	var payload struct {
		CurrentLevel string `json:"currentLevel"`
		Gap          string `json:"gap"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return LevelView{}, ErrUpstream
	}
	return LevelView{Found: true, CurrentLevel: payload.CurrentLevel, Gap: payload.Gap}, nil
}

func (h HTTPDirectory) ListPeople(ctx context.Context, authorization string) ([]DirectoryPerson, error) {
	endpoint, err := joinURL(h.BaseURL, "/people")
	if err != nil {
		return nil, err
	}
	body, status, err := getJSON(ctx, h.Client, endpoint, authorization)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, ErrUpstream
	}
	var rows []struct {
		ID             string `json:"id"`
		DepartmentName string `json:"departmentName"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, ErrUpstream
	}
	people := make([]DirectoryPerson, 0, len(rows))
	for _, row := range rows {
		people = append(people, DirectoryPerson{ID: row.ID, DepartmentName: row.DepartmentName})
	}
	return people, nil
}

func (h HTTPDirectory) ListRoleTenures(ctx context.Context, authorization string) ([]RoleTenure, error) {
	endpoint, err := joinURL(h.BaseURL, "/role-assignments")
	if err != nil {
		return nil, err
	}
	body, status, err := getJSON(ctx, h.Client, endpoint, authorization)
	if err != nil {
		return nil, err
	}
	if status != http.StatusOK {
		return nil, ErrUpstream
	}
	var rows []struct {
		PersonID  string    `json:"personId"`
		Title     string    `json:"title"`
		CreatedAt time.Time `json:"createdAt"`
	}
	if err := json.Unmarshal(body, &rows); err != nil {
		return nil, ErrUpstream
	}
	tenures := make([]RoleTenure, 0, len(rows))
	for _, row := range rows {
		tenures = append(tenures, RoleTenure{PersonID: row.PersonID, Title: row.Title, StartedAt: row.CreatedAt})
	}
	return tenures, nil
}

func CreateProfilePlan(store ProfilePlanStore, profiles ProfileReader, tenures TenureReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if !claims.HasRole("admin") {
			writeError(w, http.StatusForbidden)
			return
		}
		if profiles == nil || tenures == nil {
			writeStoreError(w, ErrProvider)
			return
		}
		var body struct {
			ProfileID        string    `json:"profileId"`
			ProfileVersionID string    `json:"profileVersionId"`
			PersonID         string    `json:"personId"`
			AssignedAt       time.Time `json:"assignedAt"`
		}
		if !decode(w, r, &body) {
			return
		}
		profile, err := profiles.ReadProfileVersion(r.Context(), r.Header.Get("Authorization"), body.ProfileID, body.ProfileVersionID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		roleTenures, err := tenures.ListRoleTenures(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		when := body.AssignedAt
		if when.IsZero() {
			when = time.Now().UTC()
		}
		created, baseline, err := CreateFromProfile(r.Context(), store, ProfilePlanInput{
			OrganisationID: claims.OrganisationID,
			PersonID:       body.PersonID,
			ActorPersonID:  claims.PersonID,
			AssignedAt:     when,
			Tenures:        roleTenures,
			Profile:        profile,
		})
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{
			"planId":             created.ID,
			"status":             created.Status,
			"profileVersionId":   baseline.ProfileVersionID,
			"employmentDecision": EmploymentDecision,
		})
	}
}

func ProfileInsights(store ProfilePlanStore, levels LevelReader, tenures TenureReader, people PeopleReader) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := claimsFrom(w, r)
		if !ok {
			return
		}
		if levels == nil || tenures == nil {
			writeStoreError(w, ErrProvider)
			return
		}
		versionID := r.URL.Query().Get("profileVersionId")
		department := r.URL.Query().Get("department")
		if department != "" && people == nil {
			writeStoreError(w, ErrProvider)
			return
		}
		baselines, err := store.ListProfileBaselines(r.Context(), claims.OrganisationID)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		roleTenures, err := tenures.ListRoleTenures(r.Context(), r.Header.Get("Authorization"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		departments := map[string]string{}
		if department != "" {
			rows, err := people.ListPeople(r.Context(), r.Header.Get("Authorization"))
			if err != nil {
				writeStoreError(w, err)
				return
			}
			for _, row := range rows {
				departments[row.ID] = row.DepartmentName
			}
		}
		subjects := make([]InsightSubject, 0, len(baselines))
		for _, baseline := range baselines {
			if versionID != "" && baseline.ProfileVersionID != versionID {
				continue
			}
			skills := make([]InsightSkill, 0)
			for _, skill := range baseline.Skills {
				if skill.Priority != "required" {
					continue
				}
				view, err := levels.ReadLevel(r.Context(), r.Header.Get("Authorization"), baseline.PersonID, baseline.ProfileVersionID, skill.SkillID)
				if err != nil {
					writeStoreError(w, err)
					return
				}
				skills = append(skills, InsightSkill{
					SkillID: skill.SkillID, RequiredProficiency: skill.RequiredProficiency,
					Found: view.Found, CurrentLevel: view.CurrentLevel, Gap: view.Gap,
				})
			}
			subjects = append(subjects, InsightSubject{
				PersonID: baseline.PersonID, Department: departments[baseline.PersonID],
				ProfileVersionID: baseline.ProfileVersionID, RoleName: baseline.RoleName,
				AssessedAt: baselineAssessedAt(roleTenures, baseline.PersonID, baseline.RoleName),
				Tenures:    roleTenures, Skills: skills,
			})
		}
		insight, err := BuildInsight(InsightActor{PersonID: claims.PersonID, Roles: claims.Roles, ManagerOf: claims.ManagerOf}, InsightRequest{
			PersonID: r.URL.Query().Get("personId"), Department: department, ProfileVersionID: versionID,
		}, subjects)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, insight)
	}
}

func baselineAssessedAt(tenures []RoleTenure, personID, roleName string) time.Time {
	var started time.Time
	for _, tenure := range tenures {
		if tenure.PersonID == personID && strings.TrimSpace(tenure.Title) == strings.TrimSpace(roleName) && (started.IsZero() || tenure.StartedAt.Before(started)) {
			started = tenure.StartedAt
		}
	}
	if started.IsZero() {
		return time.Now().UTC()
	}
	return started
}
