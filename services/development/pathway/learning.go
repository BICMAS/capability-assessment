package pathway

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"
)

type Resource struct {
	ID             string
	OrganisationID string
	Title          string
	SkillID        string
	TargetLevel    int
	Reference      string
	Rationale      string
}

type ResourceInput struct {
	OrganisationID string
	Title          string
	SkillID        string
	TargetLevel    int
	Reference      string
	Rationale      string
}

type LearningRecommendation struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	SkillID        string `json:"skillId"`
	RequiredLevel  int    `json:"requiredLevel"`
	IntendedLevel  int    `json:"intendedLevel"`
	Activity       string `json:"activity"`
	Rationale      string `json:"rationale"`
	Reference      string `json:"reference"`
	Status         string `json:"status"`
}

type PracticeInput struct {
	OrganisationID string
	PersonID       string
	ScenarioKey    string
	Message        string
	Feedback       string
	Record         bool
}

type PracticeRecord struct {
	ID             string `json:"id,omitempty"`
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	ScenarioKey    string `json:"scenarioKey"`
	Message        string `json:"message"`
	Feedback       string `json:"feedback"`
	Recorded       bool   `json:"recorded"`
	Voice          string `json:"voice"`
}

// RecommendLearning links one confirmed below-requirement gap to a course
// mapping or a local catalogue resource. Missing and met skills are refused.
// The status stays recommended. This service does not receive enrolment,
// start, or completion from a learning system for this record.
func RecommendLearning(ctx context.Context, store Store, organisationID string, signal GapSignal) (LearningRecommendation, error) {
	if organisationID == "" || signal.State != GapBelow || signal.PersonID == "" || signal.CapabilityID == "" || signal.TargetLevel < 0 || signal.TargetLevel > 4 {
		return LearningRecommendation{}, ErrInvalid
	}
	activity := ""
	reference := ""
	rationale := ""
	mapping, err := store.FindCourse(ctx, organisationID, signal.CapabilityID, signal.TargetLevel)
	if err == nil {
		activity = mapping.Title
		reference = mapping.ExternalCourseID
		rationale = "A course mapping exists for this skill. A learning system has not confirmed enrolment."
	} else if errors.Is(err, ErrNotFound) {
		resources, listErr := store.ListResources(ctx, organisationID)
		if listErr != nil {
			return LearningRecommendation{}, listErr
		}
		match, ok := matchResource(resources, signal.CapabilityID, signal.TargetLevel)
		if !ok {
			return LearningRecommendation{}, ErrNotFound
		}
		activity = match.Title
		reference = match.Reference
		rationale = match.Rationale
	} else if err != nil {
		return LearningRecommendation{}, err
	}
	return store.SaveRecommendation(ctx, LearningRecommendation{
		OrganisationID: organisationID,
		PersonID:       signal.PersonID,
		SkillID:        signal.CapabilityID,
		RequiredLevel:  signal.TargetLevel,
		IntendedLevel:  signal.TargetLevel,
		Activity:       activity,
		Rationale:      rationale,
		Reference:      reference,
		Status:         StatusRecommended,
	})
}

func matchResource(resources []Resource, skillID string, targetLevel int) (Resource, bool) {
	found := false
	var best Resource
	for _, resource := range resources {
		if resource.SkillID != skillID || resource.TargetLevel < targetLevel {
			continue
		}
		if !found || resource.TargetLevel < best.TargetLevel {
			best = resource
			found = true
		}
	}
	return best, found
}

func validateResource(input ResourceInput) error {
	if input.OrganisationID == "" || strings.TrimSpace(input.Title) == "" || input.SkillID == "" || strings.TrimSpace(input.Reference) == "" || strings.TrimSpace(input.Rationale) == "" {
		return ErrInvalid
	}
	if input.TargetLevel < 0 || input.TargetLevel > 4 || utf8.RuneCountInString(input.Rationale) > 500 {
		return ErrInvalid
	}
	return nil
}

// Practise runs the text prototype. A transcript is stored only when record
// is set and the caller has already authorised the person.
func Practise(ctx context.Context, store Store, input PracticeInput) (PracticeRecord, error) {
	feedback, err := Reply(input.ScenarioKey, input.Message)
	if err != nil {
		return PracticeRecord{}, err
	}
	record := PracticeRecord{
		OrganisationID: input.OrganisationID,
		PersonID:       input.PersonID,
		ScenarioKey:    input.ScenarioKey,
		Message:        strings.TrimSpace(input.Message),
		Feedback:       feedback,
		Recorded:       false,
		Voice:          VoiceUnconfigured,
	}
	if input.OrganisationID == "" || input.PersonID == "" {
		return PracticeRecord{}, ErrInvalid
	}
	if !input.Record {
		return record, nil
	}
	input.Feedback = feedback
	input.Message = record.Message
	saved, err := store.SavePractice(ctx, input)
	if err != nil {
		return PracticeRecord{}, err
	}
	saved.Voice = VoiceUnconfigured
	saved.Recorded = true
	return saved, nil
}
