package plan

import "strings"

func trimAssignment(input AssignmentInput) AssignmentInput {
	input.OrganisationID = strings.TrimSpace(input.OrganisationID)
	input.PersonID = strings.TrimSpace(input.PersonID)
	input.FrameworkVersionID = strings.TrimSpace(input.FrameworkVersionID)
	input.RoleID = strings.TrimSpace(input.RoleID)
	input.AssignedByPersonID = strings.TrimSpace(input.AssignedByPersonID)
	input.ExceptionReason = strings.TrimSpace(input.ExceptionReason)
	return input
}

func trimSkills(skills []SkillInput) []SkillInput {
	trimmed := make([]SkillInput, len(skills))
	for i, skill := range skills {
		skill.RequirementID = strings.TrimSpace(skill.RequirementID)
		skill.CapabilityID = strings.TrimSpace(skill.CapabilityID)
		skill.RecommendedMethod = strings.TrimSpace(skill.RecommendedMethod)
		skill.Rationale = strings.TrimSpace(skill.Rationale)
		skill.Instructions = strings.TrimSpace(skill.Instructions)
		items := make([]ItemInput, len(skill.AssessmentItems))
		for j, item := range skill.AssessmentItems {
			items[j] = ItemInput{Prompt: strings.TrimSpace(item.Prompt)}
		}
		bands := make([]RubricInput, len(skill.Rubric))
		for j, band := range skill.Rubric {
			bands[j] = RubricInput{Level: band.Level, Descriptor: strings.TrimSpace(band.Descriptor)}
		}
		skill.AssessmentItems = items
		skill.Rubric = bands
		trimmed[i] = skill
	}
	return trimmed
}
