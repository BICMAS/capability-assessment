import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { WorkflowShell } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { Workbench } from "@/features/role-profiles/workbench";
import { readRoleProfile, readRoleProfileHistory } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function SkillsPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string; skillId?: string; category?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { profileId } = await params;
  const query = await searchParams;
  try {
    const profile = await readRoleProfile(token, profileId);
    const history = await settle(readRoleProfileHistory(token, profile.id), "Review history");
    return (
      <WorkflowShell token={token} profile={profile} current="skills" error={query.error ?? ""} notice={query.notice ?? ""}>
        <Workbench profile={profile} history={history} skillId={query.skillId ?? ""} category={query.category ?? ""} />
      </WorkflowShell>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
