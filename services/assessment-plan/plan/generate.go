package plan

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

const (
	PromptVersion        = "assessment-plan-draft-1"
	MethodKnowledgeTest  = "knowledge_test"
	MethodAvatarRoleplay = "ai_avatar_roleplay"
	MethodCaseStudy      = "case_study"
	ReasonKnowledge      = `capability name contains "knowledge"`
	ReasonNegotiation    = `capability name contains "negotiation"`
	ReasonCommunication  = `capability name contains "communication"`
	ReasonDefaultMethod  = "capability name does not contain knowledge, negotiation, or communication"
)

func Generate(ctx context.Context, store Store, frameworks FrameworkReader, drafter Drafter, input PlanInput, authorization, frameworkVersionID, roleID string) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.AssignmentID) {
		return Plan{}, ErrInvalid
	}
	if drafter == nil || !present(drafter.ModelName()) {
		return Plan{}, ErrProvider
	}
	version, err := frameworks.ReadVersion(ctx, authorization, frameworkVersionID)
	if err != nil {
		return Plan{}, err
	}
	if version.Status != "published" {
		return Plan{}, ErrNotPublished
	}
	request, err := draftRequest(roleID, version)
	if err != nil {
		return Plan{}, err
	}
	raw, err := drafter.Draft(ctx, request)
	if err != nil {
		return Plan{}, err
	}
	skills, err := applyModelDraft(roleID, version, raw)
	if err != nil {
		return Plan{}, err
	}
	input.Skills = skills
	input.PromptVersion = PromptVersion
	input.ModelName = drafter.ModelName()
	return store.CreateGeneratedPlan(ctx, input)
}

func chooseMethod(capabilityName string) (string, string) {
	name := strings.ToLower(capabilityName)
	switch {
	case strings.Contains(name, "knowledge"):
		return MethodKnowledgeTest, ReasonKnowledge
	case strings.Contains(name, "negotiation"):
		return MethodAvatarRoleplay, ReasonNegotiation
	case strings.Contains(name, "communication"):
		return MethodAvatarRoleplay, ReasonCommunication
	default:
		return MethodCaseStudy, ReasonDefaultMethod
	}
}

type draftConstraint struct {
	RequirementID     string               `json:"requirementId"`
	CapabilityName    string               `json:"capabilityName"`
	RequiredLevel     int                  `json:"requiredLevel"`
	RecommendedMethod string               `json:"recommendedMethod"`
	ProficiencyLevels []int                `json:"proficiencyLevels"`
	Indicators        []publishedIndicator `json:"indicators"`
}

type publishedIndicator struct {
	Level     int    `json:"level"`
	Statement string `json:"statement"`
}

type modelDraft struct {
	Skills []modelSkill `json:"skills"`
}

type modelSkill struct {
	RequirementID     string      `json:"requirementId"`
	RecommendedMethod string      `json:"recommendedMethod"`
	Instructions      string      `json:"instructions"`
	AssessmentItems   []string    `json:"assessmentItems"`
	Rubric            []modelBand `json:"rubric"`
}

type modelBand struct {
	Level      int    `json:"level"`
	Descriptor string `json:"descriptor"`
}

func draftRequest(roleID string, version FrameworkVersion) (DraftRequest, error) {
	capabilities := map[string]FrameworkCapability{}
	for _, capability := range version.Capabilities {
		capabilities[capability.ID] = capability
	}
	constraints := make([]draftConstraint, 0)
	for _, requirement := range version.Requirements {
		if requirement.RoleID != roleID {
			continue
		}
		capability, ok := capabilities[requirement.CapabilityID]
		if !ok {
			return DraftRequest{}, ErrInvalid
		}
		method, _ := chooseMethod(capability.Name)
		levels := proficiencyLevels(capability)
		indicators := make([]publishedIndicator, 0, len(capability.Indicators))
		for _, indicator := range capability.Indicators {
			statement := strings.TrimSpace(indicator.Statement)
			if statement == "" {
				continue
			}
			indicators = append(indicators, publishedIndicator{Level: indicator.Level, Statement: statement})
		}
		constraints = append(constraints, draftConstraint{
			RequirementID:     requirement.ID,
			CapabilityName:    capability.Name,
			RequiredLevel:     requirement.RequiredLevel,
			RecommendedMethod: method,
			ProficiencyLevels: levels,
			Indicators:        indicators,
		})
	}
	if len(constraints) == 0 {
		return DraftRequest{}, ErrInvalid
	}
	return DraftRequest{PromptVersion: PromptVersion, Requirements: constraints}, nil
}

