import {
  Building2,
  CheckCheck,
  ClipboardList,
  FilePlus2,
  Layers3,
  LogIn,
  Network,
  Upload,
  UserCheck,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { AuditRecord } from "@/lib/audit";
import type { Settled } from "@/lib/safe";

import { activityStatus, describeActivity } from "./activity";

export type HomeMetric = {
  label: string;
  href: string;
  action: string;
  value: string;
  detail: string;
  tone?: "error";
  icon: "frameworks" | "assignments" | "plans" | "organisation";
};

export type ApprovalSummary = {
  awaiting: Settled<number>;
  submitted: Settled<number>;
  reviewed: Settled<number>;
};

export function HomeDashboard({
  userName,
  metrics,
  records,
  actorNames,
  showAllRecords,
  totalRecords,
  approvals,
}: {
  userName: string;
  metrics: HomeMetric[];
  records: Settled<AuditRecord[]>;
  actorNames: Map<string, string>;
  showAllRecords: boolean;
  totalRecords: number;
  approvals: ApprovalSummary;
}) {
  const shown = records.ok ? records.value : [];
  return (
    <div className="home-dashboard">
      <p className="home-welcome">
        Welcome back, {userName}. Here&apos;s what&apos;s happening across your organisation.
      </p>

      <section className="home-section" aria-labelledby="home-attention-heading">
        <div className="home-section-head">
          <h2 id="home-attention-heading">What needs your attention</h2>
        </div>
        <div className="home-kpi-grid">
          {metrics.map((metric) => {
            const Icon = metricIcon(metric.icon);
            return (
              <Link key={metric.label} href={metric.href} className="home-kpi">
                <span className="home-kpi-icon" aria-hidden="true">
                  <Icon size={20} />
                </span>
                <span className="home-kpi-label">{metric.label}</span>
                <strong className={metric.tone === "error" ? "home-kpi-value is-error" : "home-kpi-value"}>{metric.value}</strong>
                <span className="home-kpi-detail">{metric.detail}</span>
                <span className="home-kpi-action">{metric.action}</span>
              </Link>
            );
          })}
        </div>
      </section>

      <div className="home-split">
        <section className="home-panel" aria-labelledby="home-activity-heading">
          <div className="home-section-head">
            <h2 id="home-activity-heading">Recent activity</h2>
            {records.ok && totalRecords > 5 ? (
              showAllRecords ? (
                <Link href="/admin">Show recent</Link>
              ) : (
                <Link href="/admin?record=all">View all</Link>
              )
            ) : null}
          </div>
          {!records.ok ? <StatusMessage tone="error">{records.error}</StatusMessage> : null}
          {records.ok && shown.length === 0 ? <EmptyState>No recent activity is recorded yet.</EmptyState> : null}
          {records.ok && shown.length > 0 ? (
            <ul className="home-activity">
              {shown.map((record) => {
                const actor = actorNames.get(record.actorPersonId) ?? record.actorPersonId;
                const Icon = activityIcon(record.action);
                return (
                  <li key={`${record.action}-${record.entityId}-${record.occurredAt}-${record.actorPersonId}`}>
                    <span className="home-activity-icon" aria-hidden="true">
                      <Icon size={18} />
                    </span>
                    <div className="home-activity-body">
                      <p className="home-activity-copy">{describeActivity(record, actor)}</p>
                      <p className="home-activity-meta">
                        <span>{actor}</span>
                        <span className="home-activity-status">{activityStatus(record)}</span>
                      </p>
                    </div>
                    <time dateTime={record.occurredAt}>{formatRecordTime(record.occurredAt)}</time>
                  </li>
                );
              })}
            </ul>
          ) : null}
        </section>

        <div className="home-side">
          <section className="home-panel" aria-labelledby="home-actions-heading">
            <div className="home-section-head">
              <h2 id="home-actions-heading">Quick actions</h2>
            </div>
            <ul className="home-actions">
              <li>
                <Link href="/admin/role-profiles/new">
                  <span className="home-action-icon" aria-hidden="true">
                    <FilePlus2 size={18} />
                  </span>
                  <span>
                    <strong>Create a role profile</strong>
                    <span>Start a draft for a role in your organisation.</span>
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/role-profiles?entry=upload">
                  <span className="home-action-icon" aria-hidden="true">
                    <Upload size={18} />
                  </span>
                  <span>
                    <strong>Upload organogram</strong>
                    <span>Add a chart file or build positions online.</span>
                  </span>
                </Link>
              </li>
              <li>
                <Link href="/admin/role-profiles/review">
                  <span className="home-action-icon" aria-hidden="true">
                    <CheckCheck size={18} />
                  </span>
                  <span>
                    <strong>View approvals</strong>
                    <span>Open the review inbox and approval path.</span>
                  </span>
                </Link>
              </li>
            </ul>
          </section>

          <section className="home-panel" aria-labelledby="home-approvals-heading">
            <div className="home-section-head">
              <h2 id="home-approvals-heading">My approvals</h2>
              <Link href="/admin/role-profiles/review">Open inbox</Link>
            </div>
            <ul className="home-approvals">
              <ApprovalRow
                href="/admin/role-profiles/review"
                label="Awaiting my review"
                count={approvals.awaiting}
                empty="No profiles are waiting for you."
              />
              <ApprovalRow
                href="/admin/role-profiles/review?view=submitted"
                label="Submitted by me"
                count={approvals.submitted}
                empty="You have not submitted a profile for review."
              />
              <ApprovalRow
                href="/admin/role-profiles/review?view=reviewed"
                label="Previously reviewed"
                count={approvals.reviewed}
                empty="No reviewed profiles yet."
              />
            </ul>
          </section>
        </div>
      </div>
    </div>
  );
}

function ApprovalRow({
  href,
  label,
  count,
  empty,
}: {
  href: string;
  label: string;
  count: Settled<number>;
  empty: string;
}) {
  const value = count.ok ? String(count.value) : "—";
  const detail = count.ok ? (count.value === 0 ? empty : `${count.value === 1 ? "1 profile" : `${count.value} profiles`}`) : count.error;
  return (
    <li>
      <Link href={href}>
        <span>
          <strong>{label}</strong>
          <span className={count.ok ? undefined : "is-error"}>{detail}</span>
        </span>
        <em>{value}</em>
      </Link>
    </li>
  );
}

function metricIcon(kind: HomeMetric["icon"]): LucideIcon {
  switch (kind) {
    case "frameworks":
      return Layers3;
    case "assignments":
      return UserCheck;
    case "plans":
      return ClipboardList;
    case "organisation":
      return Building2;
  }
}

function activityIcon(action: string): LucideIcon {
  switch (action) {
    case "person.signed_in":
      return LogIn;
    default:
      return Network;
  }
}

function formatRecordTime(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: "UTC",
  }).format(new Date(parsed));
}
