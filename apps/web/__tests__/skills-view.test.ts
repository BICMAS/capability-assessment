import { describe, expect, it } from "vitest";

import { filterSkills, kindTotals, listSkills, proficiencyLabel } from "@/features/role-profiles/skills-view";
import type { RoleProfileSkill } from "@/lib/role-profile";

const skills: RoleProfileSkill[] = [
  {
    id: "assignment-1",
    skillId: "skill-1",
    skillName: "Territory planning",
    category: "technical",
    definition: "Plans coverage",
    requiredProficiency: 3,
    priority: "required",
    rationale: "The role owns coverage",
    provenance: "ai_suggested",
    position: 1,
  },
  {
    id: "assignment-2",
    skillId: "skill-2",
    skillName: "Coaching",
    category: "leadership",
    definition: "Coaches the team",
    requiredProficiency: 2,
    priority: "preferred",
    rationale: "The role leads a team",
    provenance: "manually_added",
    position: 2,
  },
  {
    id: "assignment-3",
    skillId: "skill-3",
    skillName: "Pricing policy",
    category: "domain",
    definition: "Knows the pricing rules",
    requiredProficiency: 4,
    priority: "required",
    rationale: "The role approves discounts",
    provenance: "ai_suggested",
    position: 3,
  },
];

const draft = {
  skills: [
    { skillId: "skill-1", kind: "technical" },
    { skillId: "skill-2", kind: "behavioural" },
    { skillId: "skill-3", kind: "knowledge" },
  ],
};

describe("skills profile view", () => {
  it("filters by category, search, level, and source", () => {
    const listed = listSkills(skills, draft);
    expect(kindTotals(listed)).toEqual({ technical: 1, behavioural: 1, knowledge: 1 });
    expect(filterSkills(listed, { kind: "technical", search: "territory", level: "3", provenance: "ai_suggested" }).map((skill) => skill.skillName)).toEqual([
      "Territory planning",
    ]);
    expect(filterSkills(listed, { kind: "behavioural", search: "", level: "", provenance: "ai_suggested" })).toEqual([]);
    expect(filterSkills(listed, { kind: "knowledge", search: "pricing", level: "4", provenance: "" }).map((skill) => skill.id)).toEqual(["assignment-3"]);
    expect(proficiencyLabel(2)).toBe("Intermediate");
    expect(proficiencyLabel(0)).toBe("Not set");
  });

  it("uses the canonical category when the draft has no kind", () => {
    const listed = listSkills(skills, {});
    expect(listed.map((skill) => skill.kind)).toEqual(["technical", "behavioural", "knowledge"]);
  });
});
