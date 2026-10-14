// Client methods for contracts/openapi/assessment-plan.yaml operations
// assignPeople, readAssignment, createPlan, readPlan, and replacePlan.

export type Assignment = {
  id: string;
  organisationId: string;
  personId: string;
  frameworkVersionId: string;
  roleId: string;
  assignedByPersonId: string;
  assignedAt: string;
  status: "assigned";
  exceptionReason: string | null;
};

export type AssignPerson = {
  personId: string;
  exceptionReason?: string | null;
};

export type AssignRequest = {
  frameworkVersionId: string;
  roleId: string;
  people: AssignPerson[];
};

export type AssignmentOutcome = "assigned" | "skipped" | "failed";

export type AssignmentResult = {
  personId: string;
  outcome: AssignmentOutcome;
  reason: string;
  assignmentId: string | null;
};

export type AssignmentBatch = {
  results: AssignmentResult[];
};

export type AssessmentItem = {
  id: string;
  prompt: string;
};

export type RubricBand = {
  level: number;
  descriptor: string;
};

export type PlanSkill = {
  id: string;
  requirementId: string;
  capabilityId: string;
  requiredLevel: number;
  recommendedMethod: string;
  rationale: string;
  instructions: string;
  requiresSmeReview: boolean;
  rubricNeedsReview: boolean;
  editedByPersonId?: string;
  assessmentItems: AssessmentItem[];
  rubric: RubricBand[];
};

export type SkillInput = {
  requirementId: string;
  capabilityId: string;
  requiredLevel: number;
  recommendedMethod: string;
  rationale: string;
  instructions: string;
  assessmentItems: { prompt: string }[];
  rubric: { level: number; descriptor: string }[];
};

export type PlanStatus = "draft" | "generated" | "under_review" | "changes_requested" | "approved";

export type Plan = {
  id: string;
  organisationId: string;
  assignmentId: string;
  versionNumber: number;
  status: PlanStatus;
  changeReason?: string;
  approvedByPersonId?: string;
  approvedAt?: string;
  skills: PlanSkill[];
};

export class PlanAlreadyGenerated extends Error {
  constructor() {
    super("generated plan already exists");
    this.name = "PlanAlreadyGenerated";
  }
}

export class VersionNotPublished extends Error {
  constructor() {
    super("framework version is not published");
    this.name = "VersionNotPublished";
  }
}

export class PlanForbidden extends Error {
  constructor() {
    super("assessment plan request was forbidden");
    this.name = "PlanForbidden";
  }
}

export class PlanImmutable extends Error {
  constructor() {
    super("approved plan cannot be changed");
    this.name = "PlanImmutable";
  }
}

export function assessmentPlanUrl(): string {
  const configured = process.env.ASSESSMENT_PLAN_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8087";
}

