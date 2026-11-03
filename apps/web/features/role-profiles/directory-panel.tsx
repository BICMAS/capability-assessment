"use client";

import Link from "next/link";
import { useMemo, useState } from "react";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { Department, DirectoryPerson, RoleAssignment } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

export function DirectoryPanel({
  departmentId,
  chartId,
  entry,
  departments,
  people,
  roles,
}: {
  departmentId: string;
  chartId: string;
  entry: string;
  departments: Settled<Department[]>;
  people: Settled<DirectoryPerson[]>;
  roles: Settled<RoleAssignment[]>;
}) {
  const [query, setQuery] = useState("");
  const departmentList = departments.ok ? departments.value : [];
  const roleList = roles.ok ? roles.value : [];
  const peopleList = people.ok
    ? people.value.filter((person) => !departmentId || person.departmentName === departmentName(departmentList, departmentId))
    : [];
  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return peopleList;
    }
    return peopleList.filter((person) => {
      const titles = roleList.filter((role) => role.personId === person.id).map((role) => role.title);
      return (
        person.name.toLowerCase().includes(needle) ||
        person.departmentName.toLowerCase().includes(needle) ||
        titles.some((title) => title.toLowerCase().includes(needle))
      );
    });
  }, [peopleList, query, roleList]);
  const groups = people.ok ? groupPeople(filtered, departmentList, departmentId) : [];

  return (
    <section className="role-card role-directory" aria-labelledby="directory-heading">
      <header className="role-card-head">
        <h2 id="directory-heading">Organisation directory</h2>
        <p>Search departments, employees, and role titles. The directory does not import an organogram.</p>
      </header>

      {!departments.ok ? <StatusMessage tone="error">{departments.error}</StatusMessage> : null}
      {!people.ok ? <StatusMessage tone="error">{people.error}</StatusMessage> : null}
      {!roles.ok ? <StatusMessage tone="error">{roles.error}</StatusMessage> : null}

      <label className="role-directory-search">
        Search the directory
        <input
          type="search"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder="Name, department, or role title"
        />
      </label>

      {departments.ok && departmentList.length === 0 ? <EmptyState>No departments are recorded.</EmptyState> : null}
      {departments.ok && departmentList.length > 0 ? (
        <nav className="department-pills" aria-label="Departments">
          <Link href={directoryHref("", chartId, entry)} aria-current={departmentId ? undefined : "true"}>
            All departments
          </Link>
          {departmentList.map((department) => (
            <Link
              key={department.id}
              href={directoryHref(department.id, chartId, entry)}
              aria-current={department.id === departmentId ? "true" : undefined}
            >
              {department.name}
            </Link>
          ))}
        </nav>
      ) : null}

      {people.ok ? (
        filtered.length === 0 ? (
          <EmptyState>{query.trim() ? "No people match this search." : "No people are recorded for this selection."}</EmptyState>
        ) : (
          groups.map((group) => (
            <section key={group.name} className="directory-group">
              <h3>{group.name}</h3>
              <ul className="role-people-list">
                {group.people.map((person) => {
                  const titles = roleList.filter((role) => role.personId === person.id).map((role) => role.title);
                  return (
                    <li key={person.id}>
                      <strong>{person.name}</strong>
                      <span className={titles.length > 0 ? undefined : "directory-muted"}>
                        {titles.length > 0 ? titles.join(", ") : "No role title"}
                      </span>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))
        )
      ) : null}
    </section>
  );
}

function directoryHref(departmentId: string, chartId: string, entry: string): string {
  const query = new URLSearchParams();
  if (departmentId) {
    query.set("department", departmentId);
  }
  if (chartId) {
    query.set("chart", chartId);
  }
  if (entry) {
    query.set("entry", entry);
  }
  const value = query.toString();
  return value ? `/admin/role-profiles?${value}` : "/admin/role-profiles";
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
