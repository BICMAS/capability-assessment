export function recordedKind(draft: unknown, skillId: string): string {
  if (!draft || typeof draft !== "object" || !("skills" in draft)) {
    return "";
  }
  const skills = (draft as { skills?: unknown }).skills;
  if (!Array.isArray(skills)) {
    return "";
  }
  for (const skill of skills) {
    if (!skill || typeof skill !== "object" || !("skillId" in skill)) {
      continue;
    }
    if ((skill as { skillId?: unknown }).skillId !== skillId) {
      continue;
    }
    const kind = (skill as { kind?: unknown }).kind;
    return typeof kind === "string" ? kind : "";
  }
  return "";
}
