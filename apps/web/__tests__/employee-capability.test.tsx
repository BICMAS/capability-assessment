import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { CapabilityInstrument, RequiredOnly } from "@/components/capability-instrument";
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

describe("employee capability card", () => {
  it("keeps an unassessed capability to one line", () => {
    render(<RequiredOnly name="Negotiation" requiredLevel={4} />);

    expect(screen.getByRole("heading", { name: "Negotiation" })).toBeTruthy();
    expect(screen.getByText("Required level 4")).toBeTruthy();
    expect(screen.getByText("Not yet assessed")).toBeTruthy();
    expect(screen.queryByText("Knowledge")).toBeNull();
  });

  it("shows scored sources and lists the ones still missing", () => {
    render(<CapabilityInstrument name="Negotiation" explanation={explanation({ knowledgePercentage: 80 })} />);

    expect(screen.getByText("80")).toBeTruthy();
    expect(screen.getByText("Still to assess: Situational judgement, AI interview, Simulation, Manager assessment")).toBeTruthy();
  });
});
