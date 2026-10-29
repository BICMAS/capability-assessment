import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { InsightsPanel } from "@/features/role-profiles/insights-panel";
import type { RoleProfile } from "@/lib/role-profile";

const profile: RoleProfile = {
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
    versionNumber: 2,
    revision: 3,
    status: "profile_published",
    purpose: "Grow the region",
    responsibilities: "Lead the team",
    businessOutcomes: "Meet the target",
    decisionScope: "Pricing",
    industry: "FMCG",
    region: "UK",
    teamSize: 6,
    reportingLevel: "Director",
    frameworkVersionId: "",
    sources: [],
    skills: [
      {
        id: "row",
        skillId: "negotiation",
        skillName: "Negotiation",
        category: "domain",
        definition: "Agrees terms",
        requiredProficiency: 3,
        priority: "required",
        rationale: "Needed",
        provenance: "hr_adjusted",
        position: 1,
      },
    ],
    ai: { model: "", sourceContextVersion: "", generatedAt: null, draft: null },
    createdAt: "2026-10-27T00:00:00Z",
  },
};

afterEach(() => {
  cleanup();
});

describe("profile insights", () => {
  it("waits for a published version", () => {
    render(<InsightsPanel profile={{ ...profile, version: { ...profile.version, status: "profile_draft" } }} insight={null} />);
    expect(screen.getByText("Insights are available after this version is published.")).toBeTruthy();
    expect(screen.queryByText("0")).toBeNull();
  });

  it("shows a missing assessment without a score", () => {
    render(
      <InsightsPanel
        profile={profile}
        insight={{
          ok: true,
          value: {
            profileVersionId: "version-1",
            population: ["person-1"],
            employmentDecision: "not_an_employment_decision",
            people: [{ personId: "person-1", skills: [{ skillId: "negotiation", requiredProficiency: 3, state: "missing" }] }],
          },
        }}
      />,
    );
    expect(screen.getByText("This result is not an employment decision. A missing assessment is not recorded as proficiency zero.")).toBeTruthy();
    expect(screen.getByText("Negotiation")).toBeTruthy();
    expect(screen.getByText("missing")).toBeTruthy();
    expect(screen.getByText("Not assessed")).toBeTruthy();
  });

  it("summarises recorded gaps without treating a missing assessment as a shortage", () => {
    render(
      <InsightsPanel
        profile={profile}
        insight={{
          ok: true,
          value: {
            profileVersionId: "version-1",
            population: ["person-1"],
            employmentDecision: "not_an_employment_decision",
            summary: {
              method: "Proficiency uses integers 0 to 4.",
              requiredRows: 1,
              assessedRows: 0,
              missingRows: 1,
              belowRows: 0,
              metRows: 0,
              criticalSkillIds: [],
              departments: [{ department: "Sales", requiredRows: 1, missingRows: 1, belowRows: 0, metRows: 0 }],
            },
            people: [{ personId: "person-1", skills: [{ skillId: "negotiation", requiredProficiency: 3, state: "missing" }] }],
          },
        }}
      />,
    );
    expect(screen.getByText("No confirmed skill shortage is recorded. A missing assessment is not counted as a shortage.")).toBeTruthy();
    expect(screen.getByText("Sales")).toBeTruthy();
    expect(screen.getByText("Gap closure over time is not shown because only the latest stored level is available.")).toBeTruthy();
  });

  it("shows an empty population and a service failure", () => {
    const { rerender } = render(
      <InsightsPanel
        profile={profile}
        insight={{ ok: true, value: { profileVersionId: "version-1", population: [], employmentDecision: "not_an_employment_decision", people: [] } }}
      />,
    );
    expect(screen.getByText("No assessed people are visible for this published version.")).toBeTruthy();
    rerender(<InsightsPanel profile={profile} insight={{ ok: false, error: "Insights are unavailable right now." }} />);
    expect(screen.getByRole("alert").textContent).toBe("Insights are unavailable right now.");
  });
});
