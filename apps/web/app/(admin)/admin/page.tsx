import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { listAuditRecords, type AuditRecord } from "@/lib/audit";
import { listFrameworkVersions } from "@/lib/capability-framework";
import { readAssignment, readPlan, type AssignmentResult } from "@/lib/generated/assessment-plan/plans";
import { listDepartments, listPeople, listRoleAssignments } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

import { AssignmentStep } from "./assignment-step";
import { FrameworkEditor } from "./framework-editor";
import { PlanStep } from "./plan-step";

export default async function AdminPage({
  searchParams,
}: {
  searchParams: Promise<{
    review?: string;
    action?: string;
    versionId?: string;
    importError?: string;
    save?: string;
    framework?: string;
    version?: string;
    assignment?: string;
    plan?: string;
    results?: string;
    assignmentError?: string;
    planError?: string;
  }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const params = await searchParams;
  const [departments, roleAssignments, people, versions, records] = await Promise.all([
    listDepartments(token),
    listRoleAssignments(token),
    listPeople(token),
    listFrameworkVersions(token),
    listAuditRecords(token),
  ]);
  const assessment = await loadAssessment(token, params.assignment ?? "", params.plan ?? "");
  const titles = new Map<string, string[]>();
  for (const roleAssignment of roleAssignments) {
    const current = titles.get(roleAssignment.personId) ?? [];
    current.push(roleAssignment.title);
    titles.set(roleAssignment.personId, current);
  }
  const published = versions.filter((version) => version.status === "published");
  const selectedVersion = versions.find((version) => version.id === assessment.assignment?.frameworkVersionId) ?? null;
  const selectedRole = selectedVersion?.roles.find((role) => role.id === assessment.assignment?.roleId);
  const personName = people.find((person) => person.id === assessment.assignment?.personId)?.name ?? "Person";
  const approverName = people.find((person) => person.id === assessment.plan?.approvedByPersonId)?.name ?? assessment.plan?.approvedByPersonId ?? "";
  const saveError =
    params.save === "frozen" && params.framework && params.version
      ? `${params.framework} version ${params.version} cannot be changed.`
      : "";

  return (
    <div className="studio-page">
      <section className="studio-region">
        <h2>Studio</h2>
        <FrameworkEditor
          versions={versions}
          reviewBlocked={params.review === "blocked"}
          blockedAction={params.action ?? ""}
          blockedVersionId={params.versionId ?? ""}
          importError={params.importError ?? ""}
          saveError={saveError}
          frozenFramework={params.framework ?? ""}
          frozenVersion={params.version ?? ""}
          frozenVersionId={params.save === "frozen" ? (params.versionId ?? "") : ""}
        />
      </section>
      <section className="studio-region" id="assignments">
        <h2>Assignments</h2>
        <AssignmentStep
          people={people.map((person) => ({
            id: person.id,
            name: person.name,
            departmentName: person.departmentName,
            titles: titles.get(person.id) ?? [],
          }))}
          versions={published.map((version) => ({
            id: version.id,
            frameworkName: version.frameworkName,
            versionNumber: version.versionNumber,
            roles: version.roles,
            capabilities: version.capabilities.map((capability) => ({ id: capability.id, name: capability.name })),
            requirements: version.requirements,
          }))}
          results={decodeResults(params.results)}
          error={params.assignmentError ?? ""}
        />
      </section>
      <section className="studio-region" id="plans">
        <h2>Plans</h2>
        {assessment.assignment ? (
          <PlanStep
            assignment={assessment.assignment}
            plan={assessment.plan}
            version={selectedVersion}
            personName={personName}
            roleName={selectedRole?.name ?? "Role"}
            approverName={approverName}
            error={params.planError ?? assessment.error}
          />
        ) : (
          <p>{assessment.error || "Confirm an assignment to generate a plan."}</p>
        )}
      </section>
      <section className="studio-region">
        <h2>Organisation</h2>
        <div className="org-grid">
          <div>
            <h3>Departments</h3>
            <ul>
              {departments.map((department) => (
                <li key={department.id}>{department.name}</li>
              ))}
            </ul>
          </div>
          <div>
            <h3>Role assignments</h3>
            <ul>
              {roleAssignments.map((assignment) => (
                <li key={assignment.id}>
                  {assignment.personName} — {assignment.title}
                </li>
              ))}
            </ul>
          </div>
        </div>
      </section>
      <section className="studio-region">
        <h2>Record</h2>
        <ul className="record-list">
          {records.length === 0 ? <li className="record-empty">None</li> : null}
          {records.map((record) => (
            <li key={`${record.action}-${record.entityId}-${record.occurredAt}`}>
              <span className="record-action">{record.action}</span>
              <span className="record-actor">{record.actorPersonId}</span>
              <time dateTime={record.occurredAt}>{recordTime(record)}</time>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}

async function loadAssessment(token: string, assignmentId: string, planId: string) {
  if (!assignmentId) {
    return { assignment: null, plan: null, error: "" };
  }
  try {
    const assignment = await readAssignment({ token, assignmentId });
    if (!assignment) {
      return { assignment: null, plan: null, error: "That assignment was not found." };
    }
    if (!planId) {
      return { assignment, plan: null, error: "" };
    }
    const plan = await readPlan({ token, planId });
    if (!plan || plan.assignmentId !== assignment.id) {
      return { assignment, plan: null, error: "That plan was not found." };
    }
    return { assignment, plan, error: "" };
  } catch {
    return { assignment: null, plan: null, error: "The assessment plan service is unavailable." };
  }
}

function decodeResults(encoded: string | undefined): AssignmentResult[] {
  if (!encoded) {
    return [];
  }
  try {
    const parsed: unknown = JSON.parse(Buffer.from(encoded, "base64url").toString("utf8"));
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed.filter(isAssignmentResult);
  } catch {
    return [];
  }
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

function recordTime(record: AuditRecord): string {
  const parsed = Date.parse(record.occurredAt);
  if (Number.isNaN(parsed)) {
    return record.occurredAt;
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: "UTC",
  }).format(new Date(parsed));
}
