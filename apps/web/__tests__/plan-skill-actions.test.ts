import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("next/cache", () => ({
  revalidatePath: vi.fn(),
}));

vi.mock("@/lib/session-token", () => ({
  sessionToken: vi.fn(async () => "token"),
}));

vi.mock("@/lib/generated/assessment-plan/plans", () => ({
  PlanImmutable: class PlanImmutable extends Error {
    constructor() {
      super("immutable");
      this.name = "PlanImmutable";
    }
  },
  readPlan: vi.fn(),
  editPlanSkill: vi.fn(),
}));

import { savePlanSkillResult } from "@/features/assessment-plans/actions";
import { editPlanSkill, PlanImmutable, readPlan } from "@/lib/generated/assessment-plan/plans";

function form(entries: Record<string, string>): FormData {
  const data = new FormData();
  for (const [key, value] of Object.entries(entries)) {
    data.set(key, value);
  }
  return data;
}

const base = {
  assignmentId: "a1",
  planId: "p1",
  skillId: "s1",
  recommendedMethod: "Interview",
  instructions: "Ask about negotiation.",
  "prompt.0": "Describe a deal.",
  "level.0": "4",
  "descriptor.0": "Reaches agreement",
};

describe("savePlanSkillResult", () => {
  beforeEach(() => {
    vi.mocked(readPlan).mockReset();
    vi.mocked(editPlanSkill).mockReset();
  });

  it("returns validation failure for empty method", async () => {
    const result = await savePlanSkillResult(form({ ...base, recommendedMethod: "" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("validation");
    }
  });

  it("refuses stale edits when expected version differs", async () => {
    vi.mocked(readPlan).mockResolvedValue({
      id: "p1",
      assignmentId: "a1",
      versionNumber: 3,
      status: "generated",
      skills: [],
    } as never);
    const result = await savePlanSkillResult(form({ ...base, expectedVersionNumber: "2" }));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("conflict");
    }
    expect(editPlanSkill).not.toHaveBeenCalled();
  });

  it("saves when versions match", async () => {
    vi.mocked(readPlan).mockResolvedValue({
      id: "p1",
      assignmentId: "a1",
      versionNumber: 2,
      status: "generated",
      skills: [],
    } as never);
    vi.mocked(editPlanSkill).mockResolvedValue({ id: "p1" } as never);
    const result = await savePlanSkillResult(form({ ...base, expectedVersionNumber: "2" }));
    expect(result.ok).toBe(true);
    if (result.ok) {
      expect(result.data?.planId).toBe("p1");
    }
  });

  it("maps immutable plans to conflict", async () => {
    vi.mocked(editPlanSkill).mockRejectedValue(new PlanImmutable());
    const result = await savePlanSkillResult(form(base));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("conflict");
      expect(result.message).toContain("approved");
    }
  });

  it("maps backend outage to unavailable", async () => {
    vi.mocked(editPlanSkill).mockRejectedValue(new TypeError("fetch failed"));
    const result = await savePlanSkillResult(form(base));
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.code).toBe("unavailable");
    }
  });
});
