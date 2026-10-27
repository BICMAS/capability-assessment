import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("@/app/(manager)/manager/actions", () => ({
  saveManagerAssessment: vi.fn(),
}));

import { PersonRecord, TeamRow, type TeamCapability } from "@/app/(manager)/manager/team";
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

function capability(name: string, explanation: Explanation | null): TeamCapability {
  return {
    name,
    capabilityCode: name,
    frameworkVersionId: "version-1",
    requiredLevel: 4,
    explanation,
  };
}

describe("manager team", () => {
  it("summarises an unassessed person in one line", () => {
    render(<TeamRow personId="eli" personName="Eli Employee" roles={["Area Sales Manager"]} capabilities={[capability("Negotiation", null)]} />);

    expect(screen.getByText("Not yet assessed")).toBeTruthy();
    expect(screen.queryByText("None")).toBeNull();
    expect(screen.queryByText("Strengths")).toBeNull();
  });

  it("names capabilities that are below the required level", () => {
    render(
      <TeamRow
        personId="eli"
        personName="Eli Employee"
        roles={["Area Sales Manager"]}
        capabilities={[capability("Negotiation", explanation("3.0"))]}
      />,
    );

    expect(screen.getByText("1 capability below the required level: Negotiation")).toBeTruthy();
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
