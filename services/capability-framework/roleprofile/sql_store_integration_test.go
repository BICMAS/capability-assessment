//go:build integration

package roleprofile_test

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bicmas-academy/learn/services/capability-framework/framework"
	"github.com/bicmas-academy/learn/services/capability-framework/migrations"
	"github.com/bicmas-academy/learn/services/capability-framework/roleprofile"
)

func TestSQLRoleProfileCommitsSkillsAndRollsBackFailures(t *testing.T) {
	database := openRoleProfileDB(t)
	store := roleprofile.NewSQLStore(database)
	frameworks := framework.NewSQLStore(database)
	ctx := context.Background()
	const organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	const actorID = "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11"

	draft, err := frameworks.CreateDraft(ctx, organisationID, "Referenced framework")
	if err != nil {
		t.Fatal(err)
	}
	role, err := store.CreateRole(ctx, organisationID, "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	skill, err := store.CreateSkill(ctx, organisationID, roleprofile.Skill{Name: "Negotiation", Category: roleprofile.CategoryDomain, Definition: "Agrees terms"})
	if err != nil {
		t.Fatal(err)
	}
	other, err := store.CreateSkill(ctx, "other-org", roleprofile.Skill{Name: "Negotiation", Category: roleprofile.CategoryDomain, Definition: "Other tenant"})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := store.CreateDraft(ctx, organisationID, role.ID, actorID, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if profile.Version.FrameworkVersionID != draft.ID || profile.Version.Status != roleprofile.StatusDraft {
		t.Fatalf("profile = %#v", profile.Version)
	}
	if _, err := store.GetProfile(ctx, "other-org", profile.ID); err != roleprofile.ErrNotFound {
		t.Fatalf("cross tenant = %v", err)
	}
	added, err := store.AddSkill(ctx, organisationID, profile.Version.ID, actorID, 1, roleprofile.SkillInput{
		SkillID:             skill.ID,
		Category:            roleprofile.CategoryDomain,
		Definition:          "Agrees terms",
		RequiredProficiency: 3,
		Priority:            roleprofile.PriorityRequired,
		Rationale:           "Core",
		Provenance:          roleprofile.ProvenanceManual,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddSkill(ctx, organisationID, profile.Version.ID, actorID, added.Revision, roleprofile.SkillInput{
		SkillID:             other.ID,
		Category:            roleprofile.CategoryDomain,
		Definition:          "Borrowed",
		RequiredProficiency: 1,
		Priority:            roleprofile.PriorityOptional,
		Rationale:           "No",
		Provenance:          roleprofile.ProvenanceManual,
	}); err != roleprofile.ErrNotFound {
		t.Fatalf("foreign skill = %v", err)
	}
	if _, err := store.AddSkill(ctx, organisationID, profile.Version.ID, actorID, added.Revision, roleprofile.SkillInput{
		SkillID:             skill.ID,
		Category:            roleprofile.CategoryDomain,
		Definition:          "Again",
		RequiredProficiency: 3,
		Priority:            roleprofile.PriorityRequired,
		Rationale:           "Core",
		Provenance:          roleprofile.ProvenanceImported,
	}); err != roleprofile.ErrDuplicate {
		t.Fatalf("duplicate = %v", err)
	}
	var revision int
	var skills int
	if err := database.QueryRowContext(ctx, `SELECT revision, (SELECT COUNT(*) FROM role_profile_skills WHERE profile_version_id = $1) FROM role_profile_versions WHERE id = $1`, profile.Version.ID).Scan(&revision, &skills); err != nil {
		t.Fatal(err)
	}
	if revision != added.Revision || skills != 1 {
		t.Fatalf("revision = %d skills = %d", revision, skills)
	}

	freshRole, err := store.CreateRole(ctx, organisationID, "Network Planner")
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := store.CreateDraft(ctx, organisationID, freshRole.ID, actorID, "")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.AttachAIDraft(ctx, organisationID, fresh.Version.ID, actorID, roleprofile.AIDraft{
		Model:                "fake-model",
		SourceContextVersion: "context-1",
		GeneratedAt:          time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC),
		Purpose:              "Plan the network",
		Skills: []roleprofile.SuggestedSkill{
			{SkillID: skill.ID, Definition: "Agrees terms", RequiredProficiency: 2, Priority: roleprofile.PriorityRequired},
			{SkillID: "missing-skill", Definition: "Unknown", RequiredProficiency: 1, Priority: roleprofile.PriorityOptional},
		},
	})
	if err == nil {
		t.Fatal("expected the draft insert to fail")
	}
	var purpose string
	if err := database.QueryRowContext(ctx, `SELECT purpose, revision, (SELECT COUNT(*) FROM role_profile_skills WHERE profile_version_id = role_profile_versions.id) FROM role_profile_versions WHERE id = $1`, fresh.Version.ID).Scan(&purpose, &revision, &skills); err != nil {
		t.Fatal(err)
	}
	if purpose != "" || revision != 1 || skills != 0 {
		t.Fatalf("rolled back purpose=%q revision=%d skills=%d", purpose, revision, skills)
	}
}

func openRoleProfileDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Fatal("INTEGRATION_DATABASE_URL is required for integration tests")
	}
	database, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := database.PingContext(ctx); err != nil {
		t.Fatalf("postgres unavailable: %v", err)
	}
	goose.SetBaseFS(migrations.FS)
	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatal(err)
	}
	if err := goose.Up(database, "."); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return database
}
