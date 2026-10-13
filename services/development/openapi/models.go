package openapi

type CourseMappingInput struct {
	ExternalCourseID string `json:"externalCourseId"`
	CapabilityID     string `json:"capabilityId"`
	TargetLevel      int    `json:"targetLevel"`
	Title            string `json:"title"`
}

type CourseMapping struct {
	ID               string `json:"id"`
	OrganisationID   string `json:"organisationId"`
	ExternalCourseID string `json:"externalCourseId"`
	CapabilityID     string `json:"capabilityId"`
	TargetLevel      int    `json:"targetLevel"`
	Title            string `json:"title"`
}

type CreatePathway struct {
	PersonID     string `json:"personId"`
	CapabilityID string `json:"capabilityId"`
	TargetLevel  int    `json:"targetLevel"`
}

type PlanItem struct {
	ID       string  `json:"id"`
	Stage    string  `json:"stage"`
	Title    string  `json:"title"`
	Status   string  `json:"status"`
	DeepLink *string `json:"deepLink"`
}

type Pathway struct {
	ID             string     `json:"id"`
	OrganisationID string     `json:"organisationId"`
	PersonID       string     `json:"personId"`
	CapabilityID   string     `json:"capabilityId"`
	TargetLevel    int        `json:"targetLevel"`
	Items          []PlanItem `json:"items"`
}

type CompletionWebhook struct {
	OrganisationID string `json:"organisationId"`
	PersonID       string `json:"personId"`
	PlanItemID     string `json:"planItemId"`
	CompletedAt    string `json:"completedAt"`
}
