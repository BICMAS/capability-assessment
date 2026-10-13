package levels

import (
	"context"
	"testing"
)

func TestManagerOverrideStoresTheChange(t *testing.T) {
	store := NewMemory()
	explanation := saveRegression(t, store, nil)

	row, err := store.Override(context.Background(), OverrideInput{
		OrganisationID: regressionOrganisationID,
		ExplanationID:  explanation.ID,
		ActorPersonID:  "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33",
		Reason:         "Observed a live negotiation",
		NewValue:       31,
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.ActorPersonID != "1a9e4c55-2d67-4f08-b3e6-8c0a7d5f1b33" || row.Reason != "Observed a live negotiation" {
		t.Fatalf("override = %#v", row)
	}
	if row.PreviousValue.String() != "2.8" || row.NewValue.String() != "3.1" {
		t.Fatalf("previous = %s new = %s", row.PreviousValue, row.NewValue)
	}

	loaded, err := store.Get(context.Background(), regressionOrganisationID, regressionPersonID, regressionFrameworkID, regressionCapability)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.CurrentLevel.String() != "2.8" || loaded.Gap.String() != "1.2" || loaded.Status != StatusProvisional {
		t.Fatalf("calculated level changed to %s gap %s status %s", loaded.CurrentLevel, loaded.Gap, loaded.Status)
	}
}
