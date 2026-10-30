import Link from "next/link";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { WorkflowShell } from "@/features/role-profiles/frame";
import { saveApprovalPathForm } from "@/features/role-profiles/actions";
import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import { listApprovalQueue } from "@/lib/role-profile";
import { settle } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

const views = [
  { id: "awaiting", label: "Awaiting my review" },
  { id: "submitted", label: "Submitted by me" },
  { id: "reviewed", label: "Previously reviewed" },
] as const;

export default async function ReviewInboxPage({ searchParams }: { searchParams: Promise<{ profile?: string; view?: string; error?: string; notice?: string }> }) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const query = await searchParams;
  if (query.profile) {
    redirect(`/admin/role-profiles/${query.profile}/review`);
  }
  const view = query.view === "submitted" || query.view === "reviewed" ? query.view : "awaiting";
  const queue = await settle(listApprovalQueue(token, view), "Approval queue");
  return (
    <WorkflowShell token={token} current="approve" error={query.error ?? ""} notice={query.notice ?? ""}>
      <h2>Approvals</h2>
      <nav className="skill-filters ds-tabs" aria-label="Approval views">
        {views.map((item) => (
          <Link
            key={item.id}
            href={item.id === "awaiting" ? "/admin/role-profiles/review" : `/admin/role-profiles/review?view=${item.id}`}
            aria-current={view === item.id ? "page" : undefined}
          >
            {item.label}
          </Link>
        ))}
      </nav>
      {!queue.ok ? <StatusMessage tone="error">{queue.error}</StatusMessage> : null}
      {queue.ok && queue.value.length === 0 ? <EmptyState>No profiles are in this view.</EmptyState> : null}
      {queue.ok ? (
        <ul className="approval-queue">
          {queue.value.map((item) => (
            <li key={item.versionId}>
              <Link href={`/admin/role-profiles/${item.profileId}/review`}>{item.roleName}</Link>
              <p>
                {item.department || "Department not recorded"} · version {item.versionNumber} · {item.status}
                {item.stageLabel ? ` · ${item.stageLabel}` : ""} · submitted {item.submittedAt || "not recorded"} · reviewer {item.assigneePersonId || "not assigned"}
              </p>
              <p>Outstanding: {item.outstanding || "None"}</p>
            </li>
          ))}
        </ul>
      ) : null}
      <h3>Approval path</h3>
      <p>Later submissions use the path saved here. A review that has already started keeps its own stages.</p>
      <form action={saveApprovalPathForm} className="framework-form">
        <input type="hidden" name="preset" value="standard" />
        <PendingButton>Use HR, line manager, and business leader</PendingButton>
      </form>
      <form action={saveApprovalPathForm} className="framework-form">
        <input type="hidden" name="preset" value="leader" />
        <PendingButton>Use business leader approval only</PendingButton>
      </form>
    </WorkflowShell>
  );
}
