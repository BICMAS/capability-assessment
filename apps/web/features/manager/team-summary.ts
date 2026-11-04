import type { TeamMemberSummary } from "./team-data";

export function memberAttentionLine(member: TeamMemberSummary): string {
  if (!member.levelsOk) {
    return member.levelsError || "Capability levels are unavailable right now.";
  }
  if (member.requiredCount === 0) {
    return "No capability requirements for recorded roles";
  }
  if (member.assessedCount === 0) {
    return "Not yet assessed";
  }
  if (member.belowNames.length === 0) {
    return "No capabilities below the required level";
  }
  const count =
    member.belowNames.length === 1
      ? "1 capability below the required level"
      : `${member.belowNames.length} capabilities below the required level`;
  return `${count}: ${member.belowNames.join(", ")}`;
}

export function needsAttention(member: TeamMemberSummary): boolean {
  if (!member.levelsOk) {
    return true;
  }
  if (member.requiredCount === 0) {
    return false;
  }
  if (member.assessedCount === 0) {
    return true;
  }
  return member.priorityCount > 0 || member.developingCount > 0;
}

export function summariseAttention(members: TeamMemberSummary[]): {
  members: number;
  needsAttention: number;
  withPriorityGaps: number;
  notYetAssessed: number;
  levelFailures: number;
} {
  return {
    members: members.length,
    needsAttention: members.filter(needsAttention).length,
    withPriorityGaps: members.filter((member) => member.levelsOk && member.priorityCount > 0).length,
    notYetAssessed: members.filter((member) => member.levelsOk && member.requiredCount > 0 && member.assessedCount === 0)
      .length,
    levelFailures: members.filter((member) => !member.levelsOk).length,
  };
}

export function uniqueRoleTitles(members: TeamMemberSummary[]): string[] {
  const titles = new Set<string>();
  for (const member of members) {
    for (const role of member.roles) {
      if (role.trim()) {
        titles.add(role);
      }
    }
  }
  return Array.from(titles).sort((left, right) => left.localeCompare(right));
}
