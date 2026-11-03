import { cookies } from "next/headers";
import { redirect } from "next/navigation";

import { WorkflowShell } from "@/features/role-profiles/frame";
import { ReviewInbox } from "@/features/role-profiles/review-inbox";
import { listPeople } from "@/lib/org-directory";
import { listApprovalQueue, readApprovalPath, type ApprovalQueueItem } from "@/lib/role-profile";
import { settle, type Settled } from "@/lib/safe";
import { SESSION_COOKIE } from "@/lib/session";

export const dynamic = "force-dynamic";

export default async function ReviewInboxPage({
  searchParams,
}: {
  searchParams: Promise<{ profile?: string; view?: string; error?: string; notice?: string }>;
}) {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) {
    redirect("/");
  }
  const query = await searchParams;
  if (query.profile) {
    redirect(`/admin/role-profiles/${query.profile}/review`);
  }
  const view = query.view === "submitted" || query.view === "reviewed" ? query.view : "awaiting";
  const [awaitingItems, submittedItems, reviewedItems, path, people] = await Promise.all([
    settle(listApprovalQueue(token, "awaiting"), "Approvals awaiting review"),
    settle(listApprovalQueue(token, "submitted"), "Submitted approvals"),
    settle(listApprovalQueue(token, "reviewed"), "Reviewed approvals"),
    settle(readApprovalPath(token), "Approval path"),
    settle(listPeople(token), "People"),
  ]);
  const queue =
    view === "submitted" ? submittedItems : view === "reviewed" ? reviewedItems : awaitingItems;
  const personNames = new Map((people.ok ? people.value : []).map((person) => [person.id, person.name]));

  return (
    <WorkflowShell
      token={token}
      current="approve"
      error={query.error ?? ""}
      notice={query.notice ?? ""}
      evidence={{ organogramCount: null, clarificationComplete: null }}
    >
      <ReviewInbox
        view={view}
        awaiting={countSettled(awaitingItems)}
        submitted={countSettled(submittedItems)}
        reviewed={countSettled(reviewedItems)}
        queue={queue}
        path={path}
        personNames={personNames}
      />
    </WorkflowShell>
  );
}

function countSettled(value: Settled<ApprovalQueueItem[]>): Settled<number> {
  if (!value.ok) {
    return value;
  }
  return { ok: true, value: value.value.length };
}
