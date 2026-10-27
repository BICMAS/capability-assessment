import type { ReactNode } from "react";

import { StatusMessage } from "@/components/ui/status-message";
import type { RoleProfile } from "@/lib/role-profile";

import { WorkflowStepper, type WorkflowStep } from "./stepper";

export function RoleProfileFrame({
  profile,
  current,
  error = "",
  notice = "",
  children,
}: {
  profile?: RoleProfile;
  current: WorkflowStep;
  error?: string;
  notice?: string;
  children: ReactNode;
}) {
  return (
    <div className="role-profile">
      <WorkflowStepper profileId={profile?.id} current={current} />
      <h2>{profile ? profile.roleName : "Role skills profiles"}</h2>
      {error ? <StatusMessage tone="error">{error}</StatusMessage> : null}
      {notice ? <StatusMessage tone="success">{notice}</StatusMessage> : null}
      {children}
    </div>
  );
}
