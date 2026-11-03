import Link from "next/link";

import { StatusMessage } from "@/components/ui/status-message";
import type { ApprovalPathStage, ApprovalQueueItem } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { ApprovalPathConfig } from "./approval-path-config";
import { ApprovalQueuePanel } from "./approval-queue-panel";

export function ReviewInbox({
  view,
  awaiting,
  submitted,
  reviewed,
  queue,
  path,
  personNames,
}: {
  view: "awaiting" | "submitted" | "reviewed";
  awaiting: Settled<number>;
  submitted: Settled<number>;
  reviewed: Settled<number>;
  queue: Settled<ApprovalQueueItem[]>;
  path: Settled<ApprovalPathStage[]>;
  personNames: Map<string, string>;
}) {
  return (
    <div className="approval-workspace">
      <section className="approval-overview" aria-labelledby="approval-overview-heading">
        <header className="role-card-head">
          <h2 id="approval-overview-heading">Approval overview</h2>
          <p>Live counts from your approval queues.</p>
        </header>
        <ul className="approval-overview-grid">
          <OverviewCard
            href="/admin/role-profiles/review"
            label="Awaiting my review"
            count={awaiting}
            current={view === "awaiting"}
            empty="Nothing waiting for you"
          />
          <OverviewCard
            href="/admin/role-profiles/review?view=submitted"
            label="Submitted by me"
            count={submitted}
            current={view === "submitted"}
            empty="No submissions yet"
          />
          <OverviewCard
            href="/admin/role-profiles/review?view=reviewed"
            label="Previously reviewed"
            count={reviewed}
            current={view === "reviewed"}
            empty="No reviews yet"
          />
        </ul>
      </section>

      {!queue.ok ? <StatusMessage tone="error">{queue.error}</StatusMessage> : null}
      {queue.ok ? <ApprovalQueuePanel view={view} items={queue.value} personNames={personNames} /> : null}

      <ApprovalPathConfig path={path.ok ? path.value : null} pathError={path.ok ? "" : path.error} />
    </div>
  );
}

function OverviewCard({
  href,
  label,
  count,
  current,
  empty,
}: {
  href: string;
  label: string;
  count: Settled<number>;
  current: boolean;
  empty: string;
}) {
  const value = count.ok ? String(count.value) : "—";
  const detail = count.ok
    ? count.value === 0
      ? empty
      : count.value === 1
        ? "1 profile"
        : `${count.value} profiles`
    : count.error;
  return (
    <li>
      <Link className={current ? "approval-overview-card is-current" : "approval-overview-card"} href={href} aria-current={current ? "page" : undefined}>
        <span>{label}</span>
        <strong>{value}</strong>
        <span>{detail}</span>
      </Link>
    </li>
  );
}
