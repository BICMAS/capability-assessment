package roleprofile

import "strings"

func defaultApprovalPath() []ApprovalPathStage {
	return []ApprovalPathStage{
		{StageKey: StageHRReview, Label: "HR/L&D Review", RequiredRole: "hr", Position: 1},
		{StageKey: StageManagerReview, Label: "Line Manager Review", RequiredRole: "manager", Position: 2},
		{StageKey: StageLeaderApproval, Label: "Business Leader Approval", RequiredRole: "leader", Position: 3},
	}
}

func normaliseApprovalPath(stages []ApprovalPathStage) ([]ApprovalPathStage, error) {
	if len(stages) == 0 || len(stages) > 6 {
		return nil, ErrInvalid
	}
	seen := make(map[string]struct{}, len(stages))
	path := make([]ApprovalPathStage, len(stages))
	for i, stage := range stages {
		key := strings.TrimSpace(stage.StageKey)
		label := strings.TrimSpace(stage.Label)
		role := strings.TrimSpace(stage.RequiredRole)
		if !validStageKey(key) || label == "" || len([]rune(label)) > 80 || !validStageRole(role) || stage.Position != i+1 {
			return nil, ErrInvalid
		}
		if _, ok := seen[key]; ok {
			return nil, ErrInvalid
		}
		seen[key] = struct{}{}
		path[i] = ApprovalPathStage{StageKey: key, Label: label, RequiredRole: role, Position: i + 1}
	}
	return path, nil
}

func validStageKey(key string) bool {
	if key == "" || len(key) > 40 {
		return false
	}
	for _, char := range key {
		if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '_' {
			return false
		}
	}
	return true
}

func validStageRole(role string) bool {
	switch role {
	case "hr", "manager", "leader":
		return true
	default:
		return false
	}
}

func hasRole(roles []string, role string) bool {
	for _, candidate := range roles {
		if candidate == role {
			return true
		}
	}
	return false
}

func stepByKey(steps []WorkflowStep, key string) (WorkflowStep, bool) {
	for _, step := range steps {
		if step.StageKey == key {
			return step, true
		}
	}
	return WorkflowStep{}, false
}

func nextStep(steps []WorkflowStep, key string) (WorkflowStep, bool) {
	for index, step := range steps {
		if step.StageKey == key && index+1 < len(steps) {
			return steps[index+1], true
		}
	}
	return WorkflowStep{}, false
}

func lastStep(steps []WorkflowStep) (WorkflowStep, bool) {
	if len(steps) == 0 {
		return WorkflowStep{}, false
	}
	return steps[len(steps)-1], true
}

func cycleDecisions(decisions []ApprovalDecision, versionID string, cycle int) []ApprovalDecision {
	matched := make([]ApprovalDecision, 0)
	for _, decision := range decisions {
		if decision.ProfileVersionID == versionID && decision.SubmissionCycle == cycle {
			matched = append(matched, decision)
		}
	}
	return matched
}

func decisionForStage(decisions []ApprovalDecision, stage string) (ApprovalDecision, bool) {
	for _, decision := range decisions {
		if decision.StageKey == stage {
			return decision, true
		}
	}
	return ApprovalDecision{}, false
}

func stageRejected(decisions []ApprovalDecision) bool {
	for _, decision := range decisions {
		if decision.Decision == DecisionRejected {
			return true
		}
	}
	return false
}

func priorStagesApproved(steps []WorkflowStep, stage string, decisions []ApprovalDecision) bool {
	for _, step := range steps {
		if step.StageKey == stage {
			return true
		}
		decision, ok := decisionForStage(decisions, step.StageKey)
		if !ok || decision.Decision != DecisionApproved {
			return false
		}
	}
	return false
}

func finalStageReady(steps []WorkflowStep, stage string, decisions []ApprovalDecision) error {
	last, ok := lastStep(steps)
	if !ok || stage == "" || stage != last.StageKey {
		return ErrImmutable
	}
	if !priorStagesApproved(steps, stage, decisions) {
		return ErrImmutable
	}
	if _, decided := decisionForStage(decisions, stage); decided || stageRejected(decisions) {
		return ErrImmutable
	}
	return nil
}

func mayDecideStage(step WorkflowStep, assignment ApprovalAssignment, actorID string, roles []string, privileged bool) error {
	if privileged {
		return nil
	}
	if assignment.ID != "" && assignment.AssigneePersonID != actorID {
		return ErrForbidden
	}
	if !hasRole(roles, step.RequiredRole) {
		return ErrForbidden
	}
	return nil
}

func canResubmitVersion(version Version, actorID string, roles []string, decisions []ApprovalDecision) bool {
	if version.Status != StatusInReview || !stageRejected(cycleDecisions(decisions, version.ID, version.SubmissionCycle)) {
		return false
	}
	return hasRole(roles, "admin") || hasRole(roles, "hr") || actorID == version.SubmittedByPersonID
}

