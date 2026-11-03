import type { FrameworkVersion } from "@/lib/capability-framework";
import { readLevel, type Explanation } from "@/lib/generated/assessment-gap/levels";
import { readPathway, type Pathway } from "@/lib/generated/development/pathways";
import { settle, type Settled } from "@/lib/safe";

export type GapBand = "strength" | "developing" | "priority";

export type CapabilityRow = {
  id: string;
  name: string;
  frameworkVersionId: string;
  frameworkName: string;
  versionNumber: number;
  capabilityCode: string;
  requiredLevel: number;
  explanation: Explanation | null;
  pathway: Pathway | null;
  level: Settled<Explanation | null>;
  pathwayResult: Settled<Pathway | null>;
};

export type CapabilityWorkbenchData = {
  rows: CapabilityRow[];
  levelFailures: number;
  pathwayFailures: number;
};

export async function loadCapabilityRows(
  token: string,
  personId: string,
  roleTitles: Set<string>,
  frameworks: FrameworkVersion[],
): Promise<CapabilityWorkbenchData> {
  const rows = await Promise.all(
    frameworks.flatMap((framework) => {
      const requirements = framework.requirements.filter((requirement) => {
        const role = framework.roles.find((item) => item.id === requirement.roleId);
        return role !== undefined && roleTitles.has(role.name);
      });
      return requirements.map(async (requirement) => {
        const capability = framework.capabilities.find((item) => item.id === requirement.capabilityId);
        const [level, pathwayResult] = await Promise.all([
          settle(
            readLevel({
              token,
              personId,
              frameworkVersionId: framework.id,
              capabilityCode: requirement.capabilityId,
            }),
            "Capability level",
          ),
          settle(readPathway({ token, personId, capabilityId: requirement.capabilityId }), "Pathway"),
        ]);
        return {
          id: requirement.id,
          name: capability?.name ?? "Capability",
          frameworkVersionId: framework.id,
          frameworkName: framework.frameworkName,
          versionNumber: framework.versionNumber,
          capabilityCode: requirement.capabilityId,
          requiredLevel: requirement.requiredLevel,
          explanation: level.ok ? level.value : null,
          pathway: pathwayResult.ok ? pathwayResult.value : null,
          level,
          pathwayResult,
        } satisfies CapabilityRow;
      });
    }),
  );

  const ordered = [...rows].sort((left, right) => bandRank(left) - bandRank(right));
  return {
    rows: ordered,
    levelFailures: ordered.filter((row) => !row.level.ok).length,
    pathwayFailures: ordered.filter((row) => !row.pathwayResult.ok).length,
  };
}

export function gapBand(gap: string): GapBand {
  if (gap.startsWith("-") || gap === "0.0") {
    return "strength";
  }
  if (gap.startsWith("0.")) {
    return "developing";
  }
  return "priority";
}

export function needsAttention(row: CapabilityRow): boolean {
  if (!row.level.ok || !row.explanation) {
    return true;
  }
  return gapBand(row.explanation.gap) !== "strength";
}

export function rowBand(row: CapabilityRow): GapBand | "unassessed" | "unavailable" {
  if (!row.level.ok) {
    return "unavailable";
  }
  if (!row.explanation) {
    return "unassessed";
  }
  return gapBand(row.explanation.gap);
}

export function groupByFramework(rows: CapabilityRow[]): {
  key: string;
  frameworkName: string;
  versionNumber: number;
  rows: CapabilityRow[];
}[] {
  const map = new Map<string, { key: string; frameworkName: string; versionNumber: number; rows: CapabilityRow[] }>();
  for (const row of rows) {
    const current = map.get(row.frameworkVersionId);
    if (current) {
      current.rows.push(row);
    } else {
      map.set(row.frameworkVersionId, {
        key: row.frameworkVersionId,
        frameworkName: row.frameworkName,
        versionNumber: row.versionNumber,
        rows: [row],
      });
    }
  }
  return Array.from(map.values());
}

export function bandSections(rows: CapabilityRow[]): {
  key: GapBand | "unassessed" | "unavailable";
  title: string;
  titleClass: string;
  rows: CapabilityRow[];
}[] {
  const order: Array<GapBand | "unassessed" | "unavailable"> = [
    "priority",
    "developing",
    "strength",
    "unassessed",
    "unavailable",
  ];
  return order
    .map((key) => {
      const matched = rows.filter((row) => rowBand(row) === key);
      return {
        key,
        title: bandSectionTitle(key),
        titleClass: bandSectionClass(key),
        rows: matched,
      };
    })
    .filter((section) => section.rows.length > 0);
}

export function bandLabel(band: GapBand | "unassessed" | "unavailable"): string {
  switch (band) {
    case "strength":
      return "Strength";
    case "developing":
      return "Developing";
    case "priority":
      return "Priority gap";
    case "unassessed":
      return "Not yet assessed";
    case "unavailable":
      return "Unavailable";
  }
}

export function bandChipClass(band: GapBand | "unassessed" | "unavailable"): string {
  if (band === "strength") {
    return "ds-status employee-band-chip is-strength";
  }
  if (band === "priority") {
    return "ds-status ds-status-attention employee-band-chip is-priority";
  }
  if (band === "developing") {
    return "ds-status ds-status-neutral employee-band-chip is-developing";
  }
  if (band === "unavailable") {
    return "ds-status ds-status-attention employee-band-chip";
  }
  return "ds-status ds-status-neutral employee-band-chip";
}

function bandSectionTitle(key: GapBand | "unassessed" | "unavailable"): string {
  switch (key) {
    case "strength":
      return "Strengths";
    case "developing":
      return "Developing areas";
    case "priority":
      return "Priority gaps";
    case "unassessed":
      return "Not yet assessed";
    case "unavailable":
      return "Levels unavailable";
  }
}

function bandSectionClass(key: GapBand | "unassessed" | "unavailable"): string {
  if (key === "strength") {
    return "band-strength";
  }
  if (key === "priority") {
    return "band-priority";
  }
  return "";
}

function bandRank(row: CapabilityRow): number {
  switch (rowBand(row)) {
    case "priority":
      return 0;
    case "developing":
      return 1;
    case "unassessed":
      return 2;
    case "unavailable":
      return 3;
    case "strength":
      return 4;
  }
}
