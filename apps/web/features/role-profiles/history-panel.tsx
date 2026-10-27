import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile, RoleProfileComparison, RoleProfileHistory, RoleProfileVersion } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { nextDraftForm } from "./actions";

export function HistoryPanel({
  profile,
  versions,
  history,
  comparison,
}: {
  profile: RoleProfile;
  versions: Settled<RoleProfileVersion[]>;
  history: Settled<RoleProfileHistory>;
  comparison: Settled<RoleProfileComparison> | null;
}) {
  const version = profile.version;
  return (
    <section className="directory">
      <h2>History</h2>
      <p>
        Current version {version.versionNumber} is <strong>{version.status}</strong>.
      </p>
      {!versions.ok ? <StatusMessage tone="error">{versions.error}</StatusMessage> : null}
      {versions.ok && versions.value.length === 0 ? <EmptyState>No versions are stored.</EmptyState> : null}
      {versions.ok ? (
        <ul>
          {versions.value.map((item) => (
            <li key={item.id}>
              Version {item.versionNumber} · {item.status} · {item.skills.length} skills · {item.id}
            </li>
          ))}
        </ul>
      ) : null}
      {versions.ok && versions.value.length > 1 ? (
        <form className="framework-form" method="get">
          <label>
            From
            <select name="from" defaultValue={versions.value[0]?.id}>
              {versions.value.map((item) => (
                <option key={item.id} value={item.id}>
                  Version {item.versionNumber}
                </option>
              ))}
            </select>
          </label>
          <label>
            To
            <select name="to" defaultValue={versions.value.at(-1)?.id}>
              {versions.value.map((item) => (
                <option key={item.id} value={item.id}>
                  Version {item.versionNumber}
                </option>
              ))}
            </select>
          </label>
          <button type="submit">Compare</button>
        </form>
      ) : null}
      {comparison && !comparison.ok ? <StatusMessage tone="error">{comparison.error}</StatusMessage> : null}
      {comparison?.ok ? (
        <article>
          <h3>
            Version {comparison.value.from.versionNumber} to version {comparison.value.to.versionNumber}
          </h3>
          <p>Added: {comparison.value.added.join(", ") || "none"}</p>
          <p>Removed: {comparison.value.removed.join(", ") || "none"}</p>
          {comparison.value.changed.length === 0 ? <p>No proficiency or priority changes.</p> : (
            <ul>
              {comparison.value.changed.map((change) => (
                <li key={change.skillId}>
                  {change.skillId}: proficiency {change.fromProficiency} to {change.toProficiency}, priority {change.fromPriority} to {change.toPriority}
                </li>
              ))}
            </ul>
          )}
        </article>
      ) : null}
      <h3>Audit</h3>
      {!history.ok ? <StatusMessage tone="error">{history.error}</StatusMessage> : null}
      {history.ok && history.value.events.length === 0 ? <EmptyState>No audit events are stored.</EmptyState> : null}
      {history.ok ? (
        <ul>
          {history.value.events.map((event) => (
            <li key={event.id}>
              {event.action} · {event.actorPersonId} · {event.occurredAt}
            </li>
          ))}
        </ul>
      ) : null}
      {version.status === "profile_published" ? (
        <form action={nextDraftForm}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <PendingButton className="primary">Create next draft</PendingButton>
        </form>
      ) : null}
    </section>
  );
}
