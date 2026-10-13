package levels

func fold(attempts []Attempt, columns Sources, required Tenths) (Sources, Tenths) {
	sources := Sources{
		AIInterview: copyInt(columns.AIInterview),
		Simulation:  copyInt(columns.Simulation),
	}
	for _, attempt := range attempts {
		switch attempt.Source {
		case SourceKnowledge:
			sources.Knowledge = copyInt(&attempt.Percentage)
		case SourceSituationalJudgement:
			sources.SituationalJudgement = copyInt(&attempt.Percentage)
		case SourceSelfAssessment:
			sources.SelfAssessment = copyInt(&attempt.Percentage)
		case SourceManagerAssessment:
			sources.ManagerAssessment = copyInt(&attempt.Percentage)
		case SourceAIInterview:
			sources.AIInterview = copyInt(&attempt.Percentage)
		case SourceSimulation:
			sources.Simulation = copyInt(&attempt.Percentage)
		}
		required = attempt.RequiredLevel
	}
	return sources, required
}

func validatePercentage(value int) error {
	if value < 0 || value > 100 {
		return ErrInvalid
	}
	return nil
}

func validateRequired(value Tenths) error {
	if value < 0 || value > 40 {
		return ErrInvalid
	}
	return nil
}

func validateIdentity(organisationID, personID, frameworkVersionID, capabilityCode string) error {
	if organisationID == "" || personID == "" || frameworkVersionID == "" || capabilityCode == "" {
		return ErrInvalid
	}
	return nil
}

func evidenceIDs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrInvalid
	}
	copied := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			return nil, ErrInvalid
		}
		copied = append(copied, value)
	}
	return copied, nil
}
