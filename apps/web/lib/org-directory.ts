// Client methods for contracts/openapi/org-directory.yaml.

export type Department = {
  id: string;
  organisationId: string;
  name: string;
};

export type DirectoryPerson = {
  id: string;
  organisationId: string;
  name: string;
  departmentName: string;
};

export type RoleTitle = {
  id: string;
  title: string;
};

export type Profile = {
  id: string;
  organisationId: string;
  name: string;
  departmentName: string;
  roles: RoleTitle[];
};

export type RoleAssignment = {
  id: string;
  organisationId: string;
  personId: string;
  personName: string;
  title: string;
  createdAt: string;
};

export function orgDirectoryUrl(): string {
  const configured = process.env.ORG_DIRECTORY_URL;
  if (configured && configured.length > 0) {
    return configured.replace(/\/$/, "");
  }
  return "http://localhost:8083";
}

export function listDepartments(token: string): Promise<Department[]> {
  return getJSON(token, "/departments", isDepartmentList);
}

export function listPeople(token: string): Promise<DirectoryPerson[]> {
  return getJSON(token, "/people", isPersonList);
}

export function listRoleAssignments(token: string): Promise<RoleAssignment[]> {
  return getJSON(token, "/role-assignments", isRoleAssignmentList);
}

export function readProfile(token: string): Promise<Profile> {
  return getJSON(token, "/profile", isProfile);
}

export function readPerson(token: string, personId: string): Promise<Profile> {
  return getJSON(token, `/people/${encodeURIComponent(personId)}`, isProfile);
}

export type Organogram = {
  id: string;
  organisationId: string;
  name: string;
  status: string;
  createdByPersonId: string;
  createdAt: string;
};

export type Position = {
  id: string;
  organisationId: string;
  organogramId: string;
  departmentId: string;
  title: string;
  personId: string;
  reportsToPositionId: string;
  createdAt: string;
};

export type OrganogramFile = {
  id: string;
  organisationId: string;
  organogramId: string;
  fileName: string;
  mediaType: string;
  byteSize: number;
  status: string;
  detail: string;
  createdByPersonId: string;
  createdAt: string;
};

export type PositionInput = {
  title: string;
  departmentId: string;
  personId: string;
  reportsToPositionId: string;
};

export function listOrganograms(token: string): Promise<Organogram[]> {
  return getJSON(token, "/organograms", isOrganogramList);
}

export function listPositions(token: string, organogramId: string): Promise<Position[]> {
  return getJSON(token, `/organograms/${encodeURIComponent(organogramId)}/positions`, isPositionList);
}

export function createOrganogram(token: string, name: string): Promise<Organogram> {
  return postJSON(token, "/organograms", { name }, isOrganogram);
}

export function createPosition(token: string, organogramId: string, input: PositionInput): Promise<Position> {
  return postJSON(token, `/organograms/${encodeURIComponent(organogramId)}/positions`, input, isPosition);
}

export function updatePosition(token: string, organogramId: string, positionId: string, input: PositionInput): Promise<Position> {
  return sendJSON(token, "PATCH", `/organograms/${encodeURIComponent(organogramId)}/positions/${encodeURIComponent(positionId)}`, input, isPosition);
}

export function deletePosition(token: string, organogramId: string, positionId: string): Promise<void> {
  return sendEmpty(token, "DELETE", `/organograms/${encodeURIComponent(organogramId)}/positions/${encodeURIComponent(positionId)}`, 204);
}

export function publishOrganogram(token: string, organogramId: string): Promise<Organogram> {
  return sendJSON(token, "POST", `/organograms/${encodeURIComponent(organogramId)}/publish`, {}, isOrganogram);
}

export function listOrganogramFiles(token: string, organogramId: string): Promise<OrganogramFile[]> {
  return getJSON(token, `/organograms/${encodeURIComponent(organogramId)}/files`, isOrganogramFileList);
}

