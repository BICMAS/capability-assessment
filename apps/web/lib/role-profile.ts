import { capabilityFrameworkUrl } from "@/lib/capability-framework";
import { fail, type ActionErr } from "@/lib/action-result";
import { USER_MESSAGES } from "@/lib/errors";

export class RoleProfileApiError extends Error {
  readonly status: number;
  readonly serviceCode: string;

  constructor(status: number, serviceCode: string) {
    super(`role profile failed with status ${status}`);
    this.name = "RoleProfileApiError";
    this.status = status;
    this.serviceCode = serviceCode;
  }
}

export type ProfileRole = { id: string; organisationId: string; name: string };

export type CanonicalSkill = {
  id: string;
  organisationId: string;
  name: string;
  category: string;
  definition: string;
};

export type RoleProfileSource = { id: string; title: string; referenceUri: string };

export type RoleProfileSkill = {
  id: string;
  skillId: string;
  skillName: string;
  category: string;
  definition: string;
  requiredProficiency: number;
  priority: string;
  rationale: string;
  provenance: string;
  position: number;
};

export type RoleProfileVersion = {
  id: string;
  profileId: string;
  organisationId: string;
  versionNumber: number;
  revision: number;
  status: string;
  purpose: string;
  responsibilities: string;
  businessOutcomes: string;
  decisionScope: string;
  industry: string;
  region: string;
  teamSize: number | null;
  reportingLevel: string;
  frameworkVersionId: string;
  sources: RoleProfileSource[];
  skills: RoleProfileSkill[];
  ai: { model: string; sourceContextVersion: string; generatedAt: string | null; draft: unknown };
  createdAt: string;
};

export type RoleProfile = {
  id: string;
  organisationId: string;
  roleId: string;
  roleName: string;
  createdByPersonId: string;
  createdAt: string;
  version: RoleProfileVersion;
};

export type ClarificationQuestion = {
  id: string;
  profileVersionId: string;
  contextVersion: number;
  round: number;
  field: string;
  question: string;
  critical: boolean;
  answer: string;
  unresolved: boolean;
  askedActorId: string;
  answeredActorId: string;
  askedAt: string;
  answeredAt: string | null;
};

export type ClarificationSession = {
  profileVersionId: string;
  contextVersion: number;
  contextHash: string;
  round: number;
  questions: ClarificationQuestion[];
  unresolved: string[];
  complete: boolean;
  snapshot: {
    purpose: string;
    responsibilities: string;
    businessOutcomes: string;
    decisionScope: string;
    reportingLevel: string;
    industry: string;
    region: string;
    teamSize: number | null;
    sources: Array<{ title: string; referenceUri: string }>;
  };
};

export type GenerationJob = {
  id: string;
  organisationId: string;
  profileVersionId: string;
  contextVersion: number;
  contextHash: string;
  idempotencyKey: string;
  status: string;
  modelId: string;
  promptVersion: string;
  inputHash: string;
  outputHash: string;
  latencyMs: number;
  promptTokens: number;
  completionTokens: number;
  attempts: number;
  lastError: string;
  unsupported: Array<{ skillId: string; kind: string; reason: string }>;
  uncertainty: string[];
  cancelRequested: boolean;
  createdBy: string;
  createdAt: string;
};

export type RoleProfileComment = {
  id: string;
  profileVersionId: string;
  authorPersonId: string;
  body: string;
  createdAt: string;
};

export type RoleProfileHistory = {
  profileId: string;
  events: Array<{ id: string; profileVersionId: string; actorPersonId: string; action: string; occurredAt: string }>;
  comments: RoleProfileComment[];
};

export type RoleProfileComparison = {
  from: RoleProfileVersion;
  to: RoleProfileVersion;
  added: string[];
  removed: string[];
  changed: Array<{ skillId: string; fromProficiency: number; toProficiency: number; fromPriority: string; toPriority: string }>;
};

export type ContextInput = {
  expectedRevision: number;
  purpose: string;
  responsibilities: string;
  businessOutcomes: string;
  decisionScope: string;
  industry: string;
  region: string;
  teamSize: number | null;
  reportingLevel: string;
  sources: Array<{ title: string; referenceUri: string }>;
};

