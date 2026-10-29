import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { GenerationPanel } from "@/features/role-profiles/generation-panel";
import { SkillsWorkspace } from "@/features/role-profiles/skills-workspace";
import { WorkflowShell } from "@/features/role-profiles/frame";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { readClarification, readRoleProfile, readRoleProfileJob } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function GenerationPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string; job?: string }>;
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
    const job = query.job ? await settle(readRoleProfileJob(token, query.job), "Generation job") : null;
    return (
      <WorkflowShell
        token={token}
        profile={profile}
        current="generate"
        error={query.error ?? ""}
        notice={query.notice ?? ""}
        clarificationComplete={session.ok ? session.value.complete : null}
      >
        <SkillsWorkspace profile={profile} jobModel={job?.ok ? job.value.modelId : profile.version.ai.model}>
          <GenerationPanel
            profileId={profile.id}
            versionId={profile.version.id}
            contextVersion={session.ok ? session.value.contextVersion : 0}
            idempotencyKey={crypto.randomUUID()}
            editable={profile.version.status === "profile_draft"}
            hasDraft={profile.version.skills.length > 0 || Boolean(profile.version.ai.generatedAt)}
            job={job}
          />
        </SkillsWorkspace>
      </WorkflowShell>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
