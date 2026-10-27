//go:build integration

package framework

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/bicmas-academy/learn/services/capability-framework/migrations"
)

func TestSQLStoreAddRoleCapabilityIsAtomic(t *testing.T) {
	store := openIntegrationStore(t)
	ctx := context.Background()
	const organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"

	draft, err := store.CreateDraft(ctx, organisationID, "CI sales framework")
	if err != nil {
		t.Fatal(err)
	}
	role, err := store.AddRole(ctx, organisationID, draft.ID, "Area Sales Manager")
	if err != nil {
		t.Fatal(err)
	}
	capability, requirement, err := store.AddRoleCapability(ctx, organisationID, draft.ID, role.ID, "Negotiation", 4)
	if err != nil {
		t.Fatal(err)
	}
	if capability.Name != "Negotiation" || requirement.RequiredLevel != 4 || requirement.CapabilityID != capability.ID {
		t.Fatalf("capability=%#v requirement=%#v", capability, requirement)
	}

	loaded, err := store.GetVersion(ctx, organisationID, draft.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Capabilities) != 1 || len(loaded.Requirements) != 1 {
		t.Fatalf("loaded = %#v", loaded)
	}
}

func TestPublishWithEventCommitsVersionAndOutboxTogether(t *testing.T) {
	store := openIntegrationStore(t)
	ctx := context.Background()
	const organisationID = "8f0c2d44-6a1e-4b7a-9c3d-1e5f7a9b0c21"
	draft, err := store.CreateDraft(ctx, organisationID, "Outbox framework")
	if err != nil {
		t.Fatal(err)
	}
	occurredAt := time.Date(2026, 10, 11, 21, 0, 0, 0, time.UTC)
	event, err := newPublicationEvent(organisationID, draft.ID, "3c1e9a70-5b24-4d8f-a6e1-2f0b8c4d6e11", "corr-publish", occurredAt)
	if err != nil {
		t.Fatal(err)
	}
	published, err := store.PublishWithEvent(ctx, organisationID, draft.ID, event)
	if err != nil {
		t.Fatal(err)
	}
	if published.Status != StatusPublished {
		t.Fatalf("status = %s", published.Status)
	}
	var status string
	if err := store.database.QueryRowContext(ctx, `SELECT status FROM outbox_events WHERE id = $1`, event.ID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("outbox status = %s", status)
	}

	second, err := store.CreateDraft(ctx, organisationID, "Rejected publication")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.PublishWithEvent(ctx, organisationID, second.ID, OutboxEvent{}); err == nil {
		t.Fatal("expected invalid event to fail")
	}
	reloaded, err := store.GetVersion(ctx, organisationID, second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Status != StatusDraft {
		t.Fatalf("status = %s", reloaded.Status)
	}
	var extra int
	if err := store.database.QueryRowContext(ctx, `SELECT COUNT(*) FROM outbox_events WHERE aggregate_id = $1`, second.ID).Scan(&extra); err != nil {
		t.Fatal(err)
	}
	if extra != 0 {
		t.Fatalf("outbox rows = %d", extra)
	}
}

func openIntegrationStore(t *testing.T) SQLStore {
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
	return NewSQLStore(database)
}
