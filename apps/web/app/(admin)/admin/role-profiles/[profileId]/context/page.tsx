import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { ContextForm } from "@/features/role-profiles/context-form";
import { RoleProfileFrame } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { readRoleProfile } from "@/lib/role-profile";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ContextPage({
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
    return (
      <RoleProfileFrame profile={profile} current="context" error={query.error ?? ""} notice={query.notice ?? ""}>
        <ContextForm profile={profile} />
      </RoleProfileFrame>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
