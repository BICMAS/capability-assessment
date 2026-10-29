package roleprofile

import (
	"context"
	"encoding/json"
	"strings"
)

const MockModelID = "mock-role-skills"

type MockProvider struct{}

func (MockProvider) ModelID() string { return MockModelID }

func (MockProvider) Clarify(context.Context, ContextSnapshot, []string) ([]QuestionDraft, error) {
	return nil, nil
}

func (MockProvider) Generate(_ context.Context, input GenerationInput) (ProviderResult, error) {
	role := strings.TrimSpace(input.RoleName)
	if role == "" {
		role = "this role"
	}
	purpose := strings.TrimSpace(input.Snapshot.Purpose)
	definition := "Development mock suggestion for " + role + "."
	if purpose != "" {
		definition = "Development mock suggestion for " + role + ": " + purpose
	}
	skills := make([]GeneratedSkill, 0, 3)
	used := map[string]struct{}{}
	for _, kind := range []string{KindTechnical, KindBehavioural, KindKnowledge} {
		for _, skill := range input.Taxonomy {
			if _, seen := used[skill.ID]; seen || !kindAgrees(kind, skill.Category) {
				continue
			}
			used[skill.ID] = struct{}{}
			skills = append(skills, GeneratedSkill{
				SkillID:             skill.ID,
				Kind:                kind,
				Definition:          definition,
				RequiredProficiency: 2,
				Priority:            PriorityPreferred,
				Rationale:           "Development mock suggestion. This is not model output and is not an approved requirement.",
				Criteria:            "Development mock suggestion. Review this note before anyone is assessed against it.",
			})
			break
		}
	}
	if len(skills) == 0 {
		return ProviderResult{}, ErrInvalid
	}
	body, err := json.Marshal(generatedOutput{Skills: skills, Unsupported: []UnsupportedSuggestion{}})
	if err != nil {
		return ProviderResult{}, ErrInvalid
	}
	return ProviderResult{Body: body}, nil
}
