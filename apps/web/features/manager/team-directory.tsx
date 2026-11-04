"use client";

import { Search } from "lucide-react";
import Link from "next/link";
import { useMemo, useState } from "react";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { Settled } from "@/lib/safe";

import type { TeamMemberSummary } from "./team-data";
import { memberAttentionLine, needsAttention, summariseAttention, uniqueRoleTitles } from "./team-summary";

type AssessmentFilter = "all" | "assessed" | "unassessed";
type GapFilter = "all" | "priority" | "clear";

const pageSize = 12;

export function TeamDirectory({
  people,
  frameworks,
  members,
}: {
  people: Settled<unknown[]>;
  frameworks: Settled<unknown[]>;
  members: TeamMemberSummary[];
}) {
  const [query, setQuery] = useState("");
  const [roleTitle, setRoleTitle] = useState("all");
  const [assessmentFilter, setAssessmentFilter] = useState<AssessmentFilter>("all");
  const [gapFilter, setGapFilter] = useState<GapFilter>("all");
  const [page, setPage] = useState(1);

  const roleOptions = useMemo(() => uniqueRoleTitles(members), [members]);
  const attention = useMemo(() => summariseAttention(members), [members]);

  const filtered = useMemo(() => {
    const needle = query.trim().toLowerCase();
    return members.filter((member) => {
      if (roleTitle !== "all" && !member.roles.includes(roleTitle)) {
        return false;
      }
      if (assessmentFilter === "assessed" && !(member.levelsOk && member.assessedCount > 0)) {
        return false;
      }
      if (
        assessmentFilter === "unassessed" &&
        !(member.levelsOk && member.requiredCount > 0 && member.assessedCount === 0)
      ) {
        return false;
      }
      if (gapFilter === "priority" && !(member.levelsOk && member.priorityCount > 0)) {
        return false;
      }
      if (gapFilter === "clear" && !(member.levelsOk && member.priorityCount === 0 && member.assessedCount > 0)) {
        return false;
      }
      if (!needle) {
        return true;
      }
      return (
        member.name.toLowerCase().includes(needle) ||
        member.departmentName.toLowerCase().includes(needle) ||
        member.roles.some((role) => role.toLowerCase().includes(needle)) ||
        member.belowNames.some((name) => name.toLowerCase().includes(needle))
      );
    });
  }, [assessmentFilter, gapFilter, members, query, roleTitle]);

  const pageCount = Math.max(1, Math.ceil(filtered.length / pageSize));
  const currentPage = Math.min(page, pageCount);
  const paged = filtered.slice((currentPage - 1) * pageSize, currentPage * pageSize);

  function resetPage() {
    setPage(1);
  }

  return (
    <div className="manager-team">
      <section className="manager-section" aria-labelledby="manager-attention-heading">
        <header className="manager-section-head">
          <h2 id="manager-attention-heading">Team attention</h2>
          <p>Counts use recorded role titles and calculated capability levels for people you can manage.</p>
        </header>

        {!people.ok ? <StatusMessage tone="error">{people.error}</StatusMessage> : null}
        {!frameworks.ok ? <StatusMessage tone="error">{frameworks.error}</StatusMessage> : null}
        {attention.levelFailures > 0 ? (
          <StatusMessage tone="info">
            {attention.levelFailures === 1
              ? "1 team member could not load capability levels."
              : `${attention.levelFailures} team members could not load capability levels.`}
          </StatusMessage>
        ) : null}

        {people.ok ? (
          <ul className="manager-kpi-grid">
            <li className="manager-kpi">
              <span className="manager-kpi-label">Team members</span>
              <strong className="manager-kpi-value">{String(attention.members)}</strong>
              <span className="manager-kpi-detail">People in your managerOf list</span>
            </li>
            <li className="manager-kpi">
              <span className="manager-kpi-label">Needs attention</span>
              <strong className="manager-kpi-value">{String(attention.needsAttention)}</strong>
              <span className="manager-kpi-detail">Priority gaps, developing areas, or not yet assessed</span>
            </li>
            <li className="manager-kpi">
              <span className="manager-kpi-label">Priority gaps</span>
              <strong className="manager-kpi-value">{String(attention.withPriorityGaps)}</strong>
              <span className="manager-kpi-detail">People with at least one priority gap</span>
            </li>
            <li className="manager-kpi">
              <span className="manager-kpi-label">Not yet assessed</span>
              <strong className="manager-kpi-value">{String(attention.notYetAssessed)}</strong>
              <span className="manager-kpi-detail">Requirements present, no calculated levels yet</span>
            </li>
          </ul>
        ) : null}
      </section>

      <section className="manager-card manager-directory" aria-labelledby="manager-directory-heading">
        <header className="manager-card-head">
          <div>
            <h2 id="manager-directory-heading">My team</h2>
            <p>Search and filter your team, then open a person record for full capability detail and assessments.</p>
          </div>
        </header>

        <div className="directory-toolbar">
          <label className="directory-search">
            <Search size={16} aria-hidden="true" />
            <span className="app-clip">Search the team</span>
            <input
              type="search"
              value={query}
              onChange={(event) => {
                setQuery(event.target.value);
                resetPage();
              }}
              placeholder="Search by name, department, role, or capability"
            />
          </label>

          <label className="manager-select">
            <span>Role title</span>
            <select
              value={roleTitle}
              onChange={(event) => {
                setRoleTitle(event.target.value);
                resetPage();
              }}
            >
              <option value="all">All role titles</option>
              {roleOptions.map((title) => (
                <option key={title} value={title}>
                  {title}
                </option>
              ))}
            </select>
          </label>

          <div className="directory-filter-chips" role="group" aria-label="Assessment filters">
            {(
              [
                ["all", "All"],
                ["assessed", "Assessed"],
                ["unassessed", "Not yet assessed"],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={assessmentFilter === value ? "is-active" : undefined}
                aria-pressed={assessmentFilter === value}
                onClick={() => {
                  setAssessmentFilter(value);
                  resetPage();
                }}
              >
                {label}
              </button>
            ))}
          </div>

          <div className="directory-filter-chips" role="group" aria-label="Priority gap filters">
            {(
              [
                ["all", "All gaps"],
                ["priority", "Has priority gaps"],
                ["clear", "No priority gaps"],
              ] as const
            ).map(([value, label]) => (
              <button
                key={value}
                type="button"
                className={gapFilter === value ? "is-active" : undefined}
                aria-pressed={gapFilter === value}
                onClick={() => {
                  setGapFilter(value);
                  resetPage();
                }}
              >
                {label}
              </button>
            ))}
          </div>
        </div>

        {people.ok && members.length === 0 ? <EmptyState>No people are in your team list yet.</EmptyState> : null}

        {people.ok && members.length > 0 && filtered.length === 0 ? (
          <EmptyState>No people match these filters.</EmptyState>
        ) : null}

        {people.ok && paged.length > 0 ? (
          <>
            <div className="directory-summary" aria-live="polite">
              Showing {pageRange(currentPage, pageSize, filtered.length)} of {filtered.length}{" "}
              {filtered.length === 1 ? "person" : "people"}
            </div>

            <div className="directory-table-wrap manager-table-wrap">
              <table className="directory-table manager-table">
                <thead>
                  <tr>
                    <th scope="col">Team member</th>
                    <th scope="col">Department</th>
                    <th scope="col">Role title</th>
                    <th scope="col">Attention</th>
                    <th scope="col">Action</th>
                  </tr>
                </thead>
                <tbody>
                  {paged.map((member) => {
                    const attentionNeeded = needsAttention(member);
                    return (
                      <tr key={member.personId}>
                        <td data-label="Team member">
                          <strong>{member.name}</strong>
                        </td>
                        <td data-label="Department">
                          <span className={member.departmentName ? undefined : "directory-muted"}>
                            {member.departmentName || "No department"}
                          </span>
                        </td>
                        <td data-label="Role title">
                          <span className={member.roles.length > 0 ? undefined : "directory-muted"}>
                            {member.roles.length > 0 ? member.roles.join(", ") : "No role recorded"}
                          </span>
                        </td>
                        <td data-label="Attention">
                          <div className="manager-attention-cell">
                            <span
                              className={
                                !member.levelsOk
                                  ? "ds-status ds-status-attention"
                                  : attentionNeeded
                                    ? member.priorityCount > 0
                                      ? "ds-status ds-status-attention"
                                      : "ds-status ds-status-neutral"
                                    : "ds-status"
                              }
                            >
                              {!member.levelsOk
                                ? "Unavailable"
                                : member.priorityCount > 0
                                  ? "Priority gaps"
                                  : member.assessedCount === 0 && member.requiredCount > 0
                                    ? "Not yet assessed"
                                    : attentionNeeded
                                      ? "Developing"
                                      : "On track"}
                            </span>
                            <span className="manager-attention-detail">{memberAttentionLine(member)}</span>
                          </div>
                        </td>
                        <td data-label="Action">
                          <Link className="directory-row-action" href={`/manager/people/${member.personId}`}>
                            Open record
                          </Link>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>

            <ul className="manager-card-list" aria-label="Team members">
              {paged.map((member) => {
                const attentionNeeded = needsAttention(member);
                return (
                  <li key={`card-${member.personId}`}>
                    <article className="manager-person-card">
                      <header className="manager-person-card-head">
                        <div>
                          <h3>{member.name}</h3>
                          <p className="manager-meta">
                            {member.departmentName || "No department"}
                            {" · "}
                            {member.roles.length > 0 ? member.roles.join(", ") : "No role recorded"}
                          </p>
                        </div>
                        <span
                          className={
                            !member.levelsOk
                              ? "ds-status ds-status-attention"
                              : attentionNeeded
                                ? member.priorityCount > 0
                                  ? "ds-status ds-status-attention"
                                  : "ds-status ds-status-neutral"
                                : "ds-status"
                          }
                        >
                          {!member.levelsOk
                            ? "Unavailable"
                            : member.priorityCount > 0
                              ? "Priority gaps"
                              : member.assessedCount === 0 && member.requiredCount > 0
                                ? "Not yet assessed"
                                : attentionNeeded
                                  ? "Developing"
                                  : "On track"}
                        </span>
                      </header>
                      <p className="manager-attention-detail">{memberAttentionLine(member)}</p>
                      <Link className="directory-row-action" href={`/manager/people/${member.personId}`}>
                        Open record
                      </Link>
                    </article>
                  </li>
                );
              })}
            </ul>

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
        ) : null}
      </section>
    </div>
  );
}

function pageRange(page: number, size: number, total: number): string {
  if (total === 0) {
    return "0";
  }
  const start = (page - 1) * size + 1;
  const end = Math.min(page * size, total);
  return start === end ? String(start) : `${start}–${end}`;
}
