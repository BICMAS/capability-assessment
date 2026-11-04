import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/app/(manager)/manager/actions", () => ({
  saveManagerAssessment: vi.fn(),
}));

import { PersonRecord } from "@/app/(manager)/manager/team";
import { teamSummary, type TeamCapability } from "@/features/manager/capability";
import { TeamDirectory } from "@/features/manager/team-directory";
import type { TeamMemberSummary } from "@/features/manager/team-data";
import { memberAttentionLine, needsAttention, summariseAttention } from "@/features/manager/team-summary";
import type { Explanation } from "@/lib/generated/assessment-gap/levels";

afterEach(() => {
  cleanup();
});

function explanation(gap: string): Explanation {
  return {
    id: "level-1",
    organisationId: "org",
    personId: "eli",
    frameworkVersionId: "version-1",
    capabilityCode: "negotiation",
    knowledgePercentage: 40,
    situationalJudgementPercentage: null,
    aiInterviewPercentage: null,
    simulationPercentage: null,
    managerAssessmentPercentage: null,
    selfAssessmentPercentage: null,
    mean: "1",
    divisor: 1,
    requiredLevel: "4",
    currentLevel: "1.0",
    gap,
    status: "verified",
  };
}

function capability(name: string, value: Explanation | null): TeamCapability {
  return {
    name,
    capabilityCode: name,
    frameworkVersionId: "version-1",
    requiredLevel: 4,
    explanation: value,
  };
}

function member(overrides: Partial<TeamMemberSummary> = {}): TeamMemberSummary {
  return {
    personId: "eli",
    name: "Eli Employee",
    departmentName: "Sales",
    roles: ["Area Sales Manager"],
    requiredCount: 1,
    assessedCount: 0,
    priorityCount: 0,
    developingCount: 0,
    priorityNames: [],
    belowNames: [],
    levelsOk: true,
    levelsError: "",
    ...overrides,
  };
}

describe("manager team summaries", () => {
  it("summarises an unassessed person in one line", () => {
    expect(teamSummary([capability("Negotiation", null)])).toBe("Not yet assessed");
    expect(memberAttentionLine(member())).toBe("Not yet assessed");
  });

  it("names capabilities that are below the required level", () => {
    expect(teamSummary([capability("Negotiation", explanation("3.0"))])).toBe(
      "1 capability below the required level: Negotiation",
    );
    expect(
      memberAttentionLine(
        member({
          assessedCount: 1,
          priorityCount: 1,
          belowNames: ["Negotiation"],
          priorityNames: ["Negotiation"],
        }),
      ),
    ).toBe("1 capability below the required level: Negotiation");
  });

  it("flags attention from priority gaps and missing assessments", () => {
    expect(needsAttention(member())).toBe(true);
    expect(needsAttention(member({ assessedCount: 1, priorityCount: 1, belowNames: ["Negotiation"] }))).toBe(true);
    expect(needsAttention(member({ assessedCount: 1, priorityCount: 0, developingCount: 0, belowNames: [] }))).toBe(
      false,
    );
    expect(
      summariseAttention([
        member(),
        member({ personId: "maya", assessedCount: 1, priorityCount: 1, belowNames: ["Negotiation"] }),
        member({ personId: "sam", assessedCount: 1, priorityCount: 0, belowNames: [] }),
      ]),
    ).toEqual({
      members: 3,
      needsAttention: 2,
      withPriorityGaps: 1,
      notYetAssessed: 1,
      levelFailures: 0,
    });
  });

  it("offers the same assessment form for a capability that has no score", () => {
    render(
      <PersonRecord personId="eli" personName="Eli Employee" roles={["Area Sales Manager"]} capabilities={[capability("Negotiation", null)]} />,
    );

    expect(screen.queryByRole("heading", { name: "Strengths" })).toBeNull();
    expect(screen.getByRole("heading", { name: "Not yet assessed" })).toBeTruthy();
    expect(screen.getByText("Required level 4")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Save manager assessment" })).toBeTruthy();
    expect(screen.getByLabelText("Evidence")).toBeTruthy();
  });
});

describe("TeamDirectory", () => {
  it("searches and filters team members while keeping person links", () => {
    render(
      <TeamDirectory
        people={{ ok: true, value: [] }}
        frameworks={{ ok: true, value: [] }}
        members={[
          member({
            assessedCount: 1,
            priorityCount: 1,
            belowNames: ["Negotiation"],
            priorityNames: ["Negotiation"],
          }),
          member({
            personId: "sam",
            name: "Sam Seller",
            roles: ["Account executive"],
            assessedCount: 1,
            priorityCount: 0,
            belowNames: [],
          }),
        ]}
      />,
    );

    expect(screen.getByRole("heading", { name: "Team attention" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "My team" })).toBeTruthy();
    expect(screen.getAllByRole("link", { name: "Open record" })[0]?.getAttribute("href")).toBe("/manager/people/eli");

    fireEvent.change(screen.getByPlaceholderText("Search by name, department, role, or capability"), {
      target: { value: "Sam" },
    });
    expect(screen.getAllByText("Sam Seller").length).toBeGreaterThan(0);
    expect(screen.queryByText("Eli Employee")).toBeNull();

    fireEvent.change(screen.getByPlaceholderText("Search by name, department, role, or capability"), {
      target: { value: "" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Has priority gaps" }));
    expect(screen.getAllByText("Eli Employee").length).toBeGreaterThan(0);
    expect(screen.queryByText("Sam Seller")).toBeNull();

    const table = screen.getByRole("table");
    expect(within(table).getByRole("link", { name: "Open record" }).getAttribute("href")).toBe("/manager/people/eli");
  });

  it("shows empty and error states from settle results", () => {
    const { rerender } = render(
      <TeamDirectory
        people={{ ok: false, error: "People are unavailable right now." }}
        frameworks={{ ok: true, value: [] }}
        members={[]}
      />,
    );
    expect(screen.getByText("People are unavailable right now.")).toBeTruthy();

    rerender(<TeamDirectory people={{ ok: true, value: [] }} frameworks={{ ok: true, value: [] }} members={[]} />);
    expect(screen.getByText("No people are in your team list yet.")).toBeTruthy();
  });
});
