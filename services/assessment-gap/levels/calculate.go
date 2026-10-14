package levels

// Divisor converts a mean percentage into a level from 0.0 to 4.0.
// 0% is 0.0 and 100% is 4.0.
const Divisor = 25

const (
	StatusProvisional = "provisional"
	StatusVerified    = "verified"
)

const (
	SourceKnowledge            = "knowledge"
	SourceSituationalJudgement = "situational_judgement"
	SourceSelfAssessment       = "self_assessment"
	SourceAIInterview          = "ai_interview"
	SourceSimulation           = "simulation"
	SourceManagerAssessment    = "manager_assessment"
)

// Sources holds the latest percentage for each assessment. A nil source is
// omitted. Self-assessment is stored and is not part of the level.
type Sources struct {
	Knowledge            *int
	SituationalJudgement *int
	AIInterview          *int
	Simulation           *int
	ManagerAssessment    *int
	SelfAssessment       *int
}

// Explain calculates the current level in this service.
// The mean uses knowledge, situational judgement, AI interview, simulation,
// and manager assessment. Missing sources are omitted. The level is that
// mean divided by 25. The result stays provisional.
func Explain(sources Sources, frameworkVersionID string, required Tenths) Explanation {
	percentages := scoringPercentages(sources)
	var mean Tenths
	var current Tenths
	if len(percentages) > 0 {
		sum := 0
		for _, percentage := range percentages {
			sum += percentage
		}
		mean = Tenths(sum * 10 / len(percentages))
		current = mean / Divisor
	}
	return Explanation{
		FrameworkVersionID:   frameworkVersionID,
		Knowledge:            copyInt(sources.Knowledge),
		SituationalJudgement: copyInt(sources.SituationalJudgement),
		AIInterview:          copyInt(sources.AIInterview),
		Simulation:           copyInt(sources.Simulation),
		ManagerAssessment:    copyInt(sources.ManagerAssessment),
		SelfAssessment:       copyInt(sources.SelfAssessment),
		Mean:                 mean,
		Divisor:              Divisor,
		RequiredLevel:        required,
		CurrentLevel:         current,
		Gap:                  required - current,
		Status:               StatusProvisional,
	}
}

func scoringPercentages(sources Sources) []int {
	values := make([]int, 0, 5)
	for _, value := range []*int{
		sources.Knowledge,
		sources.SituationalJudgement,
		sources.AIInterview,
		sources.Simulation,
		sources.ManagerAssessment,
	} {
		if value != nil {
			values = append(values, *value)
		}
	}
	return values
}

func copyInt(value *int) *int {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func writable(source string) bool {
	switch source {
	case SourceKnowledge, SourceSituationalJudgement, SourceSelfAssessment, SourceManagerAssessment, SourceAIInterview:
		return true
	default:
		return false
	}
}

func columnOnly(source string) bool {
	return source == SourceSimulation
}