export function roleProfileFailure(error: unknown): ActionErr | null {
  if (!(error instanceof RoleProfileApiError)) {
    return null;
  }
  if (error.status === 401) {
    return fail("unauthenticated", USER_MESSAGES.unauthenticated);
  }
  if (error.status === 403) {
    return fail("forbidden", USER_MESSAGES.forbidden);
  }
  if (error.status === 404) {
    return fail("not_found", USER_MESSAGES.not_found);
  }
  if (error.status === 400) {
    return fail("validation", USER_MESSAGES.validation);
  }
  if (error.status === 429 || error.serviceCode === "budget") {
    return fail("unavailable", "The generation budget for today is used.");
  }
  if (error.status === 503 || error.serviceCode === "unavailable") {
    return fail("unavailable", "Role profile generation is unavailable.");
  }
  if (error.serviceCode === "stale") {
    return fail("conflict", "The role context changed. Refresh and try again.");
  }
  if (error.serviceCode === "unresolved") {
    return fail("conflict", "Critical role information is still unresolved.");
  }
  if (error.serviceCode === "immutable") {
    return fail("conflict", "This version cannot be changed.");
  }
  if (error.serviceCode === "duplicate") {
    return fail("conflict", "That skill is already on this version.");
  }
  if (error.status === 409 || error.serviceCode === "conflict") {
    return fail("conflict", USER_MESSAGES.conflict);
  }
  if (error.status === 504 || error.serviceCode === "timeout") {
    return fail("unavailable", "Generation timed out. Retry the same job.");
  }
  return fail("unexpected", USER_MESSAGES.unexpected);
}

export function createProfileRole(token: string, name: string): Promise<ProfileRole> {
  return send(token, "/profile-roles", { method: "POST", body: { name } }, isProfileRole, 201);
}

export function createCanonicalSkill(
  token: string,
  skill: { name: string; category: string; definition: string },
): Promise<CanonicalSkill> {
  return send(token, "/canonical-skills", { method: "POST", body: skill }, isCanonicalSkill, 201);
}

export function createRoleProfile(token: string, roleId: string, frameworkVersionId = ""): Promise<RoleProfile> {
  return send(token, "/role-profiles", { method: "POST", body: { roleId, frameworkVersionId } }, isRoleProfile, 201);
}

export function readRoleProfile(token: string, profileId: string): Promise<RoleProfile> {
  return send(token, `/role-profiles/${encodeURIComponent(profileId)}`, { method: "GET" }, isRoleProfile, 200);
}

export function listRoleProfileVersions(token: string, profileId: string): Promise<RoleProfileVersion[]> {
  return send(token, `/role-profiles/${encodeURIComponent(profileId)}/versions`, { method: "GET" }, isVersionList, 200);
}

export function readRoleProfileHistory(token: string, profileId: string): Promise<RoleProfileHistory> {
  return send(token, `/role-profiles/${encodeURIComponent(profileId)}/history`, { method: "GET" }, isHistory, 200);
}

export function compareRoleProfileVersions(token: string, profileId: string, from: string, to: string): Promise<RoleProfileComparison> {
  const query = new URLSearchParams({ from, to });
  return send(token, `/role-profiles/${encodeURIComponent(profileId)}/compare?${query}`, { method: "GET" }, isComparison, 200);
}

export function updateRoleProfileContext(token: string, versionId: string, input: ContextInput): Promise<RoleProfileVersion> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}`, { method: "PATCH", body: input }, isVersion, 200);
}

export function addRoleProfileSkill(
  token: string,
  versionId: string,
  input: {
    expectedRevision: number;
    skillId: string;
    category: string;
    definition: string;
    requiredProficiency: number;
    priority: string;
    rationale: string;
    provenance: string;
  },
): Promise<RoleProfileVersion> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/skills`, { method: "POST", body: input }, isVersion, 201);
}

export function updateRoleProfileSkill(
  token: string,
  versionId: string,
  skillAssignmentId: string,
  input: { expectedRevision: number; definition: string; requiredProficiency: number; priority: string; rationale: string },
): Promise<RoleProfileVersion> {
  return send(
    token,
    `/role-profile-versions/${encodeURIComponent(versionId)}/skills/${encodeURIComponent(skillAssignmentId)}`,
    { method: "PATCH", body: input },
    isVersion,
    200,
  );
}

export function removeRoleProfileSkill(token: string, versionId: string, skillAssignmentId: string, expectedRevision: number): Promise<RoleProfileVersion> {
  const query = new URLSearchParams({ expectedRevision: String(expectedRevision) });
  return send(
    token,
    `/role-profile-versions/${encodeURIComponent(versionId)}/skills/${encodeURIComponent(skillAssignmentId)}?${query}`,
    { method: "DELETE" },
    isVersion,
    200,
  );
}

