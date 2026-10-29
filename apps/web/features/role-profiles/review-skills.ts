export const reviewCategories = ["technical", "leadership", "domain", "communication", "compliance"] as const;

export type ReviewDraft = {
  displayName: string;
  category: string;
  definition: string;
  requiredProficiency: string;
  priority: string;
  rationale: string;
  criteria: string;
};

export function skillTitle(skill: { skillName: string; displayName?: string }): string {
  const name = skill.displayName?.trim();
  return name ? name : skill.skillName;
}

export function reviewDraft(skill: {
  skillName: string;
  displayName?: string;
  category: string;
  definition: string;
  requiredProficiency: number;
  priority: string;
  rationale: string;
  criteria?: string;
}): ReviewDraft {
  return {
    displayName: skillTitle(skill),
    category: skill.category,
    definition: skill.definition,
    requiredProficiency: String(skill.requiredProficiency),
    priority: skill.priority,
    rationale: skill.rationale,
    criteria: skill.criteria ?? "",
  };
}

export function draftsDiffer(left: ReviewDraft, right: ReviewDraft): boolean {
  return JSON.stringify(left) !== JSON.stringify(right);
}

export function moveItem(ids: string[], fromId: string, toIndex: number): string[] {
  const from = ids.indexOf(fromId);
  if (from < 0 || toIndex < 0 || toIndex >= ids.length || from === toIndex) {
    return ids;
  }
  const next = [...ids];
  const [item] = next.splice(from, 1);
  if (!item) {
    return ids;
  }
  next.splice(toIndex, 0, item);
  return next;
}

export function duplicatePairs(skills: Array<{ id: string; name: string }>): Array<{ keepId: string; removeId: string; name: string }> {
  const groups = new Map<string, Array<{ id: string; name: string }>>();
  for (const skill of skills) {
    const key = skill.name.trim().toLowerCase();
    if (!key) {
      continue;
    }
    groups.set(key, [...(groups.get(key) ?? []), skill]);
  }
  const pairs = [];
  for (const group of groups.values()) {
    const keep = group[0];
    const remove = group[1];
    if (keep && remove) {
      pairs.push({ keepId: keep.id, removeId: remove.id, name: keep.name });
    }
  }
  return pairs;
}

export type GeneratedSkill = {
  skillId: string;
  definition: string;
  requiredProficiency: number;
  priority: string;
  rationale: string;
  criteria: string;
};

export function generatedSkills(draft: unknown): GeneratedSkill[] {
  if (!draft || typeof draft !== "object" || !("skills" in draft) || !Array.isArray(draft.skills)) {
    return [];
  }
  return draft.skills.flatMap((skill) => {
    if (!skill || typeof skill !== "object" || !("skillId" in skill) || typeof skill.skillId !== "string") {
      return [];
    }
    const record = skill as Record<string, unknown>;
    return [
      {
        skillId: record.skillId as string,
        definition: typeof record.definition === "string" ? record.definition : "",
        requiredProficiency: typeof record.requiredProficiency === "number" ? record.requiredProficiency : 0,
        priority: typeof record.priority === "string" ? record.priority : "",
        rationale: typeof record.rationale === "string" ? record.rationale : "",
        criteria: typeof record.criteria === "string" ? record.criteria : "",
      },
    ];
  });
}

export function suggestionGaps(
  skill: { skillId: string; definition: string; requiredProficiency: number; priority: string; rationale: string; criteria?: string },
  draft: unknown,
): string[] {
  const original = generatedSkills(draft).find((item) => item.skillId === skill.skillId);
  if (!original) {
    return [];
  }
  const gaps: string[] = [];
  if (original.definition && original.definition !== skill.definition) {
    gaps.push("definition");
  }
  if (original.requiredProficiency !== skill.requiredProficiency) {
    gaps.push("requiredProficiency");
  }
  if (original.priority && original.priority !== skill.priority) {
    gaps.push("priority");
  }
  if (original.rationale && original.rationale !== skill.rationale) {
    gaps.push("rationale");
  }
  if (original.criteria && original.criteria !== (skill.criteria ?? "")) {
    gaps.push("criteria");
  }
  return gaps;
}
