import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { ReviewPanel } from "@/features/role-profiles/review-panel";
import { listRoleProfileVersions, readRoleProfile, readRoleProfileHistory } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ReviewInboxPage({ searchParams }: { searchParams: Promise<{ profile?: string; error?: string; notice?: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const query = await searchParams;
  return (
    <RoleProfileFrame current="review" error={query.error ?? ""} notice={query.notice ?? ""}>
      <p>The capability service does not provide a profile index. Open a profile to review the version the service returns.</p>
      <form method="get" className="framework-form">
        <label>
          Profile id
          <input name="profile" defaultValue={query.profile ?? ""} required />
        </label>
        <button className="primary" type="submit">
          Open for review
        </button>
      </form>
      {query.profile ? <InboxProfile token={token} profileId={query.profile} /> : null}
    </RoleProfileFrame>
  );
}

async function InboxProfile({ token, profileId }: { token: string; profileId: string }) {
  try {
    const profile = await readRoleProfile(token, profileId);
    const [history, versions] = await Promise.all([
      settle(readRoleProfileHistory(token, profile.id), "Review history"),
      settle(listRoleProfileVersions(token, profile.id), "Earlier versions"),
    ]);
    return <ReviewPanel profile={profile} history={history} versions={versions} />;
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
