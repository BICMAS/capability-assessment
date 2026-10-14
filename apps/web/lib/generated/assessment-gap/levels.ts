// Client methods for contracts/openapi/assessment-gap.yaml operations readLevel and recordAttempt.

export type Explanation = {
  id: string;
  organisationId: string;
  personId: string;
  frameworkVersionId: string;
  capabilityCode: string;
  knowledgePercentage: number | null;
  situationalJudgementPercentage: number | null;
  aiInterviewPercentage: number | null;
  simulationPercentage: number | null;
  managerAssessmentPercentage: number | null;
  selfAssessmentPercentage: number | null;
  mean: string;
  divisor: number;
  requiredLevel: string;
  currentLevel: string;
  gap: string;
  status: string;
};

export type RecordAttempt = {
  personId: string;
  frameworkVersionId: string;
  capabilityCode: string;
  source: "knowledge" | "situational_judgement" | "self_assessment" | "manager_assessment" | "ai_interview";
  percentage: number;
  evidenceIds: string[];
  requiredLevel: number;
};

export class LevelForbidden extends Error {
  constructor() {
    super("level request was forbidden");
    this.name = "LevelForbidden";
  }
}

export function assessmentGapUrl(): string {
  const configured = process.env.ASSESSMENT_GAP_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8085";
}

export async function readLevel(args: {
  token: string;
  personId: string;
  frameworkVersionId: string;
  capabilityCode: string;
}): Promise<Explanation | null> {
  const params = new URLSearchParams({
    personId: args.personId,
    frameworkVersionId: args.frameworkVersionId,
    capabilityCode: args.capabilityCode,
  });
  const response = await fetch(`${assessmentGapUrl()}/levels?${params.toString()}`, {
    headers: { Authorization: `Bearer ${args.token}` },
    cache: "no-store",
  });
  if (response.status === 404) {
    return null;
  }
  if (response.status === 403) {
    throw new LevelForbidden();
  }
  if (!response.ok) {
    throw new Error(`readLevel failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isExplanation(payload)) {
    throw new Error("readLevel returned an unexpected payload");
  }
  return payload;
}

export async function recordAttempt(token: string, body: RecordAttempt): Promise<Explanation> {
  const response = await fetch(`${assessmentGapUrl()}/attempts`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new LevelForbidden();
  }
  if (!response.ok) {
    throw new Error(`recordAttempt failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isExplanation(payload)) {
    throw new Error("recordAttempt returned an unexpected payload");
  }
  return payload;
}

function isExplanation(value: unknown): value is Explanation {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  return (
    typeof item.id === "string" &&
    typeof item.organisationId === "string" &&
    typeof item.personId === "string" &&
    typeof item.frameworkVersionId === "string" &&
    typeof item.capabilityCode === "string" &&
    isPercentage(item.knowledgePercentage) &&
    isPercentage(item.situationalJudgementPercentage) &&
    isPercentage(item.aiInterviewPercentage) &&
    isPercentage(item.simulationPercentage) &&
    isPercentage(item.managerAssessmentPercentage) &&
    isPercentage(item.selfAssessmentPercentage) &&
    typeof item.mean === "string" &&
    typeof item.divisor === "number" &&
    typeof item.requiredLevel === "string" &&
    typeof item.currentLevel === "string" &&
    typeof item.gap === "string" &&
    typeof item.status === "string"
  );
}

function isPercentage(value: unknown): value is number | null {
  return value === null || (typeof value === "number" && Number.isInteger(value));
}
