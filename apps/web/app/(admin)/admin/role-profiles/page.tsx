import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { Organogram } from "@/features/role-profiles/organogram";
import { listDepartments, listPeople, listRoleAssignments } from "@/lib/org-directory";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function RoleProfilesPage({
  searchParams,
}: {
  searchParams: Promise<{ department?: string; error?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const params = await searchParams;
  const [departments, people, roles] = await Promise.all([
    settle(listDepartments(token), "Departments"),
    settle(listPeople(token), "People"),
    settle(listRoleAssignments(token), "Role titles"),
  ]);
  return (
    <RoleProfileFrame current="directory" error={params.error ?? ""}>
      <Organogram departmentId={params.department ?? ""} departments={departments} people={people} roles={roles} error="" />
    </RoleProfileFrame>
  );
}
