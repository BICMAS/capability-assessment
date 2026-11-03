"use client";

import { Search } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { Department, DirectoryPerson, RoleAssignment } from "@/lib/org-directory";
import type { Settled } from "@/lib/safe";

type EmployeeFilter = "all" | "titled" | "untitled";

const pageSize = 12;

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
  const [employeeFilter, setEmployeeFilter] = useState<EmployeeFilter>("all");
  const [page, setPage] = useState(1);
  const departmentList = departments.ok ? departments.value : [];
  const roleList = roles.ok ? roles.value : [];
  const peopleList = people.ok
    ? people.value.filter((person) => !departmentId || person.departmentName === departmentName(departmentList, departmentId))
    : [];

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return peopleList.filter((person) => {
      const titles = titlesFor(roleList, person.id);
      if (employeeFilter === "titled" && titles.length === 0) {
        return false;
      }
      if (employeeFilter === "untitled" && titles.length > 0) {
        return false;
      }
      if (!needle) {
        return true;
      }
      return (
        person.name.toLowerCase().includes(needle) ||
        person.departmentName.toLowerCase().includes(needle) ||
        titles.some((title) => title.toLowerCase().includes(needle))
      );
    });
  }, [employeeFilter, peopleList, query, roleList]);

  const groups = people.ok ? groupPeople(filtered, departmentList, departmentId) : [];
  const flatCount = filtered.length;
  const pageCount = Math.max(1, Math.ceil(flatCount / pageSize));
  const currentPage = Math.min(page, pageCount);
  const pagedGroups = paginateGroups(groups, currentPage, pageSize);

  function updateQuery(value: string) {
    setQuery(value);
    setPage(1);
  }

  function updateEmployeeFilter(value: EmployeeFilter) {
    setEmployeeFilter(value);
    setPage(1);
  }

  return (
    <section className="role-card role-directory" aria-labelledby="directory-heading">
      <header className="role-card-head">
        <h2 id="directory-heading">Organisation directory</h2>
        <p>
          Departments and role titles come from the organisation directory. An organogram records reporting relationships
          separately and is not imported here.
        </p>
      </header>

      {!departments.ok ? <StatusMessage tone="error">{departments.error}</StatusMessage> : null}
      {!people.ok ? <StatusMessage tone="error">{people.error}</StatusMessage> : null}
      {!roles.ok ? <StatusMessage tone="error">{roles.error}</StatusMessage> : null}

      <div className="directory-toolbar">
        <label className="directory-search">
          <Search size={16} aria-hidden="true" />
          <span className="app-clip">Search the directory</span>
          <input
            type="search"
            value={query}
            onChange={(event) => updateQuery(event.target.value)}
            placeholder="Search by name, department, or role title"
          />
        </label>

        <div className="directory-filter-chips" role="group" aria-label="Employee filters">
          {(
            [
              ["all", "All employees"],
              ["titled", "With role title"],
              ["untitled", "Without role title"],
            ] as const
          ).map(([value, label]) => (
            <button
              key={value}
              type="button"
              className={employeeFilter === value ? "is-active" : undefined}
              aria-pressed={employeeFilter === value}
              onClick={() => updateEmployeeFilter(value)}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {departments.ok && departmentList.length === 0 ? <EmptyState>No departments are recorded in the organisation directory.</EmptyState> : null}
      {departments.ok && departmentList.length > 0 ? (
        <nav className="department-pills" aria-label="Departments">
          <Link href={directoryHref("", chartId, entry)} aria-current={departmentId ? undefined : "true"} onClick={() => setPage(1)}>
            All departments
          </Link>
          {departmentList.map((department) => (
            <Link
              key={department.id}
              href={directoryHref(department.id, chartId, entry)}
              aria-current={department.id === departmentId ? "true" : undefined}
              onClick={() => setPage(1)}
            >
              {department.name}
            </Link>
          ))}
        </nav>
      ) : null}

      {people.ok ? (
        flatCount === 0 ? (
          <EmptyState>
            {query.trim() || employeeFilter !== "all"
              ? "No people match these filters."
              : "No people are recorded for this selection."}
          </EmptyState>
        ) : (
          <>
            <div className="directory-summary" aria-live="polite">
              Showing {pageRange(currentPage, pageSize, flatCount)} of {flatCount}{" "}
              {flatCount === 1 ? "person" : "people"}
            </div>
            {pagedGroups.map((group) => (
              <section key={group.name} className="directory-group">
                <h3>{group.name}</h3>
                <div className="directory-table-wrap">
                  <table className="directory-table">
                    <thead>
                      <tr>
                        <th scope="col">Employee</th>
                        <th scope="col">Department</th>
                        <th scope="col">Role title</th>
                        <th scope="col">Role title readiness</th>
                        <th scope="col">Action</th>
                      </tr>
                    </thead>
                    <tbody>
                      {group.people.map((person) => {
                        const titles = titlesFor(roleList, person.id);
                        const ready = titles.length > 0;
                        const primaryTitle = titles[0] ?? "";
                        return (
                          <tr key={person.id}>
                            <td data-label="Employee">
                              <strong>{person.name}</strong>
                            </td>
                            <td data-label="Department">
                              <span className={person.departmentName ? undefined : "directory-muted"}>
                                {person.departmentName || "No department"}
                              </span>
                            </td>
                            <td data-label="Role title">
                              <span className={ready ? undefined : "directory-muted"}>
                                {ready ? titles.join(", ") : "No role title"}
                              </span>
                            </td>
                            <td data-label="Role title readiness">
                              <span className={ready ? "ds-status" : "ds-status ds-status-neutral"}>
                                {ready ? "Has role title" : "Needs role title"}
                              </span>
                            </td>
                            <td data-label="Action">
                              <Link
                                className="directory-row-action"
                                href={
                                  ready
                                    ? `/admin/role-profiles/new?title=${encodeURIComponent(primaryTitle)}`
                                    : "/admin/role-profiles/new"
                                }
                              >
                                {ready ? "Create profile" : "Create without title"}
                              </Link>
                            </td>
                          </tr>
                        );
                      })}
                    </tbody>
                  </table>
                </div>
              </section>
            ))}
            {pageCount > 1 ? (
              <div className="directory-pagination">
                <button type="button" disabled={currentPage <= 1} onClick={() => setPage((value) => Math.max(1, value - 1))}>
                  Previous
                </button>
                <span>
                  Page {currentPage} of {pageCount}
                </span>
                <button
                  type="button"
                  disabled={currentPage >= pageCount}
                  onClick={() => setPage((value) => Math.min(pageCount, value + 1))}
                >
                  Next
                </button>
              </div>
            ) : null}
          </>
        )
      ) : null}
    </section>
  );
}

function titlesFor(roles: RoleAssignment[], personId: string): string[] {
  return roles.filter((role) => role.personId === personId).map((role) => role.title);
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

function paginateGroups(
  groups: { name: string; people: DirectoryPerson[] }[],
  page: number,
  size: number,
): { name: string; people: DirectoryPerson[] }[] {
  const start = (page - 1) * size;
  const end = start + size;
  let index = 0;
  const result: { name: string; people: DirectoryPerson[] }[] = [];
  for (const group of groups) {
    const slice: DirectoryPerson[] = [];
    for (const person of group.people) {
      if (index >= start && index < end) {
        slice.push(person);
      }
      index += 1;
    }
    if (slice.length > 0) {
      result.push({ name: group.name, people: slice });
    }
  }
  return result;
}

function pageRange(page: number, size: number, total: number): string {
  const start = (page - 1) * size + 1;
  const end = Math.min(page * size, total);
  return `${start}–${end}`;
}

function departmentName(departments: Department[], id: string): string {
  return departments.find((department) => department.id === id)?.name ?? "";
}
