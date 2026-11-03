import { StatusMessage } from "@/components/ui/status-message";
import type { Department, DirectoryPerson, RoleAssignment } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

import { DirectoryPanel } from "./directory-panel";
import { OpenProfileCard } from "./open-profile-card";

export function Organogram({
  departmentId,
  chartId = "",
  entry = "build",
  departments,
  people,
  roles,
  error,
  openError = "",
}: {
  departmentId: string;
  chartId?: string;
  entry?: string;
  departments: Settled<Department[]>;
  people: Settled<DirectoryPerson[]>;
  roles: Settled<RoleAssignment[]>;
  error: string;
  openError?: string;
}) {
  return (
    <div className="role-lower-grid">
      <StatusMessage tone="error">{error}</StatusMessage>
      <DirectoryPanel
        departmentId={departmentId}
        chartId={chartId}
        entry={entry}
        departments={departments}
        people={people}
        roles={roles}
      />
      <OpenProfileCard openError={openError} />
    </div>
  );
}
