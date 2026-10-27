import { cookies } from "next/headers";
import Link from "next/link";
import { redirect } from "next/navigation";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { AssignmentStep } from "@/features/assignments/assignment-step";
import { PlanStep } from "@/features/assessment-plans/plan-step";
import { FrameworkEditor } from "@/features/frameworks/framework-editor";
import { listAuditRecords, type AuditRecord } from "@/lib/audit";
import { listFrameworkVersions, type FrameworkVersion } from "@/lib/capability-framework";
import {
  listAssessmentSessions,
  readAssignment,
  readPlan,
  type AssessmentSession,
  type AssignmentResult,
} from "@/lib/generated/assessment-plan/plans";
import { listDepartments, listPeople, listRoleAssignments } from "@/lib/org-directory";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

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
    settle(listDepartments(token), "Departments"),
    settle(listRoleAssignments(token), "Role assignments"),
    settle(listPeople(token), "People"),
    settle(listFrameworkVersions(token), "Frameworks"),
    settle(listAuditRecords(token), "Audit record"),
  ]);

  const assessment = await loadAssessment(token, params.assignment ?? "", params.plan ?? "");
  const peopleList = people.ok ? people.value : [];
  const versionList = versions.ok ? versions.value : [];
  const roleList = roleAssignments.ok ? roleAssignments.value : [];
  const titles = new Map<string, string[]>();
  for (const roleAssignment of roleList) {
    const current = titles.get(roleAssignment.personId) ?? [];
    current.push(roleAssignment.title);
    titles.set(roleAssignment.personId, current);
  }
  const published = versionList.filter((version) => version.status === "published");
  const assignmentResults = decodeResults(params.results);
  const selectedVersion = versionList.find((version) => version.id === assessment.assignment?.frameworkVersionId) ?? null;
  const selectedRole = selectedVersion?.roles.find((role) => role.id === assessment.assignment?.roleId);
  const personName = peopleList.find((person) => person.id === assessment.assignment?.personId)?.name ?? "Person";
  const approverName =
    peopleList.find((person) => person.id === assessment.plan?.approvedByPersonId)?.name ??
    assessment.plan?.approvedByPersonId ??
    "";
  const saveError =
    params.save === "frozen" && params.framework && params.version
      ? `${params.framework} version ${params.version} cannot be changed.`
      : "";

  return (
    <div className="studio-page">
      <p>
        <Link className="learn-link" href="/admin/role-profiles">
          Role skills profiles
        </Link>
      </p>
      <section className="studio-region">
        <header className="studio-step">
          <h2>Frameworks</h2>
          <p className="studio-step-status">{versions.ok ? frameworkStatus(versionList) : versions.error}</p>
        </header>
        {versions.ok ? (
          <FrameworkEditor
            versions={versionList}
            reviewBlocked={params.review === "blocked"}
            blockedAction={params.action ?? ""}
            blockedVersionId={params.versionId ?? ""}
            importError={params.importError ?? ""}
            saveError={saveError}
            frozenFramework={params.framework ?? ""}
            frozenVersion={params.version ?? ""}
            frozenVersionId={params.save === "frozen" ? (params.versionId ?? "") : ""}
          />
        ) : (
          <StatusMessage tone="error">{versions.error}</StatusMessage>
        )}
      </section>
      <section className="studio-region" id="assignments">
        <header className="studio-step">
          <h2>Assignments</h2>
          <p className={assignmentResults.some((result) => result.outcome === "failed") ? "studio-step-status is-error" : "studio-step-status"}>
            {!people.ok
              ? people.error
              : !versions.ok
                ? versions.error
                : assignmentStatus(assignmentResults)}
          </p>
        </header>
        {people.ok && versions.ok ? (
          <AssignmentStep
            people={peopleList.map((person) => ({
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
            results={assignmentResults}
            error={params.assignmentError ?? ""}
          />
        ) : (
          <StatusMessage tone="error">
            {!people.ok ? people.error : !versions.ok ? versions.error : "Assignments are unavailable."}
          </StatusMessage>
        )}
      </section>
      <section className="studio-region" id="plans">
        <header className="studio-step">
          <h2>Plans</h2>
          <p className={assessment.error ? "studio-step-status is-error" : "studio-step-status"}>
            {planStatus(assessment, personName, selectedRole?.name ?? "", approverName)}
          </p>
        </header>
        {assessment.assignment ? (
          <PlanStep
            assignment={assessment.assignment}
            plan={assessment.plan}
            session={sessionForPlan(await personSessions(token, assessment.assignment.personId), assessment.plan?.id ?? "")}
            version={selectedVersion}
            personName={personName}
            roleName={selectedRole?.name ?? "Role"}
            approverName={approverName}
            error={params.planError ?? assessment.error}
          />
        ) : (
          <EmptyState>{assessment.error || "Confirm an assignment to generate a plan."}</EmptyState>
        )}
      </section>
      <section className="studio-region">
        <h2>Organisation</h2>
        {!departments.ok || !roleAssignments.ok ? (
          <StatusMessage tone="error">
            {[!departments.ok ? departments.error : null, !roleAssignments.ok ? roleAssignments.error : null]
              .filter(Boolean)
              .join(" ")}
          </StatusMessage>
        ) : (
          <div className="org-grid">
            <div>
              <h3>Departments</h3>
              <ul>
                {departments.value.length === 0 ? <li className="record-empty">None</li> : null}
                {departments.value.map((department) => (
                  <li key={department.id}>{department.name}</li>
                ))}
              </ul>
            </div>
            <div>
              <h3>Role assignments</h3>
              <ul>
                {roleAssignments.value.length === 0 ? <li className="record-empty">None</li> : null}
                {roleAssignments.value.map((assignment) => (
                  <li key={assignment.id}>
                    {assignment.personName} — {assignment.title}
                  </li>
                ))}
              </ul>
            </div>
          </div>
        )}
      </section>
      <section className="studio-region">
        <h2>Record</h2>
        {!records.ok ? (
          <StatusMessage tone="error">{records.error}</StatusMessage>
        ) : (
          <ul className="record-list">
            {records.value.length === 0 ? <li className="record-empty">None</li> : null}
            {records.value.map((record) => (
              <li key={`${record.action}-${record.entityId}-${record.occurredAt}`}>
                <span className="record-action">{record.action}</span>
                <span className="record-actor">{record.actorPersonId}</span>
                <time dateTime={record.occurredAt}>{recordTime(record)}</time>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

async function personSessions(token: string, personId: string): Promise<AssessmentSession[]> {
  try {
    return await listAssessmentSessions(token, personId);
  } catch {
    return [];
  }
}

function sessionForPlan(sessions: AssessmentSession[], planId: string): AssessmentSession | null {
  return sessions.find((session) => session.planId === planId) ?? null;
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

function frameworkStatus(versions: FrameworkVersion[]): string {
  const published = versions.filter((version) => version.status === "published");
  const open = versions.filter((version) => version.status !== "published" && version.status !== "retired");
  if (versions.length === 0) {
    return "No framework yet.";
  }
  const locked =
    published.length === 0
      ? "Nothing is published."
      : published.length === 1
        ? `${published[0]?.frameworkName ?? "Framework"} v${published[0]?.versionNumber ?? ""} is locked.`
        : `${published.length} published documents are locked.`;
  if (open.length === 0) {
    return locked;
  }
  return `${locked} ${open.length === 1 ? "One version is still open." : `${open.length} versions are still open.`}`;
}

function assignmentStatus(results: AssignmentResult[]): string {
  if (results.length === 0) {
    return "Search people, choose a published role, then confirm.";
  }
  const assigned = results.filter((result) => result.outcome === "assigned").length;
  const skipped = results.filter((result) => result.outcome === "skipped").length;
  const failed = results.filter((result) => result.outcome === "failed").length;
  return [assigned > 0 ? `${assigned} assigned` : "", skipped > 0 ? `${skipped} skipped` : "", failed > 0 ? `${failed} failed` : ""]
    .filter((part) => part.length > 0)
    .join(" · ");
}

function planStatus(
  assessment: { assignment: { id: string } | null; plan: { status: string } | null; error: string },
  personName: string,
  roleName: string,
  approverName: string,
): string {
  if (!assessment.assignment) {
    return assessment.error || "Confirm an assignment to generate a plan.";
  }
  const who = `${personName} · ${roleName || "Role"}`;
  if (!assessment.plan) {
    return `${who}. Ready to generate.`;
  }
  if (assessment.plan.status === "approved") {
    return `${who}. Approved by ${approverName}.`;
  }
  return `${who}. ${assessment.plan.status.replaceAll("_", " ")}.`;
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
