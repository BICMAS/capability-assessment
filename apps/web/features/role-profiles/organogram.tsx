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
  const groups = people.ok ? groupPeople(peopleList, departmentList, departmentId) : [];

  return (
    <div className="directory-layout">
      <StatusMessage tone="error">{error}</StatusMessage>
      <section className="directory-people">
        <h2>Directory</h2>
        <p className="portal-note">The organisation directory is the source for departments and role titles. It does not provide an organogram import.</p>
        {!departments.ok ? <StatusMessage tone="error">{departments.error}</StatusMessage> : null}
        {!people.ok ? <StatusMessage tone="error">{people.error}</StatusMessage> : null}
        {!roles.ok ? <StatusMessage tone="error">{roles.error}</StatusMessage> : null}
        {departments.ok && departmentList.length === 0 ? <EmptyState>No departments are recorded.</EmptyState> : null}
        {departments.ok && departmentList.length > 0 ? (
          <nav className="department-pills" aria-label="Departments">
            <Link href="/admin/role-profiles" aria-current={departmentId ? undefined : "true"}>
              All departments
            </Link>
            {departmentList.map((department) => (
              <Link
                key={department.id}
                href={`/admin/role-profiles?department=${encodeURIComponent(department.id)}`}
                aria-current={department.id === departmentId ? "true" : undefined}
              >
                {department.name}
              </Link>
            ))}
          </nav>
        ) : null}
        {people.ok ? (
          peopleList.length === 0 ? (
            <EmptyState>No people are recorded for this selection.</EmptyState>
          ) : (
            groups.map((group) => (
              <section key={group.name} className="directory-group">
                <h3>{group.name}</h3>
                <ul className="organogram">
                  {group.people.map((person) => {
                    const titles = roleList.filter((role) => role.personId === person.id).map((role) => role.title);
                    return (
                      <li key={person.id}>
                        <strong>{person.name}</strong>
                        <span className={titles.length > 0 ? undefined : "directory-muted"}>{titles.length > 0 ? titles.join(", ") : "No role title"}</span>
                      </li>
                    );
                  })}
                </ul>
              </section>
            ))
          )
        ) : null}
      </section>
      <aside className="directory-actions">
        <h2>Open a profile</h2>
        <p className="portal-note">The capability service does not list every role profile. Open one by its id, or create a new draft.</p>
        <Link className="learn-link directory-create" href="/admin/role-profiles/new">
          Create a role profile
        </Link>
        <Link className="learn-link directory-secondary" href="/admin/role-profiles/review">
          Review inbox
        </Link>
        <form action={openRoleProfileForm} className="framework-form">
          <label>
            Profile id
            <input name="profileId" required />
          </label>
          <PendingButton>Open profile</PendingButton>
        </form>
      </aside>
    </div>
  );
}

function groupPeople(people: DirectoryPerson[], departments: Department[], departmentId: string): { name: string; people: DirectoryPerson[] }[] {
  if (departmentId) {
    return [{ name: departmentName(departments, departmentId) || "Department", people }];
  }
  const groups = departments
    .map((department) => ({
      name: department.name,
      people: people.filter((person) => person.departmentName === department.name),
    }))
    .filter((group) => group.people.length > 0);
  const placed = new Set(groups.flatMap((group) => group.people.map((person) => person.id)));
  const unassigned = people.filter((person) => !placed.has(person.id));
  if (unassigned.length > 0) {
    groups.push({ name: "No department", people: unassigned });
  }
  return groups;
}

function departmentName(departments: Department[], id: string): string {
  return departments.find((department) => department.id === id)?.name ?? "";
}