export async function assignPeople(token: string, body: AssignRequest): Promise<AssignmentBatch> {
  const response = await fetch(`${assessmentPlanUrl()}/assignments`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (response.status === 409) {
    throw new VersionNotPublished();
  }
  if (!response.ok) {
    throw new Error(`assignPeople failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isAssignmentBatch(payload)) {
    throw new Error("assignPeople returned an unexpected payload");
  }
  return payload;
}

export async function readAssignment(args: { token: string; assignmentId: string }): Promise<Assignment | null> {
  const response = await fetch(`${assessmentPlanUrl()}/assignments/${encodeURIComponent(args.assignmentId)}`, {
    headers: { Authorization: `Bearer ${args.token}` },
    cache: "no-store",
  });
  if (response.status === 404) {
    return null;
  }
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (!response.ok) {
    throw new Error(`readAssignment failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isAssignment(payload)) {
    throw new Error("readAssignment returned an unexpected payload");
  }
  return payload;
}

export async function generateAssessmentPlan(token: string, assignmentId: string): Promise<Plan> {
  const response = await fetch(`${assessmentPlanUrl()}/assessment-plans`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ assignmentId }),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (response.status === 409) {
    throw new PlanAlreadyGenerated();
  }
  if (!response.ok) {
    throw new Error(`generateAssessmentPlan failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPlan(payload)) {
    throw new Error("generateAssessmentPlan returned an unexpected payload");
  }
  return payload;
}

export async function createPlan(token: string, body: { assignmentId: string; skills: SkillInput[] }): Promise<Plan> {
  const response = await fetch(`${assessmentPlanUrl()}/plans`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (!response.ok) {
    throw new Error(`createPlan failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPlan(payload)) {
    throw new Error("createPlan returned an unexpected payload");
  }
  return payload;
}

export async function readPlan(args: { token: string; planId: string }): Promise<Plan | null> {
  const response = await fetch(`${assessmentPlanUrl()}/plans/${encodeURIComponent(args.planId)}`, {
    headers: { Authorization: `Bearer ${args.token}` },
    cache: "no-store",
  });
  if (response.status === 404) {
    return null;
  }
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (!response.ok) {
    throw new Error(`readPlan failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPlan(payload)) {
    throw new Error("readPlan returned an unexpected payload");
  }
  return payload;
}

export async function editPlanSkill(
  token: string,
  planId: string,
  skillId: string,
  body: {
    recommendedMethod: string;
    instructions: string;
    assessmentItems: { prompt: string }[];
    rubric: { level: number; descriptor: string }[];
  },
): Promise<Plan> {
  return planRequest(token, "PATCH", `/plans/${encodeURIComponent(planId)}/skills/${encodeURIComponent(skillId)}`, body);
}

export async function submitPlan(token: string, planId: string): Promise<Plan> {
  return planRequest(token, "POST", `/plans/${encodeURIComponent(planId)}/submission`);
}

export async function requestPlanChanges(token: string, planId: string, reason: string): Promise<Plan> {
  return planRequest(token, "POST", `/plans/${encodeURIComponent(planId)}/change-requests`, { reason });
}

export async function approvePlan(token: string, planId: string): Promise<Plan> {
  return planRequest(token, "POST", `/plans/${encodeURIComponent(planId)}/approval`);
}

export async function copyApprovedPlan(token: string, planId: string): Promise<Plan> {
  return planRequest(token, "POST", `/plans/${encodeURIComponent(planId)}/versions`);
}

async function planRequest(token: string, method: string, path: string, body?: unknown): Promise<Plan> {
  const response = await fetch(`${assessmentPlanUrl()}${path}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(body === undefined ? {} : { "Content-Type": "application/json" }),
    },
    body: body === undefined ? undefined : JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (response.status === 409) {
    throw new PlanImmutable();
  }
  if (!response.ok) {
    throw new Error(`${method} ${path} failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPlan(payload)) {
    throw new Error(`${method} ${path} returned an unexpected payload`);
  }
  return payload;
}

export async function replacePlan(
  token: string,
  planId: string,
  body: { status: PlanStatus; skills: SkillInput[] },
): Promise<Plan> {
  const response = await fetch(`${assessmentPlanUrl()}/plans/${encodeURIComponent(planId)}`, {
    method: "PUT",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (response.status === 403) {
    throw new PlanForbidden();
  }
  if (response.status === 409) {
    throw new PlanImmutable();
  }
  if (!response.ok) {
    throw new Error(`replacePlan failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isPlan(payload)) {
    throw new Error("replacePlan returned an unexpected payload");
  }
  return payload;
}

function isAssignmentBatch(value: unknown): value is AssignmentBatch {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const batch = value as Record<string, unknown>;
  return Array.isArray(batch.results) && batch.results.every(isAssignmentResult);
}

function isAssignmentResult(value: unknown): value is AssignmentResult {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const row = value as Record<string, unknown>;
  return (
    typeof row.personId === "string" &&
    (row.outcome === "assigned" || row.outcome === "skipped" || row.outcome === "failed") &&
    typeof row.reason === "string" &&
    (row.assignmentId === null || typeof row.assignmentId === "string")
  );
}

function isAssignment(value: unknown): value is Assignment {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const row = value as Record<string, unknown>;
  return (
    typeof row.id === "string" &&
    typeof row.organisationId === "string" &&
    typeof row.personId === "string" &&
    typeof row.frameworkVersionId === "string" &&
    typeof row.roleId === "string" &&
    typeof row.assignedByPersonId === "string" &&
    typeof row.assignedAt === "string" &&
    row.status === "assigned" &&
    (row.exceptionReason === null || typeof row.exceptionReason === "string")
  );
}

function isPlan(value: unknown): value is Plan {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const row = value as Record<string, unknown>;
  return (
    typeof row.id === "string" &&
    typeof row.organisationId === "string" &&
    typeof row.assignmentId === "string" &&
    typeof row.versionNumber === "number" &&
    isPlanStatus(row.status) &&
    Array.isArray(row.skills) &&
    row.skills.every(isPlanSkill)
  );
}

function isPlanStatus(value: unknown): value is PlanStatus {
  return (
    value === "draft" ||
    value === "generated" ||
    value === "under_review" ||
    value === "changes_requested" ||
    value === "approved"
  );
}

function isPlanSkill(value: unknown): value is PlanSkill {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const skill = value as Record<string, unknown>;
  return (
    typeof skill.id === "string" &&
    typeof skill.requirementId === "string" &&
    typeof skill.capabilityId === "string" &&
    typeof skill.requiredLevel === "number" &&
    typeof skill.recommendedMethod === "string" &&
    typeof skill.rationale === "string" &&
    typeof skill.instructions === "string" &&
    typeof skill.requiresSmeReview === "boolean" &&
    typeof skill.rubricNeedsReview === "boolean" &&
    (skill.editedByPersonId === undefined || typeof skill.editedByPersonId === "string") &&
    Array.isArray(skill.assessmentItems) &&
    skill.assessmentItems.every(isAssessmentItem) &&
    Array.isArray(skill.rubric) &&
    skill.rubric.every(isRubricBand)
  );
}

function isAssessmentItem(value: unknown): value is AssessmentItem {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const item = value as Record<string, unknown>;
  return typeof item.id === "string" && typeof item.prompt === "string";
}

function isRubricBand(value: unknown): value is RubricBand {
  if (typeof value !== "object" || value === null) {
    return false;
  }
  const band = value as Record<string, unknown>;
  return typeof band.level === "number" && typeof band.descriptor === "string";
}