export function reorderRoleProfileSkills(
  token: string,
  versionId: string,
  expectedRevision: number,
  skillAssignmentIds: string[],
): Promise<RoleProfileVersion> {
  return send(
    token,
    `/role-profile-versions/${encodeURIComponent(versionId)}/skill-order`,
    { method: "PUT", body: { expectedRevision, skillAssignmentIds } },
    isVersion,
    200,
  );
}

export function submitRoleProfile(token: string, versionId: string, expectedRevision: number): Promise<RoleProfileVersion> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/submit`, { method: "POST", body: { expectedRevision } }, isVersion, 200);
}

export function publishRoleProfile(token: string, versionId: string, expectedRevision: number): Promise<RoleProfileVersion> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/publish`, { method: "POST", body: { expectedRevision } }, isVersion, 200);
}

export function createNextRoleProfileDraft(token: string, versionId: string, expectedRevision: number): Promise<RoleProfileVersion> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/next-draft`, { method: "POST", body: { expectedRevision } }, isVersion, 201);
}

export function commentOnRoleProfile(token: string, versionId: string, body: string): Promise<RoleProfileComment> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/comments`, { method: "POST", body: { body } }, isComment, 201);
}

export function readClarification(token: string, versionId: string): Promise<ClarificationSession> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/clarification`, { method: "GET" }, isSession, 200);
}

export function startClarification(token: string, versionId: string): Promise<ClarificationSession> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/clarification`, { method: "POST" }, isSession, 200);
}

export function answerClarification(
  token: string,
  versionId: string,
  answers: Array<{ questionId: string; value: string }>,
): Promise<ClarificationSession> {
  return send(token, `/role-profile-versions/${encodeURIComponent(versionId)}/clarification/answers`, { method: "POST", body: { answers } }, isSession, 200);
}

export function startRoleProfileGeneration(
  token: string,
  versionId: string,
  contextVersion: number,
  idempotencyKey: string,
): Promise<GenerationJob> {
  return send(
    token,
    `/role-profile-versions/${encodeURIComponent(versionId)}/generation`,
    { method: "POST", body: { contextVersion, idempotencyKey } },
    isJob,
    [200, 202],
  );
}

export function readRoleProfileJob(token: string, jobId: string): Promise<GenerationJob> {
  return send(token, `/role-profile-jobs/${encodeURIComponent(jobId)}`, { method: "GET" }, isJob, 200);
}

export function retryRoleProfileJob(token: string, jobId: string): Promise<GenerationJob> {
  return send(token, `/role-profile-jobs/${encodeURIComponent(jobId)}/retry`, { method: "POST" }, isJob, [200, 202]);
}

