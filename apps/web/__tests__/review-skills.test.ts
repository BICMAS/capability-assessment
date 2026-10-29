import { describe, expect, it } from "vitest";

import { duplicatePairs, generatedSkills, moveItem, suggestionGaps } from "@/features/role-profiles/review-skills";

describe("skill review helpers", () => {
  it("reorders, finds duplicate names, and compares a skill with the generated suggestion", () => {
    expect(moveItem(["a", "b", "c"], "c", 0)).toEqual(["c", "a", "b"]);
    expect(moveItem(["a", "b"], "a", 0)).toEqual(["a", "b"]);
    expect(duplicatePairs([
      { id: "one", name: "Territory planning" },
      { id: "two", name: "territory planning" },
      { id: "three", name: "Coaching" },
    ])).toEqual([{ keepId: "one", removeId: "two", name: "Territory planning" }]);
    const draft = {
      skills: [{ skillId: "skill-1", definition: "Plans coverage", requiredProficiency: 3, priority: "required", rationale: "Owns coverage", criteria: "Review a plan" }],
    };
    expect(generatedSkills(draft)).toHaveLength(1);
    expect(
      suggestionGaps(
        { skillId: "skill-1", definition: "Plans the local accounts", requiredProficiency: 3, priority: "required", rationale: "Owns coverage", criteria: "Review a plan" },
        draft,
      ),
    ).toEqual(["definition"]);
    expect(suggestionGaps({ skillId: "skill-2", definition: "Added", requiredProficiency: 1, priority: "optional", rationale: "New" }, draft)).toEqual([]);
  });
});