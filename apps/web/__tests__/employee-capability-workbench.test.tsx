import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { EmployeeCapabilities } from "@/features/employee/capabilities";
import {
  bandSections,
  gapBand,
  groupByFramework,
  needsAttention,
  rowBand,
  type CapabilityRow,
} from "@/features/employee/capability-data";
import type { Explanation } from "@/lib/generated/assessment-gap/levels";

afterEach(() => {
  cleanup();
});

function explanation(overrides: Partial<Explanation> = {}): Explanation {
  return {
    id: "level-1",
    organisationId: "org",
    personId: "eli",
    frameworkVersionId: "version-1",
    capabilityCode: "negotiation",
    knowledgePercentage: null,
    situationalJudgementPercentage: null,
    aiInterviewPercentage: null,
    simulationPercentage: null,
    managerAssessmentPercentage: null,
    selfAssessmentPercentage: null,
    mean: "0",
    divisor: 1,
    requiredLevel: "4",
    currentLevel: "0.0",
    gap: "4.0",
    status: "verified",
    ...overrides,
  };
}

function capability(overrides: Partial<CapabilityRow> = {}): CapabilityRow {
  const explanationValue = Object.prototype.hasOwnProperty.call(overrides, "explanation")
    ? overrides.explanation!
    : explanation();
  return {
    id: "req-1",
    name: "Negotiation",
    frameworkVersionId: "version-1",
    frameworkName: "Core",
    versionNumber: 1,
    capabilityCode: "negotiation",
    requiredLevel: 4,
    explanation: explanationValue,
    pathway: null,
    level: { ok: true, value: explanationValue },
    pathwayResult: { ok: true, value: null },
    ...overrides,
  };
}

describe("employee gap bands", () => {
  it("matches manager strength / developing / priority rules", () => {
    expect(gapBand("-1.0")).toBe("strength");
    expect(gapBand("0.0")).toBe("strength");
    expect(gapBand("0.5")).toBe("developing");
    expect(gapBand("4.0")).toBe("priority");
  });

  it("orders band sections like the manager person record", () => {
    const sections = bandSections([
      capability({ id: "s", explanation: explanation({ gap: "0.0" }), level: { ok: true, value: explanation({ gap: "0.0" }) } }),
      capability({ id: "p", explanation: explanation({ gap: "2.0" }), level: { ok: true, value: explanation({ gap: "2.0" }) } }),
      capability({ id: "u", explanation: null, level: { ok: true, value: null } }),
      capability({
        id: "d",
        explanation: explanation({ gap: "0.4" }),
        level: { ok: true, value: explanation({ gap: "0.4" }) },
      }),
    ]);

    expect(sections.map((section) => section.key)).toEqual(["priority", "developing", "strength", "unassessed"]);
    expect(sections[0]?.titleClass).toBe("band-priority");
    expect(sections[2]?.titleClass).toBe("band-strength");
  });

  it("groups rows by framework version", () => {
    const groups = groupByFramework([
      capability({ id: "a" }),
      capability({
        id: "b",
        frameworkVersionId: "version-2",
        frameworkName: "Leadership",
        versionNumber: 2,
      }),
    ]);
    expect(groups).toHaveLength(2);
    expect(groups[0]?.frameworkName).toBe("Core");
    expect(groups[1]?.frameworkName).toBe("Leadership");
  });

  it("treats failed level loads as attention items", () => {
    expect(needsAttention(capability({ explanation: null, level: { ok: true, value: null } }))).toBe(true);
    expect(
      needsAttention(
        capability({
          explanation: null,
          level: { ok: false, error: "Capability level is unavailable right now." },
        }),
      ),
    ).toBe(true);
    expect(rowBand(capability({ level: { ok: false, error: "Capability level is unavailable right now." }, explanation: null }))).toBe(
      "unavailable",
    );
  });
});

describe("EmployeeCapabilities workbench", () => {
  it("shows empty state when no role titles are recorded", () => {
    render(
      <EmployeeCapabilities
        personId="eli"
        hasRoleTitles={false}
        frameworks={{ ok: true, value: [] }}
        data={{ rows: [], levelFailures: 0, pathwayFailures: 0 }}
      />,
    );

    expect(screen.getByText(/No role is recorded on your profile yet/)).toBeTruthy();
  });

  it("shows empty state when role titles match no requirements", () => {
    render(
      <EmployeeCapabilities
        personId="eli"
        hasRoleTitles
        frameworks={{ ok: true, value: [] }}
        data={{ rows: [], levelFailures: 0, pathwayFailures: 0 }}
      />,
    );

    expect(screen.getByText(/No capability requirements match your recorded role titles yet/)).toBeTruthy();
  });

  it("renders framework groups, manager band headings, and a dedicated self-assessment panel", () => {
    render(
      <EmployeeCapabilities
        personId="eli"
        hasRoleTitles
        frameworks={{ ok: true, value: [] }}
        data={{
          rows: [
            capability({
              id: "priority",
              name: "Negotiation",
              explanation: explanation({ gap: "4.0" }),
              level: { ok: true, value: explanation({ gap: "4.0" }) },
            }),
            capability({
              id: "strength",
              name: "Discovery",
              capabilityCode: "discovery",
              explanation: explanation({ gap: "0.0", capabilityCode: "discovery" }),
              level: { ok: true, value: explanation({ gap: "0.0", capabilityCode: "discovery" }) },
            }),
          ],
          levelFailures: 0,
          pathwayFailures: 0,
        }}
      />,
    );

    expect(screen.getByRole("heading", { name: "Core v1" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Priority gaps" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Strengths" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Negotiation" })).toBeTruthy();
    expect(screen.getAllByRole("heading", { name: "Self-assessment" })).toHaveLength(2);
    expect(screen.getAllByText(/It is not included in the calculated level/)).toHaveLength(2);
    expect(screen.getAllByRole("button", { name: "Save self-assessment" })).toHaveLength(2);
  });

  it("surfaces framework and level load failures", () => {
    render(
      <EmployeeCapabilities
        personId="eli"
        hasRoleTitles
        frameworks={{ ok: false, error: "Frameworks are unavailable right now." }}
        data={{
          rows: [
            capability({
              explanation: null,
              level: { ok: false, error: "Capability level is unavailable right now." },
            }),
          ],
          levelFailures: 1,
          pathwayFailures: 0,
        }}
      />,
    );

    expect(screen.getByText("Frameworks are unavailable right now.")).toBeTruthy();
    expect(screen.getByText("1 capability level could not be loaded.")).toBeTruthy();
    expect(screen.getByText("Capability level is unavailable right now.")).toBeTruthy();
  });
});
