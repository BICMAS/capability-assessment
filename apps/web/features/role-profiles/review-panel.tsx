import Link from "next/link";

import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { ApprovalRecord, RoleProfile, RoleProfileHistory, RoleProfileVersion } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { assignApprovalForm, commentOnRoleProfileForm, decideRoleProfileForm, resubmitRoleProfileForm, submitRoleProfileForm } from "./actions";

export function ReviewPanel({
  profile,
  history,
  versions,
  approvals,
}: {
  profile: RoleProfile;
  history: Settled<RoleProfileHistory>;
  versions: Settled<RoleProfileVersion[]>;
  approvals?: Settled<ApprovalRecord>;
}) {
  const version = profile.version;
  const approval = approvals?.ok ? approvals.value : undefined;
  const previous = versions.ok ? versions.value.filter((item) => item.id !== version.id).at(-1) : undefined;
  const steps = approval?.steps ?? [];
  const last = steps.at(-1);
  const finalStage = Boolean(last && approval?.currentStage === last.stageKey);
  const assignment = approval?.assignments.find((item) => item.stageKey === approval.currentStage);
  const changesRequested = approval?.decisions.some((item) => item.decision === "changes_requested") ?? false;
  return (
    <section className="directory">
      <h2>{profile.roleName}</h2>
      <dl className="generation-meta">
        <div>
          <dt>Department</dt>
          <dd>{version.businessUnit || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Version</dt>
          <dd>
            {version.versionNumber} · revision {version.revision}
          </dd>
        </div>
        <div>
          <dt>Status</dt>
          <dd>{version.status}</dd>
        </div>
        <div>
          <dt>Submitted</dt>
          <dd>{version.submittedAt || "Not submitted"}</dd>
        </div>
        <div>
          <dt>Current stage</dt>
          <dd>{approval?.stageLabel || version.workflowStage || "Not in review"}</dd>
        </div>
        <div>
          <dt>Assigned reviewer</dt>
          <dd>{approval?.assigneePersonId || "Not assigned"}</dd>
        </div>
      </dl>
      <p>Outstanding: {approval?.outstanding || "No approval action is waiting."}</p>
      {approvals && !approvals.ok ? <StatusMessage tone="error">{approvals.error}</StatusMessage> : null}
      <h3>Approval path</h3>
      {steps.length === 0 ? <EmptyState>No approval stages are stored for this version.</EmptyState> : (
        <ol>
          {steps.map((step) => (
            <li key={step.id}>
              {step.position}. {step.label}
              {step.stageKey === approval?.currentStage ? " · current" : ""}
            </li>
          ))}
        </ol>
      )}
      <h3>Skills</h3>
      {version.skills.length === 0 ? <EmptyState>No skills are on this version.</EmptyState> : (
        <ul>
          {version.skills.map((skill) => (
            <li key={skill.id}>
              {skill.displayName || skill.skillName} · {skill.category} · proficiency {skill.requiredProficiency} · {skill.priority} · {skill.provenance}
            </li>
          ))}
        </ul>
      )}
      <h3>Previous version</h3>
      {!versions.ok ? <StatusMessage tone="error">{versions.error}</StatusMessage> : null}
      {versions.ok && !previous ? <EmptyState>No earlier version is stored.</EmptyState> : null}
      {previous ? (
        <p>
          Version {previous.versionNumber} is {previous.status} with {previous.skills.length} skills.
        </p>
      ) : null}
      <h3>Approval history</h3>
      {!approval || approval.decisions.length === 0 ? <EmptyState>No approval decisions are stored.</EmptyState> : (
        <ul>
          {approval.decisions.map((decision) => (
            <li key={decision.id}>
              {decision.decision} · {decision.actorPersonId} · {decision.createdAt}
              {decision.comment ? ` · ${decision.comment}` : ""}
            </li>
          ))}
        </ul>
      )}
      <h3>Reviewer comments</h3>
      {!history.ok ? <StatusMessage tone="error">{history.error}</StatusMessage> : null}
      {history.ok && history.value.comments.length === 0 ? <EmptyState>No comments are stored.</EmptyState> : null}
      {history.ok ? (
        <ul>
          {history.value.comments.map((comment) => (
            <li key={comment.id}>
              {comment.authorPersonId}: {comment.body}
            </li>
          ))}
        </ul>
      ) : null}
      <p>
        <Link href={`/admin/role-profiles/${profile.id}/history`}>View approval history</Link>
      </p>
      {version.status === "profile_draft" ? (
        <form action={submitRoleProfileForm}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <PendingButton className="primary">{changesRequested ? "Resubmit" : "Submit for review"}</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_in_review" && approval?.canDecide ? (
        <>
          <form action={decideRoleProfileForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="assignmentId" value={assignment?.id ?? ""} />
            <input type="hidden" name="decision" value="approved" />
            <input type="hidden" name="comment" value="Approved" />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <PendingButton className="primary">{finalStage ? "Approve and save" : "Approve"}</PendingButton>
          </form>
          <form action={decideRoleProfileForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="assignmentId" value={assignment?.id ?? ""} />
            <input type="hidden" name="decision" value="changes_requested" />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <label>
              Reason for changes
              <textarea name="comment" required rows={3} />
            </label>
            <PendingButton>Request changes</PendingButton>
          </form>
          <form action={decideRoleProfileForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="assignmentId" value={assignment?.id ?? ""} />
            <input type="hidden" name="decision" value="rejected" />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <label>
              Reason for rejection
              <textarea name="comment" required rows={3} />
            </label>
            <PendingButton>Reject</PendingButton>
          </form>
        </>
      ) : null}
      {version.status === "profile_in_review" && approval?.canResubmit ? (
        <form action={resubmitRoleProfileForm}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <PendingButton className="primary">Resubmit</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_in_review" ? (
        <form action={assignApprovalForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <label>
            Assign reviewer
            <input name="assigneePersonId" required />
          </label>
          <PendingButton>Assign reviewer</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_draft" || version.status === "profile_in_review" ? (
        <form action={commentOnRoleProfileForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <label>
            Comment
            <textarea name="body" required rows={3} />
          </label>
          <PendingButton>Save comment</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_published" ? (
        <p>
          Approved {version.approvedAt || "on record"} by {version.approvedByPersonId || "the recorded approver"}. This version stays immutable.
          {" "}
          <Link href={`/admin/role-profiles/${profile.id}/insights`}>Open insights</Link>
        </p>
      ) : null}
    </section>
  );
}