func applyModelDraft(roleID string, version FrameworkVersion, raw []byte) ([]SkillInput, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var draft modelDraft
	if err := decoder.Decode(&draft); err != nil {
		return nil, ErrInvalid
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return nil, ErrInvalid
	}
	allowed := map[string]FrameworkRequirement{}
	for _, requirement := range version.Requirements {
		if requirement.RoleID == roleID {
			allowed[requirement.ID] = requirement
		}
	}
	capabilities := map[string]FrameworkCapability{}
	for _, capability := range version.Capabilities {
		capabilities[capability.ID] = capability
	}
	if len(draft.Skills) == 0 || len(draft.Skills) != len(allowed) {
		return nil, ErrInvalid
	}
	byRequirement := map[string]modelSkill{}
	for _, skill := range draft.Skills {
		if _, ok := allowed[skill.RequirementID]; !ok {
			return nil, ErrInvalid
		}
		if _, dup := byRequirement[skill.RequirementID]; dup {
			return nil, ErrInvalid
		}
		byRequirement[skill.RequirementID] = skill
	}
	if len(byRequirement) != len(allowed) {
		return nil, ErrInvalid
	}
	skills := make([]SkillInput, 0, len(allowed))
	for _, requirement := range version.Requirements {
		if requirement.RoleID != roleID {
			continue
		}
		skill, ok := byRequirement[requirement.ID]
		if !ok {
			return nil, ErrInvalid
		}
		capability, ok := capabilities[requirement.CapabilityID]
		if !ok || requirement.CapabilityID == "" {
			return nil, ErrInvalid
		}
		method, reason := chooseMethod(capability.Name)
		if skill.RecommendedMethod != method {
			return nil, ErrInvalid
		}
		levels := map[int]struct{}{}
		for _, level := range proficiencyLevels(capability) {
			levels[level] = struct{}{}
		}
		rubric := make([]RubricInput, 0, len(skill.Rubric))
		seenLevel := map[int]struct{}{}
		for _, band := range skill.Rubric {
			if _, published := levels[band.Level]; !published {
				return nil, ErrInvalid
			}
			if _, dup := seenLevel[band.Level]; dup {
				return nil, ErrInvalid
			}
			seenLevel[band.Level] = struct{}{}
			rubric = append(rubric, RubricInput{Level: band.Level, Descriptor: band.Descriptor})
		}
		items := make([]ItemInput, 0, len(skill.AssessmentItems))
		for _, prompt := range skill.AssessmentItems {
			items = append(items, ItemInput{Prompt: prompt})
		}
		needsReview := len(publishedStatements(capability.Indicators)) == 0
		skills = append(skills, SkillInput{
			RequirementID:     requirement.ID,
			CapabilityID:      requirement.CapabilityID,
			RequiredLevel:     requirement.RequiredLevel,
			RecommendedMethod: method,
			Rationale:         reason,
			Instructions:      skill.Instructions,
			RequiresSMEReview: true,
			RubricNeedsReview: needsReview,
			AssessmentItems:   items,
			Rubric:            rubric,
		})
	}
	if err := validateSkills(skills); err != nil {
		return nil, err
	}
	return skills, nil
}

func proficiencyLevels(capability FrameworkCapability) []int {
	seen := map[int]struct{}{}
	levels := make([]int, 0)
	for _, indicator := range capability.Indicators {
		if strings.TrimSpace(indicator.Statement) == "" {
			continue
		}
		if _, ok := seen[indicator.Level]; ok {
			continue
		}
		seen[indicator.Level] = struct{}{}
		levels = append(levels, indicator.Level)
	}
	if len(levels) > 0 {
		return levels
	}
	return []int{0, 1, 2, 3, 4}
}

func publishedStatements(indicators []BehaviouralIndicator) []string {
	statements := make([]string, 0, len(indicators))
	for _, indicator := range indicators {
		statement := strings.TrimSpace(indicator.Statement)
		if statement == "" {
			continue
		}
		statements = append(statements, statement)
	}
	return statements
}
