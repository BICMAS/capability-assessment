import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("@/features/role-profiles/actions", () => ({
  answerClarificationForm: vi.fn(),
  startClarificationForm: vi.fn(),
}));

import { ClarificationPanel } from "@/features/role-profiles/clarification-panel";
import type { ClarificationSession } from "@/lib/role-profile";

afterEach(() => {
  cleanup();
});

function session(authoredBy: string): ClarificationSession {
  return {
    profileVersionId: "version-1",
    contextVersion: 1,
    contextHash: "hash",
    round: 1,
    unresolved: ["businessOutcomes"],
    complete: false,
    questions: [
      {
        id: "question-1",
        profileVersionId: "version-1",
        contextVersion: 1,
        round: 1,
        field: "businessOutcomes",
        question: "What are the three to five most important business outcomes of this role (Own the regional sales plan)?",
        critical: true,
        answer: "",
        unresolved: true,
        authoredBy,
        reason: "This was not supplied. The skills profile should not guess it.",
        askedActorId: "ada",
        answeredActorId: "",
        askedAt: "2026-10-29T00:00:00Z",
        answeredAt: null,
      },
    ],
    snapshot: {
      purpose: "Own the regional sales plan",
      responsibilities: "",
      businessOutcomes: "",
      decisionScope: "",
      reportingLevel: "",
      industry: "",
      region: "",
      teamSize: null,
      sources: [],
    },
  };
}

describe("clarification panel", () => {
  it("shows why a question was asked and does not present a failed model as live AI", () => {
    render(
      <ClarificationPanel
        profileId="profile-1"
        versionId="version-1"
        editable
        session={{ ok: true, value: session("service_after_model_error") }}
      />,
    );
    expect(screen.getByText(/three to five most important business outcomes/)).toBeTruthy();
    expect(screen.getByText(/skills profile should not guess it/)).toBeTruthy();
    expect(screen.getByText(/model could not be reached/)).toBeTruthy();
    expect(screen.queryByText(/score/i)).toBeNull();
  });

  it("offers the reporting level choices when the question has options", () => {
    const value = session("service");
    value.questions[0] = {
      ...value.questions[0],
      field: "reportingLevel",
      question: "Which level of responsibility applies to this role?",
      options: ["individual", "manager"],
    };
    render(<ClarificationPanel profileId="profile-1" versionId="version-1" editable session={{ ok: true, value }} />);
    expect(screen.getByRole("combobox")).toBeTruthy();
    expect(screen.getByRole("option", { name: "manager" })).toBeTruthy();
  });
});
