import type { FrameworkVersion } from "@/lib/capability-framework";
import { readLevel } from "@/lib/generated/assessment-gap/levels";
import type { DirectoryPerson } from "@/lib/org-directory";
import { readPerson } from "@/lib/org-directory";
import { settle } from "@/lib/safe";

import { capabilitiesFor, gapBand } from "@/features/manager/capability";

export type TeamMemberSummary = {
  personId: string;
  name: string;
  departmentName: string;
  roles: string[];
  requiredCount: number;
  assessedCount: number;
  priorityCount: number;
  developingCount: number;
  priorityNames: string[];
  belowNames: string[];
  levelsOk: boolean;
  levelsError: string;
};

export async function loadTeamMemberSummaries(
  token: string,
  people: DirectoryPerson[],
  frameworks: FrameworkVersion[],
): Promise<TeamMemberSummary[]> {
  return Promise.all(people.map((person) => loadMemberSummary(token, person, frameworks)));
}

async function loadMemberSummary(
  token: string,
  person: DirectoryPerson,
  frameworks: FrameworkVersion[],
): Promise<TeamMemberSummary> {
  const profile = await settle(readPerson(token, person.id), "Profile");
  if (!profile.ok) {
    return {
      personId: person.id,
      name: person.name,
      departmentName: person.departmentName,
      roles: [],
      requiredCount: 0,
      assessedCount: 0,
      priorityCount: 0,
      developingCount: 0,
      priorityNames: [],
      belowNames: [],
      levelsOk: false,
      levelsError: profile.error,
    };
  }

  const roles = profile.value.roles.map((role) => role.title);
  const titles = new Set(roles);
  const required = frameworks.flatMap((framework) => capabilitiesFor(framework, titles, new Map()));
  const unique = uniqueRequirements(required);

  const levels = await Promise.all(
    unique.map((capability) =>
      settle(
        readLevel({
          token,
          personId: person.id,
          frameworkVersionId: capability.frameworkVersionId,
          capabilityCode: capability.capabilityCode,
        }),
        "Capability level",
      ),
    ),
  );

  if (levels.some((level) => !level.ok)) {
    return {
      personId: person.id,
      name: person.name,
      departmentName: person.departmentName,
      roles,
      requiredCount: unique.length,
      assessedCount: 0,
      priorityCount: 0,
      developingCount: 0,
      priorityNames: [],
      belowNames: [],
      levelsOk: false,
      levelsError: "Capability levels are unavailable right now.",
    };
  }

  let assessedCount = 0;
  let priorityCount = 0;
  let developingCount = 0;
  const priorityNames: string[] = [];
  const belowNames: string[] = [];

  unique.forEach((capability, index) => {
    const level = levels[index];
    if (!level || !level.ok || !level.value) {
      return;
    }
    assessedCount += 1;
    const band = gapBand(level.value.gap);
    if (band === "priority") {
      priorityCount += 1;
      priorityNames.push(capability.name);
      belowNames.push(capability.name);
    } else if (band === "developing") {
      developingCount += 1;
      belowNames.push(capability.name);
    }
  });

  return {
    personId: person.id,
    name: person.name,
    departmentName: person.departmentName,
    roles,
    requiredCount: unique.length,
    assessedCount,
    priorityCount,
    developingCount,
    priorityNames,
    belowNames,
    levelsOk: true,
    levelsError: "",
  };
}

function uniqueRequirements(
  items: ReturnType<typeof capabilitiesFor>,
): ReturnType<typeof capabilitiesFor> {
  const seen = new Set<string>();
  const unique: ReturnType<typeof capabilitiesFor> = [];
  for (const item of items) {
    const key = `${item.frameworkVersionId}:${item.capabilityCode}`;
    if (seen.has(key)) {
      continue;
    }
    seen.add(key);
    unique.push(item);
  }
  return unique;
}
