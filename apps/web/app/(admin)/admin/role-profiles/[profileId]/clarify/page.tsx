import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { ClarificationPanel } from "@/features/role-profiles/clarification-panel";
import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { readClarification, readRoleProfile } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ClarifyPage({
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
    const session = await settle(readClarification(token, profile.version.id), "Clarification");
    return (
      <RoleProfileFrame profile={profile} current="clarify" error={query.error ?? ""} notice={query.notice ?? ""}>
        <ClarificationPanel profileId={profile.id} versionId={profile.version.id} editable={profile.version.status === "profile_draft"} session={session} />
      </RoleProfileFrame>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
