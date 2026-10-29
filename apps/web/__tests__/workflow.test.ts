import { describe, expect, it } from "vitest";

import { buildWorkflow, stageStateLabel } from "@/features/role-profiles/workflow";
import type { RoleProfile } from "@/lib/role-profile";

function profile(overrides: Partial<RoleProfile["version"]> = {}): RoleProfile {
  return {
    id: "profile-1",
    organisationId: "org",
    roleId: "role-1",
    roleName: "Regional Sales Manager",
    createdByPersonId: "ada",
    createdAt: "2026-10-27T00:00:00Z",
    version: {
      id: "version-1",
      profileId: "profile-1",
      organisationId: "org",
      versionNumber: 1,
      revision: 1,
      status: "profile_draft",
      purpose: "",
      responsibilities: "",
      businessOutcomes: "",
      decisionScope: "",
      industry: "",
      region: "",
      teamSize: null,
      reportingLevel: "",
      frameworkVersionId: "",
      sources: [],
      skills: [],
      ai: { model: "", sourceContextVersion: "", generatedAt: null, draft: null },
      createdAt: "2026-10-27T00:00:00Z",
      ...overrides,
    },
  };
}

function stage(id: string, input: Parameters<typeof buildWorkflow>[0]) {
  const found = buildWorkflow(input).find((item) => item.id === id);
  if (!found) {
    throw new Error(id);
  }
  return found;
}

describe("role skills workflow", () => {
  it("does not mark an organogram complete when the list is empty or unknown", () => {
    const empty = stage("organogram", { current: "skills", profile: profile(), evidence: { organogramCount: 0, clarificationComplete: null } });
    const unknown = stage("organogram", { current: "skills", profile: profile(), evidence: { organogramCount: null, clarificationComplete: null } });
    expect(empty.status).toBe("pending");
    expect(unknown.status).toBe("pending");
    expect(stage("organogram", { current: "role", profile: profile(), evidence: { organogramCount: 2, clarificationComplete: null } }).status).toBe("completed");
  });

  it("keeps later stages blocked until a profile exists", () => {
    const role = stage("role", { current: "organogram" });
    const insights = stage("insights", { current: "organogram" });
    expect(role.status).toBe("blocked");
    expect(role.href).toBe("");
    expect(insights.status).toBe("blocked");
  });

  it("records role information only when the required context is present", () => {
    expect(stage("role", { current: "skills", profile: profile() }).done).toBe(false);
    const recorded = stage("role", {
      current: "skills",
      profile: profile({
        purpose: "Grow the region",
        responsibilities: "Lead the team",
        businessOutcomes: "Meet the target",
        decisionScope: "Pricing",
        reportingLevel: "Director",
        teamSize: 6,
        positionId: "position-1",
      }),
    });
    expect(recorded.status).toBe("completed");
  });

  it("does not treat an unknown clarification session as complete", () => {
    const unknown = stage("clarify", { current: "role", profile: profile(), evidence: { organogramCount: null, clarificationComplete: null } });
    expect(unknown.done).toBe(false);
    expect(stage("clarify", { current: "role", profile: profile(), evidence: { organogramCount: null, clarificationComplete: true } }).status).toBe("completed");
  });

  it("marks generation from the recorded draft and approval only after publication", () => {
    const draft = profile({
      status: "profile_draft",
      skills: [
        {
          id: "row",
          skillId: "skill",
          skillName: "Negotiation",
          category: "domain",
          definition: "Agrees terms",
          requiredProficiency: 3,
          priority: "required",
          rationale: "Needed",
          provenance: "manually_added",
          position: 1,
        },
      ],
    });
    expect(stage("generate", { current: "skills", profile: draft }).done).toBe(false);
    expect(stage("skills", { current: "generate", profile: draft }).done).toBe(false);
    const generated = profile({
      ...draft.version,
      ai: { model: "model", sourceContextVersion: "1", generatedAt: "2026-10-27T00:00:00Z", draft: null },
    });
    expect(stage("generate", { current: "skills", profile: generated }).status).toBe("completed");
    const reviewing = profile({ ...generated.version, status: "profile_in_review" });
    expect(stage("skills", { current: "generate", profile: reviewing }).status).toBe("completed");
    const approval = stage("approve", { current: "skills", profile: reviewing });
    expect(approval.status).toBe("pending");
    expect(approval.tone).toBe("attention");
    expect(stage("approve", { current: "skills", profile: profile({ ...reviewing.version, status: "profile_published" }) }).status).toBe("completed");
  });

  it("never marks insights complete and opens them only for a published version", () => {
    const published = profile({ status: "profile_published" });
    const insights = stage("insights", { current: "approve", profile: published });
    expect(insights.done).toBe(false);
    expect(insights.status).toBe("pending");
    expect(insights.href).toBe("/admin/role-profiles/profile-1/insights");
    expect(stageStateLabel(stage("insights", { current: "insights", profile: published }))).toBe("Current");
    expect(stage("insights", { current: "skills", profile: profile() }).href).toBe("");
  });
});