func approvalView(version Version, steps []WorkflowStep, assignments []ApprovalAssignment, decisions []ApprovalDecision, actorID string, roles []string) ApprovalRecord {
	current, _ := stepByKey(steps, version.WorkflowStage)
	assignee := assignmentForStage(assignments, version.ID, version.WorkflowStage)
	cycle := cycleDecisions(decisions, version.ID, version.SubmissionCycle)
	record := ApprovalRecord{
		Assignments:      nonNilAssignments(assignmentsForVersion(assignments, version.ID)),
		Decisions:        nonNilDecisions(decisionsForVersion(decisions, version.ID)),
		Steps:            nonNilSteps(steps),
		CurrentStage:     version.WorkflowStage,
		StageLabel:       current.Label,
		AssigneePersonID: assignee.AssigneePersonID,
		CanResubmit:      canResubmitVersion(version, actorID, roles, decisions),
	}
	switch {
	case version.Status == StatusPublished:
		record.Outstanding = "Approved"
	case record.CanResubmit:
		record.Outstanding = "Resubmit"
	case version.Status == StatusInReview:
		record.Outstanding = current.Label
		_, decided := decisionForStage(cycle, version.WorkflowStage)
		if current.StageKey != "" && !decided && !stageRejected(cycle) && mayDecideStage(current, assignee, actorID, roles, hasRole(roles, "admin")) == nil {
			record.CanDecide = true
		}
	case version.Status == StatusDraft && len(decisionsForVersion(decisions, version.ID)) > 0:
		record.Outstanding = "Revise and resubmit"
	default:
		record.Outstanding = ""
	}
	return record
}

func assignmentForStage(assignments []ApprovalAssignment, versionID, stage string) ApprovalAssignment {
	for _, assignment := range assignments {
		if assignment.ProfileVersionID == versionID && assignment.StageKey == stage {
			return assignment
		}
	}
	return ApprovalAssignment{}
}

func assignmentsForVersion(assignments []ApprovalAssignment, versionID string) []ApprovalAssignment {
	matched := make([]ApprovalAssignment, 0)
	for _, assignment := range assignments {
		if assignment.ProfileVersionID == versionID {
			matched = append(matched, assignment)
		}
	}
	return matched
}

func decisionsForVersion(decisions []ApprovalDecision, versionID string) []ApprovalDecision {
	matched := make([]ApprovalDecision, 0)
	for _, decision := range decisions {
		if decision.ProfileVersionID == versionID {
			matched = append(matched, decision)
		}
	}
	return matched
}

func nonNilSteps(steps []WorkflowStep) []WorkflowStep {
	if steps == nil {
		return []WorkflowStep{}
	}
	return steps
}

func nonNilAssignments(rows []ApprovalAssignment) []ApprovalAssignment {
	if rows == nil {
		return []ApprovalAssignment{}
	}
	return rows
}

func nonNilDecisions(rows []ApprovalDecision) []ApprovalDecision {
	if rows == nil {
		return []ApprovalDecision{}
	}
	return rows
}

func copyPath(stages []ApprovalPathStage) []ApprovalPathStage {
	return append([]ApprovalPathStage{}, stages...)
}

func planDecision(version Version, steps []WorkflowStep, assignments []ApprovalAssignment, decisions []ApprovalDecision, assignmentID, actorID, decision string, privileged bool, roles []string) (ApprovalAssignment, string, error) {
	step, ok := stepByKey(steps, version.WorkflowStage)
	if !ok {
		return ApprovalAssignment{}, "", ErrImmutable
	}
	cycle := cycleDecisions(decisions, version.ID, version.SubmissionCycle)
	if stageRejected(cycle) {
		return ApprovalAssignment{}, "", ErrImmutable
	}
	if _, decided := decisionForStage(cycle, step.StageKey); decided {
		return ApprovalAssignment{}, "", ErrDuplicate
	}
	assignment := assignmentForStage(assignments, version.ID, step.StageKey)
	if assignmentID != "" {
		found := ApprovalAssignment{}
		exists := false
		for _, candidate := range assignments {
			if candidate.ID == assignmentID && candidate.ProfileVersionID == version.ID {
				found = candidate
				exists = true
				break
			}
		}
		if !exists {
			return ApprovalAssignment{}, "", ErrNotFound
		}
		for _, decision := range cycle {
			if decision.AssignmentID == found.ID {
				return ApprovalAssignment{}, "", ErrDuplicate
			}
		}
		if found.StageKey != step.StageKey {
			return ApprovalAssignment{}, "", ErrImmutable
		}
		assignment = found
	}
	if err := mayDecideStage(step, assignment, actorID, roles, privileged); err != nil {
		return ApprovalAssignment{}, "", err
	}
	if !validDecision(strings.TrimSpace(decision)) {
		return ApprovalAssignment{}, "", ErrInvalid
	}
	return assignment, step.StageKey, nil
}
