package plan

import (
	"context"
	"fmt"
	"strings"
)

const (
	MethodKnowledgeTest  = "knowledge_test"
	MethodAvatarRoleplay = "ai_avatar_roleplay"
	MethodCaseStudy      = "case_study"
	ReasonKnowledge      = `capability name contains "knowledge"`
	ReasonNegotiation    = `capability name contains "negotiation"`
	ReasonCommunication  = `capability name contains "communication"`
	ReasonDefaultMethod  = "capability name does not contain knowledge, negotiation, or communication"
)

func Generate(ctx context.Context, store Store, frameworks FrameworkReader, input PlanInput, authorization, frameworkVersionID, roleID string) (Plan, error) {
	if !present(input.OrganisationID) || !present(input.AssignmentID) {
		return Plan{}, ErrInvalid
	}
	version, err := frameworks.ReadVersion(ctx, authorization, frameworkVersionID)
	if err != nil {
		return Plan{}, err
	}
	if version.Status != "published" {
		return Plan{}, ErrNotPublished
	}
	draft, err := buildGeneratedDraft(roleID, version)
	if err != nil {
		return Plan{}, err
	}
	if err := validateGeneratedDraft(roleID, version, draft); err != nil {
		return Plan{}, err
	}
	input.Skills = draft
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

func buildGeneratedDraft(roleID string, version FrameworkVersion) ([]SkillInput, error) {
	capabilities := map[string]FrameworkCapability{}
	for _, capability := range version.Capabilities {
		capabilities[capability.ID] = capability
	}
	skills := make([]SkillInput, 0)
	for _, requirement := range version.Requirements {
		if requirement.RoleID != roleID {
			continue
		}
		capability, ok := capabilities[requirement.CapabilityID]
		if !ok {
			return nil, ErrInvalid
		}
		method, reason := chooseMethod(capability.Name)
		indicators := capability.Indicators
		rubric := make([]RubricInput, 0, len(indicators))
		statements := make([]string, 0, len(indicators))
		for _, indicator := range indicators {
			statement := strings.TrimSpace(indicator.Statement)
			if statement == "" {
				continue
			}
			statements = append(statements, statement)
			rubric = append(rubric, RubricInput{Level: indicator.Level, Descriptor: statement})
		}
		needsReview := len(statements) == 0
		skills = append(skills, SkillInput{
			RequirementID:     requirement.ID,
			CapabilityID:      requirement.CapabilityID,
			RequiredLevel:     requirement.RequiredLevel,
			RecommendedMethod: method,
			Rationale:         reason,
			Instructions:      instructionsFor(capability.Name, requirement.RequiredLevel, needsReview),
			RequiresSMEReview: true,
			RubricNeedsReview: needsReview,
			AssessmentItems: []ItemInput{{
				Prompt: assessmentPrompt(capability.Name, requirement.RequiredLevel),
			}},
			Rubric: rubric,
		})
	}
	if len(skills) == 0 {
		return nil, ErrInvalid
	}
	return skills, nil
}

func validateGeneratedDraft(roleID string, version FrameworkVersion, skills []SkillInput) error {
	allowed := map[string]FrameworkRequirement{}
	for _, requirement := range version.Requirements {
		if requirement.RoleID == roleID {
			allowed[requirement.ID] = requirement
		}
	}
	if len(skills) == 0 || len(skills) != len(allowed) {
		return ErrInvalid
	}
	capabilities := map[string]FrameworkCapability{}
	for _, capability := range version.Capabilities {
		capabilities[capability.ID] = capability
	}
	seen := map[string]struct{}{}
	for _, skill := range skills {
		requirement, ok := allowed[skill.RequirementID]
		if !ok || requirement.RoleID != roleID || requirement.CapabilityID != skill.CapabilityID || requirement.RequiredLevel != skill.RequiredLevel {
			return ErrInvalid
		}
		if _, dup := seen[skill.RequirementID]; dup {
			return ErrInvalid
		}
		seen[skill.RequirementID] = struct{}{}
		capability, ok := capabilities[skill.CapabilityID]
		if !ok {
			return ErrInvalid
		}
		method, reason := chooseMethod(capability.Name)
		if skill.RecommendedMethod != method || skill.Rationale != reason || !skill.RequiresSMEReview {
			return ErrInvalid
		}
		if !present(skill.Instructions) || len(skill.AssessmentItems) != 1 || !present(skill.AssessmentItems[0].Prompt) {
			return ErrInvalid
		}
		statements := publishedStatements(capability.Indicators)
		if len(statements) == 0 {
			if !skill.RubricNeedsReview || len(skill.Rubric) != 0 {
				return ErrInvalid
			}
			continue
		}
		if skill.RubricNeedsReview || len(skill.Rubric) != len(statements) {
			return ErrInvalid
		}
		for i, statement := range statements {
			if skill.Rubric[i].Descriptor != statement {
				return ErrInvalid
			}
		}
	}
	return nil
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

func instructionsFor(capability string, level int, needsReview bool) string {
	text := fmt.Sprintf("Assess %s at required level %d.", capability, level)
	if needsReview {
		return text + " The rubric needs review because the published requirement has no behavioural indicators."
	}
	return text + " Use the published behavioural indicators as the rubric anchors."
}

func assessmentPrompt(capability string, level int) string {
	return fmt.Sprintf("Assess %s at required level %d.", capability, level)
}
