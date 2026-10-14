package openapi

// RecordAttempt is a stored attempt. Simulation has no writer.
// Self-assessment is stored and is not part of the level.
type RecordAttempt struct {
	PersonID           string   `json:"personId"`
	FrameworkVersionID string   `json:"frameworkVersionId"`
	CapabilityCode     string   `json:"capabilityCode"`
	Source             string   `json:"source"`
	Percentage         int      `json:"percentage"`
	EvidenceIDs        []string `json:"evidenceIds"`
	RequiredLevel      float64  `json:"requiredLevel"`
}

// Explanation is the persisted level calculation.
type Explanation struct {
	ID                             string `json:"id"`
	OrganisationID                 string `json:"organisationId"`
	PersonID                       string `json:"personId"`
	FrameworkVersionID             string `json:"frameworkVersionId"`
	CapabilityCode                 string `json:"capabilityCode"`
	KnowledgePercentage            *int   `json:"knowledgePercentage"`
	SituationalJudgementPercentage *int   `json:"situationalJudgementPercentage"`
	AIInterviewPercentage          *int   `json:"aiInterviewPercentage"`
	SimulationPercentage           *int   `json:"simulationPercentage"`
	ManagerAssessmentPercentage    *int   `json:"managerAssessmentPercentage"`
	SelfAssessmentPercentage       *int   `json:"selfAssessmentPercentage"`
	Mean                           string `json:"mean"`
	Divisor                        int    `json:"divisor"`
	RequiredLevel                  string `json:"requiredLevel"`
	CurrentLevel                   string `json:"currentLevel"`
	Gap                            string `json:"gap"`
	Status                         string `json:"status"`
}

// CreateOverride records a manager's replacement level.
type CreateOverride struct {
	Reason   string  `json:"reason"`
	NewValue float64 `json:"newValue"`
}

// Override is the stored manager override.
type Override struct {
	ID             string `json:"id"`
	OrganisationID string `json:"organisationId"`
	ExplanationID  string `json:"explanationId"`
	ActorPersonID  string `json:"actorPersonId"`
	Reason         string `json:"reason"`
	PreviousValue  string `json:"previousValue"`
	NewValue       string `json:"newValue"`
}
