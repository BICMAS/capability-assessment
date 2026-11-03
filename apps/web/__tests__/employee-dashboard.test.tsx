import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { gapBand, needsAttention, type CapabilityRow } from "@/features/employee/capability-data";
import { EmployeeAssessments } from "@/features/employee/assessments";
import { EmployeeDashboard } from "@/features/employee/dashboard";
import type { AssessmentSession } from "@/lib/generated/assessment-plan/plans";
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

function session(overrides: Partial<AssessmentSession> = {}): AssessmentSession {
  return {
    id: "session-1",
    planId: "plan-1",
    personId: "eli",
    status: "assigned",
    skills: [
      {
        instructions: "Answer clearly.",
        items: [{ id: "item-1", prompt: "Describe a tough sale.", answer: "" }],
      },
    ],
    ...overrides,
  };
}

describe("employee capability attention", () => {
  it("bands gaps into strength, developing, and priority", () => {
    expect(gapBand("-1.0")).toBe("strength");
    expect(gapBand("0.0")).toBe("strength");
    expect(gapBand("0.5")).toBe("developing");
    expect(gapBand("4.0")).toBe("priority");
  });

  it("flags unassessed and non-strength capabilities", () => {
    expect(needsAttention(capability({ explanation: null }))).toBe(true);
    expect(needsAttention(capability({ explanation: explanation({ gap: "0.0" }) }))).toBe(false);
    expect(needsAttention(capability({ explanation: explanation({ gap: "2.0" }) }))).toBe(true);
  });
});

describe("EmployeeDashboard", () => {
  it("shows waiting assessment count and attention from real rows", () => {
    render(
      <EmployeeDashboard
        name="Eli Okonkwo"
        roleLine="Account executive"
        sessions={{ ok: true, value: [session(), session({ id: "session-2", status: "submitted" })] }}
        capabilities={[
          capability({ explanation: explanation({ gap: "4.0" }) }),
          capability({
            id: "req-2",
            name: "Discovery",
            explanation: explanation({ gap: "0.0" }),
            pathway: {
              id: "path-1",
              organisationId: "org",
              personId: "eli",
              capabilityId: "discovery",
              targetLevel: 3,
              items: [
                { id: "step-1", stage: "learn", title: "Watch", status: "ready", deepLink: null },
              ],
            },
          }),
        ]}
      />,
    );

    expect(screen.getByRole("heading", { name: "Eli Okonkwo" })).toBeTruthy();
    expect(screen.getByText("Waiting assessments").closest("a")?.textContent).toContain("1");
    expect(screen.getByText("Needs attention").closest("a")?.textContent).toContain("1");
    expect(screen.getByText("Active pathways").closest("div")?.textContent).toContain("1");
    expect(screen.getByText("Negotiation")).toBeTruthy();
    expect(screen.getByText(/Gap 4\.0/)).toBeTruthy();
    expect(screen.getByText(/1 step on the development pathway/)).toBeTruthy();
  });

  it("surfaces assessment load errors without inventing counts", () => {
    render(
      <EmployeeDashboard
        name="Eli Okonkwo"
        roleLine=""
        sessions={{ ok: false, error: "Assessments are unavailable right now." }}
        capabilities={[]}
      />,
    );

    expect(screen.getByText("Assessments are unavailable right now.")).toBeTruthy();
    expect(screen.getByText("Waiting assessments").closest("a")?.textContent).toContain("—");
  });
});

describe("EmployeeAssessments", () => {
  it("renders a submit form for open sessions and read-only answers when submitted", () => {
    render(
      <EmployeeAssessments
        sessions={{
          ok: true,
          value: [
            session(),
            session({
              id: "session-2",
              status: "submitted",
              confirmed: true,
              proposals: [
                {
                  capabilityId: "negotiation",
                  source: "knowledge",
                  requiredLevel: 4,
                  percentage: 72,
                  bandLevels: [0, 1, 2, 3, 4],
                },
              ],
              skills: [
                {
                  instructions: "Review.",
                  items: [{ id: "item-2", prompt: "What worked?", answer: "Listening first." }],
                },
              ],
            }),
          ],
        }}
      />,
    );

    expect(screen.getByRole("button", { name: "Submit answers" })).toBeTruthy();
    expect(screen.getByDisplayValue("")).toBeTruthy();
    expect(screen.getByText("Listening first.")).toBeTruthy();
    expect(screen.getByText(/Knowledge 72% · Provisional/)).toBeTruthy();
  });

  it("shows empty and error states", () => {
    const { rerender } = render(<EmployeeAssessments sessions={{ ok: true, value: [] }} />);
    expect(screen.getByText("No assessment sessions are assigned to you yet.")).toBeTruthy();

    rerender(<EmployeeAssessments sessions={{ ok: false, error: "Assessments are unavailable right now." }} />);
    expect(screen.getByText("Assessments are unavailable right now.")).toBeTruthy();
  });
});