async function send<T>(
  token: string,
  path: string,
  init: { method: string; body?: unknown },
  guard: (value: unknown) => value is T,
  expected: number | number[],
): Promise<T> {
  const response = await fetch(`${capabilityFrameworkUrl()}${path}`, {
    method: init.method,
    headers: {
      Authorization: `Bearer ${token}`,
      ...(init.body === undefined ? {} : { "Content-Type": "application/json" }),
    },
    body: init.body === undefined ? undefined : JSON.stringify(init.body),
    cache: "no-store",
  });
  const payload: unknown = await response.json().catch(() => null);
  const allowed = Array.isArray(expected) ? expected : [expected];
  if (!allowed.includes(response.status)) {
    const serviceCode = isRecord(payload) && typeof payload.error === "string" ? payload.error : "";
    throw new RoleProfileApiError(response.status, serviceCode);
  }
  if (!guard(payload)) {
    throw new RoleProfileApiError(response.status, "invalid_payload");
  }
  return payload;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function strings(value: Record<string, unknown>, keys: string[]): boolean {
  return keys.every((key) => typeof value[key] === "string");
}

function isProfileRole(value: unknown): value is ProfileRole {
  return isRecord(value) && strings(value, ["id", "organisationId", "name"]);
}

function isCanonicalSkill(value: unknown): value is CanonicalSkill {
  return isRecord(value) && strings(value, ["id", "organisationId", "name", "category", "definition"]);
}

function isSource(value: unknown): value is RoleProfileSource {
  return isRecord(value) && strings(value, ["id", "title", "referenceUri"]);
}

function isSkill(value: unknown): value is RoleProfileSkill {
  return (
    isRecord(value) &&
    strings(value, ["id", "skillId", "skillName", "category", "definition", "priority", "rationale", "provenance"]) &&
    typeof value.requiredProficiency === "number" &&
    typeof value.position === "number"
  );
}

function isVersion(value: unknown): value is RoleProfileVersion {
  if (!isRecord(value) || !isRecord(value.ai)) {
    return false;
  }
  return (
    strings(value, [
      "id",
      "profileId",
      "organisationId",
      "status",
      "purpose",
      "responsibilities",
      "businessOutcomes",
      "decisionScope",
      "industry",
      "region",
      "reportingLevel",
      "frameworkVersionId",
      "createdAt",
    ]) &&
    typeof value.versionNumber === "number" &&
    typeof value.revision === "number" &&
    (value.teamSize === null || typeof value.teamSize === "number") &&
    Array.isArray(value.sources) &&
    value.sources.every(isSource) &&
    Array.isArray(value.skills) &&
    value.skills.every(isSkill) &&
    typeof value.ai.model === "string" &&
    typeof value.ai.sourceContextVersion === "string" &&
    (value.ai.generatedAt === null || typeof value.ai.generatedAt === "string")
  );
}

function isVersionList(value: unknown): value is RoleProfileVersion[] {
  return Array.isArray(value) && value.every(isVersion);
}

function isRoleProfile(value: unknown): value is RoleProfile {
  return isRecord(value) && strings(value, ["id", "organisationId", "roleId", "roleName", "createdByPersonId", "createdAt"]) && isVersion(value.version);
}

function isQuestion(value: unknown): value is ClarificationQuestion {
  return (
    isRecord(value) &&
    strings(value, ["id", "profileVersionId", "field", "question", "answer", "askedActorId", "answeredActorId", "askedAt"]) &&
    typeof value.contextVersion === "number" &&
    typeof value.round === "number" &&
    typeof value.critical === "boolean" &&
    typeof value.unresolved === "boolean" &&
    (value.answeredAt === null || typeof value.answeredAt === "string")
  );
}

function isSession(value: unknown): value is ClarificationSession {
  if (!isRecord(value) || !isRecord(value.snapshot)) {
    return false;
  }
  const snapshot = value.snapshot;
  return (
    strings(value, ["profileVersionId", "contextHash"]) &&
    typeof value.contextVersion === "number" &&
    typeof value.round === "number" &&
    typeof value.complete === "boolean" &&
    Array.isArray(value.questions) &&
    value.questions.every(isQuestion) &&
    Array.isArray(value.unresolved) &&
    value.unresolved.every((item) => typeof item === "string") &&
    strings(snapshot, ["purpose", "responsibilities", "businessOutcomes", "decisionScope", "reportingLevel", "industry", "region"]) &&
    (snapshot.teamSize === null || typeof snapshot.teamSize === "number") &&
    Array.isArray(snapshot.sources)
  );
}

function isJob(value: unknown): value is GenerationJob {
  return (
    isRecord(value) &&
    strings(value, [
      "id",
      "organisationId",
      "profileVersionId",
      "contextHash",
      "idempotencyKey",
      "status",
      "modelId",
      "promptVersion",
      "inputHash",
      "outputHash",
      "lastError",
      "createdBy",
      "createdAt",
    ]) &&
    typeof value.contextVersion === "number" &&
    typeof value.latencyMs === "number" &&
    typeof value.promptTokens === "number" &&
    typeof value.completionTokens === "number" &&
    typeof value.attempts === "number" &&
    typeof value.cancelRequested === "boolean" &&
    Array.isArray(value.unsupported) &&
    Array.isArray(value.uncertainty) &&
    value.uncertainty.every((item) => typeof item === "string")
  );
}

function isComment(value: unknown): value is RoleProfileComment {
  return isRecord(value) && strings(value, ["id", "profileVersionId", "authorPersonId", "body", "createdAt"]);
}

function isHistory(value: unknown): value is RoleProfileHistory {
  return (
    isRecord(value) &&
    typeof value.profileId === "string" &&
    Array.isArray(value.events) &&
    Array.isArray(value.comments) &&
    value.comments.every(isComment)
  );
}

function isComparison(value: unknown): value is RoleProfileComparison {
  return isRecord(value) && isVersion(value.from) && isVersion(value.to) && Array.isArray(value.added) && Array.isArray(value.removed) && Array.isArray(value.changed);
}
