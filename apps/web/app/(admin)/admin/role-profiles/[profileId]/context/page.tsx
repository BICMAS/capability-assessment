import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { ContextForm, type FrameworkChoice, type PositionChoice } from "@/features/role-profiles/context-form";
import { WorkflowShell } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { listPublishedFrameworks } from "@/lib/capability-framework";
import { listDepartments, listOrganograms, listPositions } from "@/lib/org-directory";
import { readRoleProfile } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ContextPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string; organogram?: string; position?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { profileId } = await params;
  const query = await searchParams;
  try {
    const profile = await readRoleProfile(token, profileId);
    const charts = await settle(listOrganograms(token), "Organograms");
    const departments = await settle(listDepartments(token), "Departments");
    const frameworks = await settle(listPublishedFrameworks(token), "Frameworks");
    const positions: PositionChoice[] = [];
    let positionError = "";
    if (charts.ok && departments.ok) {
      const names = new Map(departments.value.map((department) => [department.id, department.name]));
      for (const chart of charts.value) {
        try {
          const rows = await listPositions(token, chart.id);
          const byId = new Map(rows.map((position) => [position.id, position]));
          for (const position of rows) {
            const parent = position.reportsToPositionId ? byId.get(position.reportsToPositionId) : undefined;
            positions.push({
              organogramId: chart.id,
              organogramName: chart.name,
              positionId: position.id,
              title: position.title,
              departmentName: names.get(position.departmentId) ?? "",
              reportsToTitle: parent?.title ?? "",
            });
          }
        } catch {
          positionError = `${chart.name} positions could not be loaded.`;
        }
      }
    }
    const frameworkChoices: FrameworkChoice[] = frameworks.ok
      ? frameworks.value.map((framework) => ({ id: framework.id, label: `${framework.frameworkName} version ${framework.versionNumber}` }))
      : [];
    if (profile.version.frameworkVersionId && !frameworkChoices.some((framework) => framework.id === profile.version.frameworkVersionId)) {
      frameworkChoices.unshift({ id: profile.version.frameworkVersionId, label: "Current framework" });
    }
    const directoryError = [!charts.ok ? charts.error : "", !departments.ok ? departments.error : "", positionError, !frameworks.ok ? frameworks.error : ""].filter(Boolean).join(" ");
    return (
      <WorkflowShell token={token} profile={profile} current="role" error={query.error ?? ""} notice={query.notice ?? ""}>
        <p>
          <Link href={`/admin/role-profiles/${profile.id}/sources`}>Add another reference</Link>
        </p>
        <ContextForm
          profile={profile}
          positions={positions}
          frameworks={frameworkChoices}
          directoryError={directoryError}
          requestedOrganogramId={(query.organogram ?? "").trim()}
          requestedPositionId={(query.position ?? "").trim()}
        />
      </WorkflowShell>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
