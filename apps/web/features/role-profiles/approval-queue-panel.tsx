"use client";

import { Search } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";

import { EmptyState } from "@/components/ui/status-message";
import type { ApprovalQueueItem } from "@/lib/role-profile";

const views = [
  { id: "awaiting", label: "Awaiting my review", href: "/admin/role-profiles/review" },
  { id: "submitted", label: "Submitted by me", href: "/admin/role-profiles/review?view=submitted" },
  { id: "reviewed", label: "Previously reviewed", href: "/admin/role-profiles/review?view=reviewed" },
] as const;

export function ApprovalQueuePanel({
  view,
  items,
  personNames,
}: {
  view: "awaiting" | "submitted" | "reviewed";
  items: ApprovalQueueItem[];
  personNames: Map<string, string>;
}) {
  const [query, setQuery] = useState("");
  const [department, setDepartment] = useState("");
  const [status, setStatus] = useState("");

  const departments = useMemo(
    () =>
      Array.from(new Set(items.map((item) => item.department.trim()).filter((value) => value !== ""))).sort((left, right) =>
        left.localeCompare(right),
      ),
    [items],
  );
  const statuses = useMemo(
    () => Array.from(new Set(items.map((item) => item.status).filter((value) => value !== ""))).sort((left, right) => left.localeCompare(right)),
    [items],
  );

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return items.filter((item) => {
      if (department && item.department !== department) {
        return false;
      }
      if (status && item.status !== status) {
        return false;
      }
      if (!needle) {
        return true;
      }
      const submitter = personLabel(personNames, item.submittedByPersonId, "").toLowerCase();
      const assignee = personLabel(personNames, item.assigneePersonId, "").toLowerCase();
      return (
        item.roleName.toLowerCase().includes(needle) ||
        item.department.toLowerCase().includes(needle) ||
        item.stageLabel.toLowerCase().includes(needle) ||
        item.status.toLowerCase().includes(needle) ||
        item.submittedByPersonId.toLowerCase().includes(needle) ||
        item.assigneePersonId.toLowerCase().includes(needle) ||
        submitter.includes(needle) ||
        assignee.includes(needle) ||
        item.outstanding.toLowerCase().includes(needle)
      );
    });
  }, [department, items, personNames, query, status]);

  return (
    <section className="role-card approval-manage" aria-labelledby="approval-manage-heading">
      <header className="role-card-head">
        <h2 id="approval-manage-heading">Approval management</h2>
        <p>Search and filter the live approval queue for this organisation. Records come from the capability service.</p>
      </header>

      <nav className="ds-tabs approval-tabs" aria-label="Approval views">
        {views.map((item) => (
          <Link key={item.id} href={item.href} aria-current={view === item.id ? "page" : undefined}>
            {item.label}
          </Link>
        ))}
      </nav>

      <div className="approval-toolbar">
        <label className="directory-search">
          <Search size={16} aria-hidden="true" />
          <span className="app-clip">Search approvals</span>
          <input
            type="search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search role, department, stage, or status"
          />
        </label>
        <label>
          Department
          <select value={department} onChange={(event) => setDepartment(event.target.value)}>
            <option value="">All departments</option>
            {departments.map((name) => (
              <option key={name} value={name}>
                {name}
              </option>
            ))}
          </select>
        </label>
        <label>
          Status
          <select value={status} onChange={(event) => setStatus(event.target.value)}>
            <option value="">All statuses</option>
            {statuses.map((name) => (
              <option key={name} value={name}>
                {formatStatus(name)}
              </option>
            ))}
          </select>
        </label>
      </div>

      {filtered.length === 0 ? (
        <EmptyState>
          {items.length === 0
            ? emptyCopy(view)
            : "No approval records match these filters."}
        </EmptyState>
      ) : (
        <div className="directory-table-wrap">
          <table className="directory-table approval-table">
            <thead>
              <tr>
                <th scope="col">Role title</th>
                <th scope="col">Department</th>
                <th scope="col">Submitted by</th>
                <th scope="col">Submission date</th>
                <th scope="col">Current approver</th>
                <th scope="col">Approval stage</th>
                <th scope="col">Status</th>
                <th scope="col">Action</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((item) => (
                <tr key={item.versionId}>
                  <td data-label="Role title">
                    <strong>{item.roleName}</strong>
                    <span className="directory-muted">Version {item.versionNumber}</span>
                  </td>
                  <td data-label="Department">
                    <span className={item.department ? undefined : "directory-muted"}>
                      {item.department || "Not recorded"}
                    </span>
                  </td>
                  <td data-label="Submitted by">
                    <span className={item.submittedByPersonId ? undefined : "directory-muted"}>
                      {personLabel(personNames, item.submittedByPersonId, "Not recorded")}
                    </span>
                  </td>
                  <td data-label="Submission date">{formatDate(item.submittedAt)}</td>
                  <td data-label="Current approver">
                    <span className={item.assigneePersonId ? undefined : "directory-muted"}>
                      {personLabel(personNames, item.assigneePersonId, "Not assigned")}
                    </span>
                  </td>
                  <td data-label="Approval stage">
                    <span className={item.stageLabel ? undefined : "directory-muted"}>
                      {item.stageLabel || item.workflowStage || "Not started"}
                    </span>
                    {item.outstanding ? <span className="directory-muted">Outstanding: {item.outstanding}</span> : null}
                  </td>
                  <td data-label="Status">
                    <span className={statusClass(item.status)}>{formatStatus(item.status)}</span>
                  </td>
                  <td data-label="Action">
                    <Link className="directory-row-action" href={`/admin/role-profiles/${item.profileId}/review`}>
                      View
                    </Link>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

function personLabel(names: Map<string, string>, personId: string, empty: string): string {
  if (!personId) {
    return empty;
  }
  return names.get(personId) || personId;
}

function emptyCopy(view: "awaiting" | "submitted" | "reviewed"): string {
  if (view === "submitted") {
    return "You have not submitted a profile for review.";
  }
  if (view === "reviewed") {
    return "No previously reviewed profiles yet.";
  }
  return "No profiles are awaiting your review.";
}

function formatDate(value: string | null): string {
  if (!value) {
    return "Not recorded";
  }
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }
  return new Intl.DateTimeFormat("en-GB", { day: "2-digit", month: "short", year: "numeric" }).format(date);
}

function formatStatus(status: string): string {
  return status
    .split("_")
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

function statusClass(status: string): string {
  if (status.includes("approved") || status.includes("published")) {
    return "ds-status";
  }
  if (status.includes("reject") || status.includes("chang")) {
    return "ds-status ds-status-attention";
  }
  return "ds-status ds-status-neutral";
}
