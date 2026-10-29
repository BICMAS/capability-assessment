import Link from "next/link";
import type { ReactNode } from "react";

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
  return (
    <div className="role-profile">
      <nav className="breadcrumb" aria-label="Breadcrumb">
        <ol>
          <li>
            <Link href="/admin">Home</Link>
          </li>
          <li>
            <Link href="/admin/role-profiles">Role skills profiles</Link>
          </li>
          {profile ? <li>{profile.roleName}</li> : null}
          <li aria-current="page">{stage?.title}</li>
        </ol>
      </nav>
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
