import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { WorkflowShell } from "@/features/role-profiles/frame";
import { InsightsPanel } from "@/features/role-profiles/insights-panel";
import { InsightWorkspace } from "@/features/role-profiles/insight-workspace";
import { ProfileLoadError } from "@/features/role-profiles/load-error";
import { listLearningRecommendations, listLearningResources } from "@/lib/capability-insights";
import { readProfileInsights } from "@/lib/profile-insights";
import { readRoleProfile } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function InsightsPage({
  params,
  searchParams,
}: {
  params: Promise<{ profileId: string }>;
  searchParams: Promise<{ error?: string; notice?: string; person?: string; department?: string; role?: string; q?: string; from?: string; to?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const { profileId } = await params;
  const query = await searchParams;
  try {
    const profile = await readRoleProfile(token, profileId);
    const insight =
      profile.version.status === "profile_published"
        ? await settle(
            readProfileInsights(token, {
              profileVersionId: profile.version.id,
              personId: query.person,
              department: query.department,
              role: query.role,
              q: query.q,
              from: query.from,
              to: query.to,
            }),
            "Insights",
          )
        : null;
    const resources = await settle(listLearningResources(token), "The learning catalogue");
    const recommendations = await settle(listLearningRecommendations(token, ""), "Learning recommendations");
    return (
      <WorkflowShell token={token} profile={profile} current="insights" error={query.error ?? ""} notice={query.notice ?? ""}>
        <InsightsPanel profile={profile} insight={insight} />
        <InsightWorkspace
          profile={profile}
          insight={insight}
          resources={resources}
          recommendations={recommendations}
          filters={{
            person: query.person ?? "",
            department: query.department ?? "",
            role: query.role ?? "",
            q: query.q ?? "",
            from: query.from ?? "",
            to: query.to ?? "",
          }}
        />
      </WorkflowShell>
    );
  } catch (error) {
    return <ProfileLoadError error={error} />;
  }
}
