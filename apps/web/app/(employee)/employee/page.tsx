import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { EmployeeAssessments } from "@/features/employee/assessments";
import { loadCapabilityRows } from "@/features/employee/capability-data";
import { EmployeeCapabilities } from "@/features/employee/capabilities";
import { EmployeeDashboard } from "@/features/employee/dashboard";
import { StatusMessage } from "@/components/ui/status-message";
import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listAssessmentSessions, type AssessmentSession } from "@/lib/generated/assessment-plan/plans";
import { readProfile } from "@/lib/org-directory";
import { settle, type Settled } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export default async function EmployeePage({
  searchParams,
}: {
  searchParams: Promise<{ sessionError?: string; assessmentError?: string; assessmentNotice?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }

  const [profile, frameworks, sessions, params] = await Promise.all([
    readProfile(token),
    settle(listPublishedFrameworks(token), "Frameworks"),
    settle(listAssessmentSessions(token), "Assessments"),
    searchParams,
  ]);

  const ownSessions: Settled<AssessmentSession[]> = sessions.ok
    ? { ok: true, value: sessions.value.filter((session) => session.personId === profile.id) }
    : sessions;

  const titles = new Set(profile.roles.map((role) => role.title));
  const roleLine = profile.roles.map((role) => role.title).join(" · ");
  const capabilityData = frameworks.ok
    ? await loadCapabilityRows(token, profile.id, titles, frameworks.value)
    : null;
  const capabilities = capabilityData?.rows ?? [];

  return (
    <div className="employee-page">
      <EmployeeDashboard
        name={profile.name}
        roleLine={roleLine}
        sessions={ownSessions}
        capabilities={capabilities}
        assessmentError={params.assessmentError ?? ""}
        assessmentNotice={params.assessmentNotice ?? ""}
      />

      {!frameworks.ok ? <StatusMessage tone="error">{frameworks.error}</StatusMessage> : null}

      <EmployeeAssessments sessions={ownSessions} sessionError={params.sessionError ?? ""} />

      <EmployeeCapabilities
        personId={profile.id}
        hasRoleTitles={titles.size > 0}
        frameworks={frameworks}
        data={capabilityData}
      />
    </div>
  );
}
