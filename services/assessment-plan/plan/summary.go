package plan

// GapMethod is the only proficiency comparison used for skills-gap analysis.
// Proficiency labels are the integers 0–4. assessment-gap stores a level in
// tenths: the mean of the scored sources (knowledge, situational judgement,
// AI interview, simulation, and manager assessment) is divided by 25.
// The gap is required tenths minus current tenths. A positive gap is below
// the requirement. Zero or a negative gap is met. Self-assessment is stored
// and is not a scored source. A skill with no scored level is missing.
// Missing is not proficiency zero and is not a confirmed gap.
const GapMethod = "Proficiency uses integers 0 to 4. A scored level is the mean of the scored assessment sources divided by 25, kept in tenths. The gap is required tenths minus current tenths. A positive gap is below. Zero or a negative gap is met. No scored level is missing, which is not proficiency zero and is not a confirmed gap. Self-assessment is stored and is not a scored source."

type InsightSummary struct {
	Method           string              `json:"method"`
	RequiredRows     int                 `json:"requiredRows"`
	AssessedRows     int                 `json:"assessedRows"`
	MissingRows      int                 `json:"missingRows"`
	BelowRows        int                 `json:"belowRows"`
	MetRows          int                 `json:"metRows"`
	CriticalSkillIDs []string            `json:"criticalSkillIds"`
	Departments      []DepartmentSummary `json:"departments"`
}

type DepartmentSummary struct {
	Department   string `json:"department"`
	RequiredRows int    `json:"requiredRows"`
	MissingRows  int    `json:"missingRows"`
	BelowRows    int    `json:"belowRows"`
	MetRows      int    `json:"metRows"`
}

// SummariseInsight counts only the people and skills already authorised for
// the caller. It does not invent a trend. Missing rows stay out of the
// confirmed shortage list.
func SummariseInsight(people []InsightPerson) InsightSummary {
	summary := InsightSummary{
		Method:           GapMethod,
		CriticalSkillIDs: []string{},
		Departments:      []DepartmentSummary{},
	}
	critical := map[string]struct{}{}
	order := []string{}
	byDepartment := map[string]*DepartmentSummary{}
	for _, person := range people {
		department := person.Department
		if department == "" {
			department = "Not recorded"
		}
		row := byDepartment[department]
		if row == nil {
			summary.Departments = append(summary.Departments, DepartmentSummary{Department: department})
			row = &summary.Departments[len(summary.Departments)-1]
			byDepartment[department] = row
		}
		for _, skill := range person.Skills {
			summary.RequiredRows++
			row.RequiredRows++
			switch skill.State {
			case GapMissing:
				summary.MissingRows++
				row.MissingRows++
			case GapBelow:
				summary.BelowRows++
				summary.AssessedRows++
				row.BelowRows++
				if _, seen := critical[skill.SkillID]; !seen {
					critical[skill.SkillID] = struct{}{}
					order = append(order, skill.SkillID)
				}
			case GapMet:
				summary.MetRows++
				summary.AssessedRows++
				row.MetRows++
			}
		}
	}
	summary.CriticalSkillIDs = order
	return summary
}
