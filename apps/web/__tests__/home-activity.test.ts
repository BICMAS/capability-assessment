import { describe, expect, it } from "vitest";

import { activityStatus, describeActivity } from "@/features/home/activity";

describe("home activity copy", () => {
  it("turns a sign-in audit row into a readable sentence", () => {
    const record = {
      actorPersonId: "ada",
      organisationId: "bicmas",
      action: "person.signed_in",
      entityType: "person",
      entityId: "ada",
      occurredAt: "2026-10-29T20:52:00Z",
    };
    expect(describeActivity(record, "Ada Admin")).toBe("Ada Admin signed in");
    expect(activityStatus(record)).toBe("Signed in");
  });

  it("keeps an unknown action readable without inventing a story", () => {
    const record = {
      actorPersonId: "maya",
      organisationId: "bicmas",
      action: "framework.published",
      entityType: "framework",
      entityId: "v1",
      occurredAt: "2026-10-29T20:52:00Z",
    };
    expect(describeActivity(record, "Maya Manager")).toBe("Maya Manager recorded framework published");
  });
});
