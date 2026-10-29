import { developmentUrl } from "@/lib/generated/development/pathways";

export type LearningResource = {
  id: string;
  title: string;
  skillId: string;
  targetLevel: number;
  reference: string;
  rationale: string;
};

export type LearningRecommendation = {
  id: string;
  personId: string;
  skillId: string;
  requiredLevel: number;
  intendedLevel: number;
  activity: string;
  rationale: string;
  reference: string;
  status: string;
};

export type PracticeReply = {
  id?: string;
  personId: string;
  scenarioKey: string;
  message: string;
  feedback: string;
  recorded: boolean;
  voice: string;
};

const scenarios = [
  ["sales_negotiation", "Sales and distributor negotiations"],
  ["customer_relationship", "Customer relationship management"],
  ["team_leadership", "Team leadership"],
  ["conflict_resolution", "Conflict resolution"],
  ["performance_conversation", "Performance conversations"],
] as const;

export const practiceScenarios = scenarios;

export async function listLearningResources(token: string): Promise<LearningResource[]> {
  return readJSON(token, "/learning-resources", isResourceList);
}

export async function listLearningRecommendations(token: string, personId: string): Promise<LearningRecommendation[]> {
  const query = personId ? `?personId=${encodeURIComponent(personId)}` : "";
  return readJSON(token, `/learning-recommendations${query}`, isRecommendationList);
}

export async function createLearningResource(
  token: string,
  body: { title: string; skillId: string; targetLevel: number; reference: string; rationale: string },
): Promise<void> {
  await sendJSON(token, "/learning-resources", body, 201);
}

export async function createLearningRecommendation(
  token: string,
  body: { personId: string; capabilityId: string; targetLevel: number; state: string },
): Promise<void> {
  await sendJSON(token, "/learning-recommendations", body, 201);
}

export async function createPracticeSession(
  token: string,
  body: { personId: string; scenarioKey: string; message: string; record: boolean },
): Promise<PracticeReply> {
  return sendJSON(token, "/practice-sessions", body, 201, isPracticeReply);
}

async function readJSON<T>(token: string, path: string, guard: (value: unknown) => value is T): Promise<T> {
  const response = await fetch(`${developmentUrl()}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`development request failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!guard(payload)) {
    throw new Error("development request returned an unexpected payload");
  }
  return payload;
}

async function sendJSON<T>(token: string, path: string, body: unknown, status: number, guard?: (value: unknown) => value is T): Promise<T> {
  const response = await fetch(`${developmentUrl()}${path}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status !== status) {
    throw new Error(`development request failed with status ${response.status}`);
  }
  if (!guard) {
    return undefined as T;
  }
  const payload: unknown = await response.json();
  if (!guard(payload)) {
    throw new Error("development request returned an unexpected payload");
  }
  return payload;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isResource(value: unknown): value is LearningResource {
  return isRecord(value) && typeof value.id === "string" && typeof value.title === "string" && typeof value.skillId === "string" && typeof value.targetLevel === "number" && typeof value.reference === "string" && typeof value.rationale === "string";
}

function isResourceList(value: unknown): value is LearningResource[] {
  return Array.isArray(value) && value.every(isResource);
}

function isRecommendation(value: unknown): value is LearningRecommendation {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.personId === "string" &&
    typeof value.skillId === "string" &&
    typeof value.requiredLevel === "number" &&
    typeof value.intendedLevel === "number" &&
    typeof value.activity === "string" &&
    typeof value.rationale === "string" &&
    typeof value.reference === "string" &&
    typeof value.status === "string"
  );
}

function isRecommendationList(value: unknown): value is LearningRecommendation[] {
  return Array.isArray(value) && value.every(isRecommendation);
}

function isPracticeReply(value: unknown): value is PracticeReply {
  return isRecord(value) && typeof value.personId === "string" && typeof value.scenarioKey === "string" && typeof value.feedback === "string" && typeof value.recorded === "boolean" && typeof value.voice === "string";
}
