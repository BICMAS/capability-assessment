import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { OrganogramStage } from "@/features/role-profiles/chart";
import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { Organogram } from "@/features/role-profiles/organogram";
import { listDepartments, listOrganogramFiles, listOrganograms, listPeople, listPositions, listRoleAssignments } from "@/lib/org-directory";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function RoleProfilesPage({
  searchParams,
}: {
  searchParams: Promise<{ department?: string; error?: string; notice?: string; chart?: string; entry?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const params = await searchParams;
  const entry = params.entry === "upload" || params.entry === "template" ? params.entry : "build";
  const [departments, people, roles, charts] = await Promise.all([
    settle(listDepartments(token), "Departments"),
    settle(listPeople(token), "People"),
    settle(listRoleAssignments(token), "Role titles"),
    settle(listOrganograms(token), "Organograms"),
  ]);
  const chart = charts.ok ? (charts.value.find((item) => item.id === params.chart) ?? charts.value[0] ?? null) : null;
  const [positions, files] = chart
    ? await Promise.all([settle(listPositions(token, chart.id), "Positions"), settle(listOrganogramFiles(token, chart.id), "Organogram files")])
    : [null, null];
  return (
    <RoleProfileFrame
      current="organogram"
      error={params.error ?? ""}
      notice={params.notice ?? ""}
      evidence={{ organogramCount: charts.ok ? charts.value.length : null, clarificationComplete: null }}
    >
      <OrganogramStage
        charts={charts}
        chart={chart}
        positions={positions}
        files={files}
        departments={departments.ok ? departments.value : []}
        people={people.ok ? people.value : []}
        entry={entry}
      />
      <Organogram
        departmentId={params.department ?? ""}
        chartId={chart?.id ?? ""}
        entry={entry}
        departments={departments}
        people={people}
        roles={roles}
        error=""
        openError={params.error ?? ""}
      />
    </RoleProfileFrame>
  );
}
