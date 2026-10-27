import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { Department, DirectoryPerson, RoleAssignment } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

import { openRoleProfileForm } from "./actions";

export function Organogram({
  departmentId,
  departments,
  people,
  roles,
  error,
}: {
  departmentId: string;
  departments: Settled<Department[]>;
  people: Settled<DirectoryPerson[]>;
  roles: Settled<RoleAssignment[]>;
  error: string;
}) {
  const departmentList = departments.ok ? departments.value : [];
  const peopleList = people.ok ? people.value.filter((person) => !departmentId || person.departmentName === departmentName(departmentList, departmentId)) : [];
  const roleList = roles.ok ? roles.value : [];

  return (
    <div className="directory">
      <StatusMessage tone="error">{error}</StatusMessage>
      <section>
        <h2>Organogram</h2>
        <p>The organisation directory is the source for departments and role titles. It does not provide an organogram import.</p>
        {!departments.ok ? <StatusMessage tone="error">{departments.error}</StatusMessage> : null}
        {!people.ok ? <StatusMessage tone="error">{people.error}</StatusMessage> : null}
        {!roles.ok ? <StatusMessage tone="error">{roles.error}</StatusMessage> : null}
        {departments.ok && departmentList.length === 0 ? <EmptyState>No departments are recorded.</EmptyState> : null}
        {departments.ok && departmentList.length > 0 ? (
          <ul className="workflow-choices">
            <li>
              <Link href="/admin/role-profiles">All departments</Link>
            </li>
            {departmentList.map((department) => (
              <li key={department.id}>
                <Link href={`/admin/role-profiles?department=${encodeURIComponent(department.id)}`} aria-current={department.id === departmentId ? "true" : undefined}>
                  {department.name}
                </Link>
              </li>
            ))}
          </ul>
        ) : null}
        {people.ok ? (
          peopleList.length === 0 ? (
            <EmptyState>No people are recorded for this selection.</EmptyState>
          ) : (
            <ul className="organogram">
              {peopleList.map((person) => {
                const titles = roleList.filter((role) => role.personId === person.id).map((role) => role.title);
                return (
                  <li key={person.id}>
                    <strong>{person.name}</strong>
                    <span>{person.departmentName || "No department"}</span>
                    <span>{titles.length > 0 ? titles.join(", ") : "No role title"}</span>
                  </li>
                );
              })}
            </ul>
          )
        ) : null}
      </section>
      <section className="framework-form">
        <h2>Open a profile</h2>
        <p>The capability service does not list every role profile. Open one by its id, or create a new draft.</p>
        <form action={openRoleProfileForm}>
          <label>
            Profile id
            <input name="profileId" required />
          </label>
          <PendingButton>Open profile</PendingButton>
        </form>
        <Link className="learn-link" href="/admin/role-profiles/new">
          Create a role profile
        </Link>
        <Link className="learn-link" href="/admin/role-profiles/review">
          Review inbox
        </Link>
      </section>
    </div>
  );
}

function departmentName(departments: Department[], id: string): string {
  return departments.find((department) => department.id === id)?.name ?? "";
}
