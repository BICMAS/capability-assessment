import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { HistoryPanel } from "@/features/role-profiles/history-panel";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { compareRoleProfileVersions, listRoleProfileVersions, readRoleProfile, readRoleProfileHistory } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function HistoryPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string; from?: string; to?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { profileId } = await params;
  const query = await searchParams;
  try {
    const profile = await readRoleProfile(token, profileId);
    const [history, versions] = await Promise.all([
      settle(readRoleProfileHistory(token, profile.id), "History"),
      settle(listRoleProfileVersions(token, profile.id), "Versions"),
    ]);
    const comparison = query.from && query.to ? await settle(compareRoleProfileVersions(token, profile.id, query.from, query.to), "Comparison") : null;
    return (
      <RoleProfileFrame profile={profile} current="history" error={query.error ?? ""} notice={query.notice ?? ""}>
        <HistoryPanel profile={profile} history={history} versions={versions} comparison={comparison} />
      </RoleProfileFrame>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
