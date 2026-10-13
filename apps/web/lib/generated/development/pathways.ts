// Client methods for contracts/openapi/development.yaml operation readPathway.

export type PlanItem = {
  id: string;
  stage: "learn" | "practise" | "apply" | "validate";
  title: string;
  status: "ready" | "complete" | "upcoming";
  deepLink: string | null;
};

export type Pathway = {
  id: string;
  organisationId: string;
  personId: string;
  capabilityId: string;
  targetLevel: number;
  items: PlanItem[];
};

export function developmentUrl(): string {
  const configured = process.env.DEVELOPMENT_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8086";
}

export async function readPathway(args: {
  token: string;
  personId: string;
  capabilityId: string;
}): Promise<Pathway | null> {
  const params = new URLSearchParams({
    personId: args.personId,
    capabilityId: args.capabilityId,
  });
  const response = await fetch(`${developmentUrl()}/pathways?${params.toString()}`, {
    headers: { Authorization: `Bearer ${args.token}` },
    cache: "no-store",
  });
  if (response.status === 404) {
    return null;
  }
  if (!response.ok) {
    throw new Error(`readPathway failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPathway(payload)) {
    throw new Error("readPathway returned an unexpected payload");
  }
  return payload;
}

function isPathway(value: unknown): value is Pathway {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const pathway = value as Record<string, unknown>;
  return (
    typeof pathway.id === "string" &&
    typeof pathway.organisationId === "string" &&
    typeof pathway.personId === "string" &&
    typeof pathway.capabilityId === "string" &&
    typeof pathway.targetLevel === "number" &&
    Array.isArray(pathway.items) &&
    pathway.items.every(isPlanItem)
  );
}

function isPlanItem(value: unknown): value is PlanItem {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  return (
    typeof item.id === "string" &&
    (item.stage === "learn" || item.stage === "practise" || item.stage === "apply" || item.stage === "validate") &&
    typeof item.title === "string" &&
    (item.status === "ready" || item.status === "complete" || item.status === "upcoming") &&
    (item.deepLink === null || typeof item.deepLink === "string")
  );
}
