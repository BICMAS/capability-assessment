package plan

import (
	"strings"
	"unicode/utf8"
)

func validateAssignment(input AssignmentInput) error {
	if !present(input.OrganisationID) || !present(input.PersonID) || !present(input.FrameworkVersionID) || !present(input.RoleID) || !present(input.AssignedByPersonID) {
		return ErrInvalid
	}
	if input.AssignedAt.IsZero() {
		return ErrInvalid
	}
	if input.ExceptionReason != "" && !shortText(input.ExceptionReason, 280) {
		return ErrInvalid
	}
	return nil
}

func validateSkills(skills []SkillInput) error {
	if len(skills) == 0 {
		return ErrInvalid
	}
	requirements := map[string]struct{}{}
	capabilities := map[string]struct{}{}
	for _, skill := range skills {
		if !present(skill.RequirementID) || !present(skill.CapabilityID) {
			return ErrInvalid
		}
		if _, ok := requirements[skill.RequirementID]; ok {
			return ErrInvalid
		}
		if _, ok := capabilities[skill.CapabilityID]; ok {
			return ErrInvalid
		}
		requirements[skill.RequirementID] = struct{}{}
		capabilities[skill.CapabilityID] = struct{}{}
		if skill.RequiredLevel < 0 || skill.RequiredLevel > 4 {
			return ErrInvalid
		}
		if !bounded(skill.RecommendedMethod, 80) || !shortText(skill.Rationale, 280) || !bounded(skill.Instructions, 8000) {
			return ErrInvalid
		}
		if len(skill.AssessmentItems) == 0 || len(skill.Rubric) == 0 {
			return ErrInvalid
		}
		levels := map[int]struct{}{}
		for _, item := range skill.AssessmentItems {
			if !bounded(item.Prompt, 2000) {
				return ErrInvalid
			}
		}
		for _, band := range skill.Rubric {
			if band.Level < 0 || band.Level > 4 || !bounded(band.Descriptor, 2000) {
				return ErrInvalid
			}
			if _, ok := levels[band.Level]; ok {
				return ErrInvalid
			}
			levels[band.Level] = struct{}{}
		}
	}
	return nil
}

func validateSkillEdit(input EditSkillInput) error {
	if !present(input.OrganisationID) || !present(input.PlanID) || !present(input.SkillID) || !present(input.EditorPersonID) {
		return ErrInvalid
	}
	if !bounded(input.RecommendedMethod, 80) || !bounded(input.Instructions, 8000) {
		return ErrInvalid
	}
	if len(input.AssessmentItems) == 0 || len(input.Rubric) == 0 {
		return ErrInvalid
	}
	levels := map[int]struct{}{}
	for _, item := range input.AssessmentItems {
		if !bounded(item.Prompt, 2000) {
			return ErrInvalid
		}
	}
	for _, band := range input.Rubric {
		if band.Level < 0 || band.Level > 4 || !bounded(band.Descriptor, 2000) {
			return ErrInvalid
		}
		if _, ok := levels[band.Level]; ok {
			return ErrInvalid
		}
		levels[band.Level] = struct{}{}
	}
	return nil
}

func ensureSkillEditable(status string) error {
	if status == StatusApproved {
		return ErrImmutable
	}
	if status != StatusGenerated && status != StatusChangesRequested {
		return ErrNotEditable
	}
	return nil
}

func ensureSubmittable(status string) error {
	if status == StatusApproved {
		return ErrImmutable
	}
	if status != StatusGenerated && status != StatusChangesRequested {
		return ErrNotEditable
	}
	return nil
}

func ensureReviewable(status string) error {
	if status == StatusApproved {
		return ErrImmutable
	}
	if status != StatusUnderReview {
		return ErrNotEditable
	}
	return nil
}

func ensureApprovable(status string) error {
	if status == StatusApproved {
		return ErrImmutable
	}
	if status != StatusGenerated && status != StatusUnderReview && status != StatusChangesRequested {
		return ErrNotEditable
	}
	return nil
}

func validStatus(status string) bool {
	switch status {
	case StatusDraft, StatusGenerated, StatusUnderReview, StatusChangesRequested, StatusApproved:
		return true
	default:
		return false
	}
}

func present(value string) bool {
	return strings.TrimSpace(value) != ""
}

func bounded(value string, max int) bool {
	count := utf8.RuneCountInString(strings.TrimSpace(value))
	return count > 0 && count <= max
}

func shortText(value string, max int) bool {
	return bounded(value, max)
}