async function getJSON<T>(
  token: string,
  path: string,
  guard: (value: unknown) => value is T,
): Promise<T> {
  const response = await fetch(`${orgDirectoryUrl()}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
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

async function postJSON<T>(token: string, path: string, body: unknown, guard: (value: unknown) => value is T): Promise<T> {
  return sendJSON(token, "POST", path, body, guard);
}

async function sendJSON<T>(token: string, method: string, path: string, body: unknown, guard: (value: unknown) => value is T): Promise<T> {
  const response = await fetch(`${orgDirectoryUrl()}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}`, "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });
  if (!response.ok) {
    throw await directoryError(response, path);
  }
  const payload: unknown = await response.json();
  if (!guard(payload)) {
    throw new Error(`${path} returned an unexpected payload`);
  }
  return payload;
}

async function sendEmpty(token: string, method: string, path: string, status: number): Promise<void> {
  const response = await fetch(`${orgDirectoryUrl()}${path}`, {
    method,
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (response.status !== status) {
    throw await directoryError(response, path);
  }
}

async function directoryError(response: Response, path: string): Promise<Error> {
  const text = (await response.text()).trim();
  return new Error(text || `${path} failed with status ${response.status}`);
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isDepartment(value: unknown): value is Department {
  if (!isRecord(value)) {
    return false;
  }
  return (
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.name === "string"
  );
}

function isDepartmentList(value: unknown): value is Department[] {
  return Array.isArray(value) && value.every(isDepartment);
}

function isPerson(value: unknown): value is DirectoryPerson {
  if (!isRecord(value)) {
    return false;
  }
  return (
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.name === "string" &&
    typeof value.departmentName === "string"
  );
}

function isPersonList(value: unknown): value is DirectoryPerson[] {
  return Array.isArray(value) && value.every(isPerson);
}

function isRoleTitle(value: unknown): value is RoleTitle {
  if (!isRecord(value)) {
    return false;
  }
  return typeof value.id === "string" && typeof value.title === "string";
}

function isProfile(value: unknown): value is Profile {
  if (!isRecord(value) || !Array.isArray(value.roles) || !value.roles.every(isRoleTitle)) {
    return false;
  }
  return (
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.name === "string" &&
    typeof value.departmentName === "string"
  );
}

function isRoleAssignment(value: unknown): value is RoleAssignment {
  if (!isRecord(value)) {
    return false;
  }
  return (
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.personId === "string" &&
    typeof value.personName === "string" &&
    typeof value.title === "string" &&
    typeof value.createdAt === "string"
  );
}

function isRoleAssignmentList(value: unknown): value is RoleAssignment[] {
  return Array.isArray(value) && value.every(isRoleAssignment);
}

function isOrganogram(value: unknown): value is Organogram {
  return isRecord(value) && typeof value.id === "string" && typeof value.organisationId === "string" && typeof value.name === "string" && typeof value.status === "string" && typeof value.createdByPersonId === "string" && typeof value.createdAt === "string";
}

function isOrganogramList(value: unknown): value is Organogram[] {
  return Array.isArray(value) && value.every(isOrganogram);
}

function isPosition(value: unknown): value is Position {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.organisationId === "string" &&
    typeof value.organogramId === "string" &&
    typeof value.departmentId === "string" &&
    typeof value.title === "string" &&
    typeof value.personId === "string" &&
    typeof value.reportsToPositionId === "string" &&
    typeof value.createdAt === "string"
  );
}

function isPositionList(value: unknown): value is Position[] {
  return Array.isArray(value) && value.every(isPosition);
}

function isOrganogramFile(value: unknown): value is OrganogramFile {
  return (
    isRecord(value) &&
    typeof value.id === "string" &&
    typeof value.organogramId === "string" &&
    typeof value.fileName === "string" &&
    typeof value.mediaType === "string" &&
    typeof value.byteSize === "number" &&
    typeof value.status === "string" &&
    typeof value.detail === "string"
  );
}

function isOrganogramFileList(value: unknown): value is OrganogramFile[] {
  return Array.isArray(value) && value.every(isOrganogramFile);
}
