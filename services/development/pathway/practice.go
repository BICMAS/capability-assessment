package pathway

import "strings"

const PracticePrototype = "Text practice prototype. This is not a live voice session."

type Scenario struct {
	Key     string
	Title   string
	Opening string
}

var Scenarios = []Scenario{
	{Key: "sales_negotiation", Title: "Sales and distributor negotiations", Opening: "A distributor asks for a larger discount than the approved range."},
	{Key: "customer_relationship", Title: "Customer relationship management", Opening: "A long-standing customer says a recent delivery missed the agreed date."},
	{Key: "team_leadership", Title: "Team leadership", Opening: "Two team members disagree about who owns the next customer visit."},
	{Key: "conflict_resolution", Title: "Conflict resolution", Opening: "A colleague says your last decision left their team out of the plan."},
	{Key: "performance_conversation", Title: "Performance conversations", Opening: "You are preparing to discuss a missed target with a team member."},
}

func ScenarioByKey(key string) (Scenario, bool) {
	for _, scenario := range Scenarios {
		if scenario.Key == key {
			return scenario, true
		}
	}
	return Scenario{}, false
}

// Reply returns a text prototype for a known scenario.
// It does not call a model or a voice service.
func Reply(scenarioKey, message string) (string, error) {
	scenario, ok := ScenarioByKey(scenarioKey)
	message = strings.TrimSpace(message)
	if !ok || message == "" || len([]rune(message)) > 2000 {
		return "", ErrInvalid
	}
	return PracticePrototype + " " + scenario.Title + ". You said: " + message + " " + scenario.Opening + " The prototype asks what you would do next. It does not score the answer.", nil
}
