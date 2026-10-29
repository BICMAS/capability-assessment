import { assessmentPlanUrl } from "@/lib/generated/assessment-plan/plans";

export type InsightSkill = {
  skillId: string;
  requiredProficiency: number;
  state: string;
  currentLevel?: string;
  gap?: string;
};

export type InsightPerson = {
  personId: string;
  department?: string;
  roleName?: string;
  skills: InsightSkill[];
};

export type DepartmentSummary = {
  department: string;
  requiredRows: number;
  missingRows: number;
  belowRows: number;
  metRows: number;
};

export type InsightSummary = {
  method: string;
  requiredRows: number;
  assessedRows: number;
  missingRows: number;
  belowRows: number;
  metRows: number;
  criticalSkillIds: string[];
  departments: DepartmentSummary[];
};

export type ProfileInsight = {
  profileVersionId: string;
  population: string[];
  people: InsightPerson[];
  employmentDecision: string;
  summary?: InsightSummary;
};

export type InsightQuery = {
  profileVersionId: string;
  personId?: string;
  department?: string;
  role?: string;
  q?: string;
  from?: string;
  to?: string;
};

export function readProfileInsights(token: string, query: InsightQuery): Promise<ProfileInsight> {
  const params = new URLSearchParams({ profileVersionId: query.profileVersionId });
  for (const [key, value] of Object.entries({
    personId: query.personId,
    department: query.department,
    role: query.role,
    q: query.q,
    from: query.from,
    to: query.to,
  })) {
    if (value) {
      params.set(key, value);
    }
  }
  return getJSON(token, `/profile-insights?${params.toString()}`);
}

async function getJSON(token: string, path: string): Promise<ProfileInsight> {
  const response = await fetch(`${assessmentPlanUrl()}${path}`, {
    headers: { Authorization: `Bearer ${token}` },
    cache: "no-store",
  });
  if (!response.ok) {
    throw new Error(`profile insights failed with status ${response.status}`);
  }
  const payload: unknown = await response.json();
  if (!isInsight(payload)) {
    throw new Error("profile insights returned an unexpected payload");
  }
  return payload;
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null;
}

function isInsight(value: unknown): value is ProfileInsight {
  return (
    isRecord(value) &&
    typeof value.profileVersionId === "string" &&
    typeof value.employmentDecision === "string" &&
    Array.isArray(value.population) &&
    value.population.every((item) => typeof item === "string") &&
    Array.isArray(value.people) &&
    value.people.every(isPerson) &&
    (!("summary" in value) || value.summary == null || isSummary(value.summary))
  );
}

function isPerson(value: unknown): value is InsightPerson {
  if (!isRecord(value) || typeof value.personId !== "string" || !Array.isArray(value.skills) || !value.skills.every(isSkill)) {
    return false;
  }
  if ("department" in value && value.department != null && typeof value.department !== "string") {
    return false;
  }
  if ("roleName" in value && value.roleName != null && typeof value.roleName !== "string") {
    return false;
  }
  return true;
}

function isSummary(value: unknown): value is InsightSummary {
  return (
    isRecord(value) &&
    typeof value.method === "string" &&
    typeof value.requiredRows === "number" &&
    typeof value.assessedRows === "number" &&
    typeof value.missingRows === "number" &&
    typeof value.belowRows === "number" &&
    typeof value.metRows === "number" &&
    Array.isArray(value.criticalSkillIds) &&
    value.criticalSkillIds.every((item) => typeof item === "string") &&
    Array.isArray(value.departments) &&
    value.departments.every(isDepartment)
  );
}

function isDepartment(value: unknown): value is DepartmentSummary {
  return (
    isRecord(value) &&
    typeof value.department === "string" &&
    typeof value.requiredRows === "number" &&
    typeof value.missingRows === "number" &&
    typeof value.belowRows === "number" &&
    typeof value.metRows === "number"
  );
}

function isSkill(value: unknown): value is InsightSkill {
  if (!isRecord(value) || typeof value.skillId !== "string" || typeof value.state !== "string" || typeof value.requiredProficiency !== "number") {
    return false;
  }
  if ("currentLevel" in value && value.currentLevel != null && typeof value.currentLevel !== "string") {
    return false;
  }
  if ("gap" in value && value.gap != null && typeof value.gap !== "string") {
    return false;
  }
  return true;
}
