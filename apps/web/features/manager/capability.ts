import type { FrameworkVersion } from "@/lib/capability-framework";
import type { Explanation } from "@/lib/generated/assessment-gap/levels";

export type TeamCapability = {
  name: string;
  capabilityCode: string;
  frameworkVersionId: string;
  requiredLevel: number;
  explanation: Explanation | null;
};

export function gapBand(gap: string): "strength" | "developing" | "priority" {
  if (gap.startsWith("-") || gap === "0.0") {
    return "strength";
  }
  if (gap.startsWith("0.")) {
    return "developing";
  }
  return "priority";
}

export function capabilitiesFor(
  framework: FrameworkVersion,
  roleTitles: Set<string>,
  explanations: Map<string, Explanation | null>,
): TeamCapability[] {
  return framework.requirements.flatMap((requirement) => {
    const role = framework.roles.find((item) => item.id === requirement.roleId);
    const capability = framework.capabilities.find((item) => item.id === requirement.capabilityId);
    if (!role || !capability || !roleTitles.has(role.name)) {
      return [];
    }
    return [
      {
        name: capability.name,
        capabilityCode: capability.id,
        frameworkVersionId: framework.id,
        requiredLevel: requirement.requiredLevel,
        explanation: explanations.get(capability.id) ?? null,
      },
    ];
  });
}

export function teamSummary(capabilities: TeamCapability[]): string {
  const assessed = capabilities.some((capability) => capability.explanation !== null);
  if (!assessed) {
    return "Not yet assessed";
  }
  const below = capabilities.filter((capability) => {
    if (!capability.explanation) {
      return false;
    }
    const band = gapBand(capability.explanation.gap);
    return band === "developing" || band === "priority";
  });
  if (below.length === 0) {
    return "No capabilities below the required level";
  }
  const count = below.length === 1 ? "1 capability below the required level" : `${below.length} capabilities below the required level`;
  return `${count}: ${below.map((capability) => capability.name).join(", ")}`;
}
