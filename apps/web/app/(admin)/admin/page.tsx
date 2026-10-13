import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { listAuditRecords, type AuditRecord } from "@/lib/audit";
import { listFrameworkVersions } from "@/lib/capability-framework";
import { listDepartments, listRoleAssignments } from "@/lib/org-directory";
import { SESSION_COOKIE } from "@/lib/session";

import { FrameworkEditor } from "./framework-editor";

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
  }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const params = await searchParams;
  const [departments, assignments, versions, records] = await Promise.all([
    listDepartments(token),
    listRoleAssignments(token),
    listFrameworkVersions(token),
    listAuditRecords(token),
  ]);
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
              {assignments.map((assignment) => (
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
