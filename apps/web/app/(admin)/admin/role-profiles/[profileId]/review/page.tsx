import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { WorkflowShell } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { ReviewPanel } from "@/features/role-profiles/review-panel";
import { listRoleProfileVersions, readRoleProfile, readRoleProfileHistory, listRoleProfileApprovals } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ReviewPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { profileId } = await params;
  const query = await searchParams;
  try {
    const profile = await readRoleProfile(token, profileId);
    const [history, versions, approvals] = await Promise.all([
      settle(readRoleProfileHistory(token, profile.id), "Review history"),
      settle(listRoleProfileVersions(token, profile.id), "Earlier versions"),
      settle(listRoleProfileApprovals(token, profile.version.id), "Approval record"),
    ]);
    return (
      <WorkflowShell token={token} profile={profile} current="approve" error={query.error ?? ""} notice={query.notice ?? ""}>
        <ReviewPanel profile={profile} history={history} versions={versions} approvals={approvals} />
      </WorkflowShell>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
