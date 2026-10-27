// Client methods for contracts/openapi/capability-framework.yaml.

export type Indicator = {
  id: string;
  statement: string;
};

export type ProficiencyLevel = {
  id: string;
  level: number;
  indicators: Indicator[];
};

export type Capability = {
  id: string;
  name: string;
  levels: ProficiencyLevel[];
};

export type FrameworkRole = {
  id: string;
  name: string;
};

export type Requirement = {
  id: string;
  roleId: string;
  capabilityId: string;
  requiredLevel: number;
};

export type RowError = {
  row: number;
  message: string;
};

export type Proposal = {
  id: string;
  kind: string;
  status: string;
  capabilityCode: string;
  message: string;
};

export type FrameworkVersion = {
  id: string;
  organisationId: string;
  frameworkId: string;
  frameworkName: string;
  versionNumber: number;
  status: string;
  roles: FrameworkRole[];
  capabilities: Capability[];
  requirements: Requirement[];
  rowErrors: RowError[];
  proposals: Proposal[];
};

export function capabilityFrameworkUrl(): string {
  const configured = process.env.CAPABILITY_FRAMEWORK_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8084";
}

export function listFrameworkVersions(token: string): Promise<FrameworkVersion[]> {
  return request(token, "/framework-versions", { method: "GET" }, isVersionList);
}

export function createDraft(token: string, name: string): Promise<FrameworkVersion> {
  return request(token, "/frameworks", { method: "POST", body: { name } }, isVersion);
}

export function addRole(token: string, versionId: string, name: string): Promise<FrameworkRole> {
  return request(token, `/framework-versions/${versionId}/roles`, { method: "POST", body: { name } }, isRole);
}

export function addCapability(token: string, versionId: string, name: string): Promise<Capability> {
  return request(
    token,
    `/framework-versions/${versionId}/capabilities`,
    { method: "POST", body: { name } },
    isCapability,
  );
}

export type RoleCapability = {
  capability: Capability;
  requirement: Requirement;
};

export function addRoleCapability(
  token: string,
  versionId: string,
  roleId: string,
  name: string,
  requiredLevel: number,
): Promise<RoleCapability> {
  return request(
    token,
    `/framework-versions/${versionId}/role-capabilities`,
    { method: "POST", body: { name, roleId, requiredLevel } },
    isRoleCapability,
  );
}

export function addRequirement(
  token: string,
  versionId: string,
  roleId: string,
  capabilityId: string,
  requiredLevel: number,
): Promise<Requirement> {
  return request(
    token,
    `/framework-versions/${versionId}/requirements`,
    { method: "POST", body: { roleId, capabilityId, requiredLevel } },
    isRequirement,
  );
}

export function addIndicator(
  token: string,
  versionId: string,
  capabilityId: string,
  level: number,
  statement: string,
): Promise<Indicator> {
  return request(
    token,
    `/framework-versions/${versionId}/indicators`,
    { method: "POST", body: { capabilityId, level, statement } },
    isIndicator,
  );
}

export function createNextDraft(token: string, versionId: string): Promise<FrameworkVersion> {
  return request(token, `/framework-versions/${versionId}/next-draft`, { method: "POST" }, isVersion);
}

export type ImportOutcome = {
  status: number;
  missingColumns: string[];
  rowErrors: RowError[];
  created: boolean;
};

export async function uploadImport(token: string, file: File): Promise<ImportOutcome> {
  const body = new FormData();
  body.append("file", file, file.name);
  const response = await fetch(`${capabilityFrameworkUrl()}/imports`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body,
    cache: "no-store",
  });
  const payload: unknown = await response.json().catch(() => null);
  return {
    status: response.status,
    missingColumns: stringList(payload, "missingColumns"),
    rowErrors: isRecord(payload) && Array.isArray(payload.rowErrors) ? payload.rowErrors.filter(isRowError) : [],
    created: response.status === 201,
  };
}

export function submitForReview(token: string, versionId: string): Promise<number> {
  return fetch(`${capabilityFrameworkUrl()}/framework-versions/${versionId}/in-review`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  }).then((response) => response.status);
}

