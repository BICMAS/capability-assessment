package pathway

import "context"

const EmploymentDecision = "not_an_employment_decision"

const (
	GapMissing = "missing"
	GapBelow   = "below"
	GapMet     = "met"
)

type GapSignal struct {
	PersonID     string
	CapabilityID string
	TargetLevel  int
	State        string
}

type Recommendation struct {
	PersonID           string
	CapabilityID       string
	PathwayID          string
	Status             string
	EmploymentDecision string
}

// RecommendPathways maps assessed gaps that are below the required level onto
// an existing course mapping. Missing and met skills are not recommendations.
// An existing pathway is returned unchanged. Completion stays on that pathway.
func RecommendPathways(ctx context.Context, store Store, organisationID string, signals []GapSignal) ([]Recommendation, error) {
	if organisationID == "" {
		return nil, ErrInvalid
	}
	recommendations := make([]Recommendation, 0)
	for _, signal := range signals {
		if signal.State != GapBelow || signal.PersonID == "" || signal.CapabilityID == "" || signal.TargetLevel < 0 || signal.TargetLevel > 4 {
			continue
		}
		pathway, err := store.CreatePathway(ctx, PathwayInput{
			OrganisationID: organisationID,
			PersonID:       signal.PersonID,
			CapabilityID:   signal.CapabilityID,
			TargetLevel:    signal.TargetLevel,
		})
		if err == ErrNotFound {
			continue
		}
		if err != nil {
			return nil, err
		}
		status := ""
		if len(pathway.Items) > 0 {
			status = pathway.Items[0].Status
		}
		recommendations = append(recommendations, Recommendation{
			PersonID:           signal.PersonID,
			CapabilityID:       signal.CapabilityID,
			PathwayID:          pathway.ID,
			Status:             status,
			EmploymentDecision: EmploymentDecision,
		})
	}
	return recommendations, nil
}
