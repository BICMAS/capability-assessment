import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { AssessmentSession } from "@/lib/generated/assessment-plan/plans";
import type { Settled } from "@/lib/safe";

import { needsAttention, type CapabilityRow } from "./capability-data";

export function EmployeeDashboard({
  name,
  roleLine,
  sessions,
  capabilities,
  assessmentError = "",
  assessmentNotice = "",
}: {
  name: string;
  roleLine: string;
  sessions: Settled<AssessmentSession[]>;
  capabilities: CapabilityRow[];
  assessmentError?: string;
  assessmentNotice?: string;
}) {
  const waitingCount = sessions.ok ? sessions.value.filter((session) => session.status !== "submitted").length : null;
  const attentionAll = capabilities.filter(needsAttention);
  const attention = attentionAll.slice(0, 5);
  const pathwaysAll = capabilities.filter((row) => row.pathway !== null);
  const pathways = pathwaysAll.slice(0, 5);

  return (
    <div className="employee-dashboard">
      <header className="employee-intro">
        <div>
          <h2 className="employee-name">{name}</h2>
          {roleLine ? <p className="status-pill">{roleLine}</p> : <p className="portal-note">No role is recorded yet.</p>}
        </div>
        <p className="employee-lead">Focus on waiting assessments, capability gaps, and active pathways.</p>
      </header>

      {assessmentError ? <StatusMessage tone="error">{assessmentError}</StatusMessage> : null}
      {assessmentNotice ? <StatusMessage tone="success">{assessmentNotice}</StatusMessage> : null}

      <section className="employee-section" aria-labelledby="employee-overview-heading">
        <div className="employee-section-head">
          <h2 id="employee-overview-heading">Overview</h2>
        </div>
        <ul className="employee-kpi-grid">
          <li>
            <Link className="employee-kpi" href="/employee#assessments">
              <span className="employee-kpi-label">Waiting assessments</span>
              <strong className={!sessions.ok ? "employee-kpi-value is-error" : "employee-kpi-value"}>
                {sessions.ok ? String(waitingCount) : "—"}
              </strong>
              <span className="employee-kpi-detail">
                {!sessions.ok
                  ? sessions.error
                  : waitingCount === 0
                    ? "Nothing waiting for you"
                    : waitingCount === 1
                      ? "1 session needs answers"
                      : `${waitingCount} sessions need answers`}
              </span>
              <span className="employee-kpi-action">Open assessments</span>
            </Link>
          </li>
          <li>
            <Link className="employee-kpi" href="/employee#capabilities">
              <span className="employee-kpi-label">Needs attention</span>
              <strong className="employee-kpi-value">{String(attentionAll.length)}</strong>
              <span className="employee-kpi-detail">
                {attentionAll.length === 0
                  ? "No priority or unassessed capabilities in this view"
                  : "Gaps and unassessed capabilities from your role titles"}
              </span>
              <span className="employee-kpi-action">Review capabilities</span>
            </Link>
          </li>
          <li>
            <div className="employee-kpi">
              <span className="employee-kpi-label">Active pathways</span>
              <strong className="employee-kpi-value">{String(pathwaysAll.length)}</strong>
              <span className="employee-kpi-detail">
                {pathwaysAll.length === 0 ? "No development pathways are recorded yet" : "Pathways linked to your capabilities"}
              </span>
            </div>
          </li>
        </ul>
      </section>

      <div className="employee-overview-split">
        <section className="employee-card" aria-labelledby="employee-attention-heading">
          <header className="employee-card-head">
            <h3 id="employee-attention-heading">Capabilities needing attention</h3>
          </header>
          {attention.length === 0 ? <EmptyState>You have no priority gaps or unassessed capabilities in this list.</EmptyState> : null}
          {attention.length > 0 ? (
            <ul className="employee-attention-list">
              {attention.map((row) => (
                <li key={row.id}>
                  <strong>{row.name}</strong>
                  <span>
                    {row.frameworkName} v{row.versionNumber}
                    {row.explanation ? ` · Gap ${row.explanation.gap}` : " · Not assessed"}
                  </span>
                </li>
              ))}
            </ul>
          ) : null}
        </section>

        <section className="employee-card" aria-labelledby="employee-pathways-heading">
          <header className="employee-card-head">
            <h3 id="employee-pathways-heading">Pathways</h3>
          </header>
          {pathways.length === 0 ? <EmptyState>No pathways are available for your current capabilities.</EmptyState> : null}
          {pathways.length > 0 ? (
            <ul className="employee-attention-list">
              {pathways.map((row) => (
                <li key={`pathway-${row.id}`}>
                  <strong>{row.name}</strong>
                  <span>
                    {row.pathway?.items.length ?? 0}{" "}
                    {(row.pathway?.items.length ?? 0) === 1 ? "step" : "steps"} on the development pathway
                  </span>
                </li>
              ))}
            </ul>
          ) : null}
        </section>
      </div>
    </div>
  );
}