export function listPublishedFrameworks(token: string): Promise<FrameworkVersion[]> {
  return request(token, "/published-frameworks", { method: "GET" }, isVersionList);
}

export function approveVersion(token: string, versionId: string): Promise<number> {
  return postStatus(token, `/framework-versions/${versionId}/approve`);
}

export function publishVersion(token: string, versionId: string): Promise<number> {
  return postStatus(token, `/framework-versions/${versionId}/publish`);
}

export function saveIndicatorStatement(
  token: string,
  versionId: string,
  indicatorId: string,
  statement: string,
): Promise<number> {
  return fetch(`${capabilityFrameworkUrl()}/framework-versions/${versionId}/indicators/${indicatorId}`, {
    method: "PATCH",
    headers: {
      Authorization: `Bearer ${token}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({ statement }),
    cache: "no-store",
  }).then((response) => response.status);
}

function postStatus(token: string, path: string): Promise<number> {
  return fetch(`${capabilityFrameworkUrl()}${path}`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  }).then((response) => response.status);
}

export function decideProposal(
  token: string,
  versionId: string,
  proposalId: string,
  status: "accepted" | "edited" | "rejected",
): Promise<Proposal> {
  return request(
    token,
    `/framework-versions/${versionId}/proposals/${proposalId}`,
    { method: "POST", body: { status } },
    isProposal,
  );
}

async function request<T>(
  token: string,
  path: string,
  init: { method: string; body?: unknown },
  guard: (value: unknown) => value is T,
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
  if (!response.ok) {
    throw new Error(`${path} failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!guard(payload)) {
    throw new Error(`${path} returned an unexpected payload`);
  }
  return payload;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isIndicator(value: unknown): value is Indicator {
  return isRecord(value) && typeof value.id === "string" && typeof value.statement === "string";
}

function isLevel(value: unknown): value is ProficiencyLevel {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.level === "number" &&
    Array.isArray(value.indicators) &&
    value.indicators.every(isIndicator)
  );
}

function isCapability(value: unknown): value is Capability {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.name === "string" &&
    Array.isArray(value.levels) &&
    value.levels.every(isLevel)
  );
}

function isRole(value: unknown): value is FrameworkRole {
  return isRecord(value) && typeof value.id === "string" && typeof value.name === "string";
}

function isRequirement(value: unknown): value is Requirement {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.roleId === "string" &&
    typeof value.capabilityId === "string" &&
    typeof value.requiredLevel === "number"
  );
}

function isRoleCapability(value: unknown): value is RoleCapability {
  return isRecord(value) && isCapability(value.capability) && isRequirement(value.requirement);
}

function isRowError(value: unknown): value is RowError {
  return isRecord(value) && typeof value.row === "number" && typeof value.message === "string";
}

function isProposal(value: unknown): value is Proposal {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.kind === "string" &&
    typeof value.status === "string" &&
    typeof value.capabilityCode === "string" &&
    typeof value.message === "string"
  );
}

function stringList(value: unknown, key: string): string[] {
  if (!isRecord(value) || !Array.isArray(value[key])) {
    return [];
  }
  return value[key].filter((item): item is string => typeof item === "string");
}

function isVersion(value: unknown): value is FrameworkVersion {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.frameworkId === "string" &&
    typeof value.frameworkName === "string" &&
    typeof value.versionNumber === "number" &&
    typeof value.status === "string" &&
    Array.isArray(value.roles) &&
    value.roles.every(isRole) &&
    Array.isArray(value.capabilities) &&
    value.capabilities.every(isCapability) &&
    Array.isArray(value.requirements) &&
    value.requirements.every(isRequirement) &&
    Array.isArray(value.rowErrors) &&
    value.rowErrors.every(isRowError) &&
    Array.isArray(value.proposals) &&
    value.proposals.every(isProposal)
  );
}

function isVersionList(value: unknown): value is FrameworkVersion[] {
  return Array.isArray(value) && value.every(isVersion);
}
