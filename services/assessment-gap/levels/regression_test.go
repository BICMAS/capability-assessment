package levels

import (
	"context"
	"testing"
)

const (
	regressionOrganisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	regressionPersonID       = "7d2b6f18-9c40-4e15-8a73-5b1d0e6f4a22"
	regressionFrameworkID    = "9a1c4e72-6b30-4f58-8d14-2c7e5b0a3d66"
	regressionCapability     = "NEG"
)

func TestRegressionCurrentLevel(t *testing.T) {
	store := NewMemory()
	got := saveRegression(t, store, nil)

	if got.CurrentLevel.String() != "2.8" || got.Gap.String() != "1.2" {
		t.Fatalf("currentLevel = %s gap = %s", got.CurrentLevel, got.Gap)
	}
	assertExplanation(t, got)
	if got.SelfAssessment != nil {
		t.Fatal("self-assessment was stored")
	}

	loaded, err := store.Get(context.Background(), regressionOrganisationID, regressionPersonID, regressionFrameworkID, regressionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentLevel.String() != "2.8" || loaded.Gap.String() != "1.2" || loaded.ID != got.ID {
		t.Fatalf("loaded = %s gap %s", loaded.CurrentLevel, loaded.Gap)
	}
	assertAttempts(t, store, false)
}

func TestSelfAssessmentDoesNotChangeCurrentLevel(t *testing.T) {
	store := NewMemory()
	self := 91
	got := saveRegression(t, store, &self)

	if got.CurrentLevel.String() != "2.8" || got.Gap.String() != "1.2" {
		t.Fatalf("currentLevel = %s gap = %s", got.CurrentLevel, got.Gap)
	}
	if got.SelfAssessment == nil || *got.SelfAssessment != 91 {
		t.Fatal("self-assessment was not stored")
	}
	assertExplanation(t, got)

	loaded, err := store.Get(context.Background(), regressionOrganisationID, regressionPersonID, regressionFrameworkID, regressionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentLevel.String() != "2.8" || loaded.Gap.String() != "1.2" {
		t.Fatalf("loaded currentLevel = %s gap = %s", loaded.CurrentLevel, loaded.Gap)
	}
	assertAttempts(t, store, true)
}

func TestRecordRejectsSimulationAndStoresAIInterview(t *testing.T) {
	store := NewMemory()
	_, err := store.Record(context.Background(), AttemptInput{
		OrganisationID:     regressionOrganisationID,
		PersonID:           regressionPersonID,
		FrameworkVersionID: regressionFrameworkID,
		CapabilityCode:     regressionCapability,
		Source:             SourceSimulation,
		Percentage:         68,
		EvidenceIDs:        []string{"evidence-1"},
		RequiredLevel:      40,
	})
	if err != ErrNoWriter {
		t.Fatalf("simulation error = %v", err)
	}
	got, err := store.Record(context.Background(), AttemptInput{
		OrganisationID:     regressionOrganisationID,
		PersonID:           regressionPersonID,
		FrameworkVersionID: regressionFrameworkID,
		CapabilityCode:     regressionCapability,
		Source:             SourceAIInterview,
		Percentage:         75,
		EvidenceIDs:        []string{"session-1"},
		RequiredLevel:      40,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.AIInterview == nil || *got.AIInterview != 75 || got.CurrentLevel.String() != "3.0" || got.Gap.String() != "1.0" {
		t.Fatalf("explanation = %+v", got)
	}
}

func saveRegression(t *testing.T, store *Memory, self *int) Explanation {
	t.Helper()
	sources := regressionSources(self)
	got, err := store.Calculate(context.Background(), Calculation{
		OrganisationID:     regressionOrganisationID,
		PersonID:           regressionPersonID,
		FrameworkVersionID: regressionFrameworkID,
		CapabilityCode:     regressionCapability,
		Sources:            sources,
		RequiredLevel:      40,
		Evidence:           evidenceFor(sources),
	})
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func regressionSources(self *int) Sources {
	knowledge := 82
	situational := 74
	interview := 68
	simulation := 61
	manager := 65
	return Sources{
		Knowledge:            &knowledge,
		SituationalJudgement: &situational,
		AIInterview:          &interview,
		Simulation:           &simulation,
		ManagerAssessment:    &manager,
		SelfAssessment:       self,
	}
}

func evidenceFor(sources Sources) map[string][]string {
	evidence := map[string][]string{}
	if sources.Knowledge != nil {
		evidence[SourceKnowledge] = []string{"evidence-knowledge"}
	}
	if sources.SituationalJudgement != nil {
		evidence[SourceSituationalJudgement] = []string{"evidence-situational"}
	}
	if sources.ManagerAssessment != nil {
		evidence[SourceManagerAssessment] = []string{"evidence-manager"}
	}
	if sources.SelfAssessment != nil {
		evidence[SourceSelfAssessment] = []string{"evidence-self"}
	}
	return evidence
}

func assertExplanation(t *testing.T, got Explanation) {
	t.Helper()
	if got.FrameworkVersionID != regressionFrameworkID {
		t.Fatalf("frameworkVersionId = %s", got.FrameworkVersionID)
	}
	if got.Status != StatusProvisional {
		t.Fatalf("status = %s", got.Status)
	}
	if got.Mean.String() != "70.0" || got.Divisor != 25 || got.RequiredLevel.String() != "4.0" {
		t.Fatalf("mean = %s divisor = %d required = %s", got.Mean, got.Divisor, got.RequiredLevel)
	}
	assertPercentage(t, "knowledge", got.Knowledge, 82)
	assertPercentage(t, "situational judgement", got.SituationalJudgement, 74)
	assertPercentage(t, "ai interview", got.AIInterview, 68)
	assertPercentage(t, "simulation", got.Simulation, 61)
	assertPercentage(t, "manager assessment", got.ManagerAssessment, 65)
}

func assertPercentage(t *testing.T, name string, got *int, want int) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("%s = %v", name, got)
	}
}

func assertAttempts(t *testing.T, store *Memory, withSelf bool) {
	t.Helper()
	seen := map[string]bool{}
	for _, attempt := range store.attemptsFor(regressionOrganisationID, regressionPersonID, regressionFrameworkID, regressionCapability) {
		seen[attempt.Source] = true
		if len(attempt.EvidenceIDs) == 0 {
			t.Fatalf("%s attempt has no evidence", attempt.Source)
		}
	}
	if !seen[SourceKnowledge] || !seen[SourceSituationalJudgement] || !seen[SourceManagerAssessment] {
		t.Fatalf("attempts = %#v", seen)
	}
	if seen[SourceAIInterview] || seen[SourceSimulation] {
		t.Fatal("ai interview and simulation were stored as attempts")
	}
	if withSelf != seen[SourceSelfAssessment] {
		t.Fatalf("self-assessment stored = %v", seen[SourceSelfAssessment])
	}
}
