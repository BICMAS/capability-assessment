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
