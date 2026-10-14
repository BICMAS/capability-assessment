package openapi

type AssignPerson struct {
	PersonID        string  `json:"personId"`
	ExceptionReason *string `json:"exceptionReason"`
}

type AssignRequest struct {
	FrameworkVersionID string         `json:"frameworkVersionId"`
	RoleID             string         `json:"roleId"`
	People             []AssignPerson `json:"people"`
}

type AssignmentResult struct {
	PersonID     string  `json:"personId"`
	Outcome      string  `json:"outcome"`
	Reason       string  `json:"reason"`
	AssignmentID *string `json:"assignmentId"`
}

type AssignmentBatch struct {
	Results []AssignmentResult `json:"results"`
}

type Assignment struct {
	ID                 string  `json:"id"`
	OrganisationID     string  `json:"organisationId"`
	PersonID           string  `json:"personId"`
	FrameworkVersionID string  `json:"frameworkVersionId"`
	RoleID             string  `json:"roleId"`
	AssignedByPersonID string  `json:"assignedByPersonId"`
	AssignedAt         string  `json:"assignedAt"`
	Status             string  `json:"status"`
	ExceptionReason    *string `json:"exceptionReason"`
}

type AssessmentItemInput struct {
	Prompt string `json:"prompt"`
}

type RubricBandInput struct {
	Level      int    `json:"level"`
	Descriptor string `json:"descriptor"`
}

type SkillInput struct {
	RequirementID     string                `json:"requirementId"`
	CapabilityID      string                `json:"capabilityId"`
	RequiredLevel     int                   `json:"requiredLevel"`
	RecommendedMethod string                `json:"recommendedMethod"`
	Rationale         string                `json:"rationale"`
	Instructions      string                `json:"instructions"`
	AssessmentItems   []AssessmentItemInput `json:"assessmentItems"`
	Rubric            []RubricBandInput     `json:"rubric"`
}

type GeneratePlan struct {
	AssignmentID string `json:"assignmentId"`
}

type CreatePlan struct {
	AssignmentID string       `json:"assignmentId"`
	Skills       []SkillInput `json:"skills"`
}

type EditSkill struct {
	RecommendedMethod string                `json:"recommendedMethod"`
	Instructions      string                `json:"instructions"`
	AssessmentItems   []AssessmentItemInput `json:"assessmentItems"`
	Rubric            []RubricBandInput     `json:"rubric"`
}

type ChangeRequest struct {
	Reason string `json:"reason"`
}

type ReplacePlan struct {
	Status string       `json:"status"`
	Skills []SkillInput `json:"skills"`
}

type AssessmentItem struct {
	ID     string `json:"id"`
	Prompt string `json:"prompt"`
}

type RubricBand struct {
	Level      int    `json:"level"`
	Descriptor string `json:"descriptor"`
}

type PlanSkill struct {
	ID                string           `json:"id"`
	RequirementID     string           `json:"requirementId"`
	CapabilityID      string           `json:"capabilityId"`
	RequiredLevel     int              `json:"requiredLevel"`
	RecommendedMethod string           `json:"recommendedMethod"`
	Rationale         string           `json:"rationale"`
	Instructions      string           `json:"instructions"`
	RequiresSMEReview bool             `json:"requiresSmeReview"`
	RubricNeedsReview bool             `json:"rubricNeedsReview"`
	EditedByPersonID  *string          `json:"editedByPersonId,omitempty"`
	AssessmentItems   []AssessmentItem `json:"assessmentItems"`
	Rubric            []RubricBand     `json:"rubric"`
}

type Plan struct {
	ID                 string      `json:"id"`
	OrganisationID     string      `json:"organisationId"`
	AssignmentID       string      `json:"assignmentId"`
	VersionNumber      int         `json:"versionNumber"`
	Status             string      `json:"status"`
	ChangeReason       *string     `json:"changeReason,omitempty"`
	ApprovedByPersonID *string     `json:"approvedByPersonId,omitempty"`
	ApprovedAt         *string     `json:"approvedAt,omitempty"`
	Skills             []PlanSkill `json:"skills"`
}
