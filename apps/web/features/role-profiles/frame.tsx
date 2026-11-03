import type { ReactNode } from "react";

import { SetBreadcrumb } from "@/components/shell-breadcrumb";
import { StatusMessage } from "@/components/ui/status-message";
import type { RoleProfile } from "@/lib/role-profile";
import { listOrganograms } from "@/lib/org-directory";
import { settle } from "@/lib/safe";

import { WorkflowStepper, type WorkflowStep } from "./stepper";
import { workflowStages, type WorkflowEvidence } from "./workflow";

export function RoleProfileFrame({
  profile,
  current,
  error = "",
  notice = "",
  evidence,
  children,
}: {
  profile?: RoleProfile;
  current: WorkflowStep;
  error?: string;
  notice?: string;
  evidence?: WorkflowEvidence;
  children: ReactNode;
}) {
  const stage = workflowStages.find((item) => item.id === current);
  const crumbs = [
    { href: "/admin", label: "Home" },
    { href: "/admin/role-profiles", label: "Role skills profiles" },
    ...(profile ? [{ label: profile.roleName }] : []),
    ...(stage ? [{ label: stage.title }] : []),
  ];

  return (
    <div className="role-profile">
      <SetBreadcrumb items={crumbs} />
      {!profile ? (
        <p className="role-page-lead">Create and manage role skills profiles for your organisation.</p>
      ) : null}
      <WorkflowStepper profile={profile} current={current} evidence={evidence} />
      <StatusMessage tone="error">{error}</StatusMessage>
      <StatusMessage tone="success">{notice}</StatusMessage>
      {profile ? <h2>{profile.roleName}</h2> : null}
      {children}
    </div>
  );
}

export async function WorkflowShell({
  token,
  profile,
  current,
  error = "",
  notice = "",
  clarificationComplete = null,
  children,
}: {
  token: string;
  profile?: RoleProfile;
  current: WorkflowStep;
  error?: string;
  notice?: string;
  clarificationComplete?: boolean | null;
  children: ReactNode;
}) {
  const charts = await settle(listOrganograms(token), "Organograms");
  return (
    <RoleProfileFrame
      profile={profile}
      current={current}
      error={error}
      notice={notice}
      evidence={{ organogramCount: charts.ok ? charts.value.length : null, clarificationComplete }}
    >
      {children}
    </RoleProfileFrame>
  );
}
