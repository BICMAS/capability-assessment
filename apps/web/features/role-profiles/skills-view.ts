import type { RoleProfileSkill } from "@/lib/role-profile";

import { recordedKind } from "./kind";

export type SkillKind = "technical" | "behavioural" | "knowledge";

export const proficiencyLabels: Record<number, string> = {
  0: "Not set",
  1: "Basic",
  2: "Intermediate",
  3: "Advanced",
  4: "Expert",
};

export function proficiencyLabel(level: number): string {
  return proficiencyLabels[level] ?? "Not set";
}

export function skillKind(category: string, recorded: string): SkillKind {
  if (recorded === "technical" || recorded === "behavioural" || recorded === "knowledge") {
    return recorded;
  }
  if (category === "leadership" || category === "communication") {
    return "behavioural";
  }
  if (category === "domain" || category === "compliance") {
    return "knowledge";
  }
  return "technical";
}

export type ListedSkill = RoleProfileSkill & { kind: SkillKind };

export function listSkills(skills: RoleProfileSkill[], draft: unknown): ListedSkill[] {
  return [...skills]
    .sort((left, right) => left.position - right.position)
    .map((skill) => ({ ...skill, kind: skillKind(skill.category, recordedKind(draft, skill.skillId)) }));
}

export function kindTotals(skills: ListedSkill[]): Record<SkillKind, number> {
  return {
    technical: skills.filter((skill) => skill.kind === "technical").length,
    behavioural: skills.filter((skill) => skill.kind === "behavioural").length,
    knowledge: skills.filter((skill) => skill.kind === "knowledge").length,
  };
}

export function filterSkills(
  skills: ListedSkill[],
  query: { kind: SkillKind; search: string; level: string; provenance: string },
): ListedSkill[] {
  const search = query.search.trim().toLowerCase();
  return skills.filter((skill) => {
    if (skill.kind !== query.kind) {
      return false;
    }
    if (query.level !== "" && String(skill.requiredProficiency) !== query.level) {
      return false;
    }
    if (query.provenance !== "" && skill.provenance !== query.provenance) {
      return false;
    }
    if (search === "") {
      return true;
    }
    return skill.skillName.toLowerCase().includes(search) || skill.definition.toLowerCase().includes(search);
  });
}

export function sourceLabel(provenance: string): string {
  switch (provenance) {
    case "ai_suggested":
      return "AI suggestion";
    case "hr_adjusted":
      return "Adjusted by a person";
    case "imported":
      return "Imported";
    case "manually_added":
      return "Added by a person";
    default:
      return provenance || "Not recorded";
  }
}

const reportingLabels: Record<string, string> = {
  individual: "Individual contributor",
  team_lead: "Team lead",
  manager: "Manager",
  director: "Director",
  executive: "Executive",
};

export function reportingLabel(level: string): string {
  return reportingLabels[level] ?? (level || "Not recorded");
}
