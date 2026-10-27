import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile, RoleProfileHistory, RoleProfileVersion } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { commentOnRoleProfileForm, publishRoleProfileForm, submitRoleProfileForm } from "./actions";

export function ReviewPanel({
  profile,
  history,
  versions,
}: {
  profile: RoleProfile;
  history: Settled<RoleProfileHistory>;
  versions: Settled<RoleProfileVersion[]>;
}) {
  const version = profile.version;
  const previous = versions.ok ? versions.value.filter((item) => item.id !== version.id).at(-1) : undefined;
  return (
    <section className="directory">
      <h2>Review</h2>
      <p>
        Version {version.versionNumber} is <strong>{version.status}</strong>. Submit and publish are the transitions this service accepts. It does not accept reject or request-changes.
      </p>
      <h3>Context used for this version</h3>
      <dl className="generation-meta">
        <div>
          <dt>Purpose</dt>
          <dd>{version.purpose || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Responsibilities</dt>
          <dd>{version.responsibilities || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Business outcomes</dt>
          <dd>{version.businessOutcomes || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Decision scope</dt>
          <dd>{version.decisionScope || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Reporting level</dt>
          <dd>{version.reportingLevel || "Not recorded"}</dd>
        </div>
      </dl>
      <h3>Skills</h3>
      {version.skills.length === 0 ? <EmptyState>No skills are on this version.</EmptyState> : (
        <ul>
          {version.skills.map((skill) => (
            <li key={skill.id}>
              {skill.skillName} · {skill.category} · proficiency {skill.requiredProficiency} · {skill.priority} · {skill.provenance}
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
      <h3>Earlier comments</h3>
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
      {version.status === "profile_draft" ? (
        <form action={submitRoleProfileForm}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <PendingButton className="primary">Submit for review</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_in_review" ? (
        <>
          <form action={publishRoleProfileForm}>
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <PendingButton className="primary">Publish</PendingButton>
          </form>
          <form action={commentOnRoleProfileForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <label>
              Comment
              <textarea name="body" required rows={3} />
            </label>
            <PendingButton>Save comment</PendingButton>
          </form>
        </>
      ) : null}
      {version.status === "profile_published" ? <p>This version is published.</p> : null}
    </section>
  );
}
