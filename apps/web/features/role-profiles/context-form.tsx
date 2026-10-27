import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile } from "@/lib/role-profile";

import { saveRoleContext } from "./actions";

const levels = [
  ["", "Not set"],
  ["individual", "Individual"],
  ["team_lead", "Team lead"],
  ["manager", "Manager"],
  ["director", "Director"],
  ["executive", "Executive"],
] as const;

export function ContextForm({ profile }: { profile: RoleProfile }) {
  const version = profile.version;
  const editable = version.status === "profile_draft";
  return (
    <form action={saveRoleContext} className="framework-form">
      <input type="hidden" name="profileId" value={profile.id} />
      <input type="hidden" name="versionId" value={version.id} />
      <input type="hidden" name="expectedRevision" value={version.revision} />
      {version.sources.map((source) => (
        <span key={source.id}>
          <input type="hidden" name="sourceTitle" value={source.title} />
          <input type="hidden" name="sourceUri" value={source.referenceUri} />
        </span>
      ))}
      <p>
        Status <strong>{version.status}</strong>. Revision {version.revision}.
      </p>
      <label>
        Purpose
        <textarea name="purpose" defaultValue={version.purpose} disabled={!editable} rows={3} />
      </label>
      <label>
        Responsibilities
        <textarea name="responsibilities" defaultValue={version.responsibilities} disabled={!editable} rows={3} />
      </label>
      <label>
        Business outcomes
        <textarea name="businessOutcomes" defaultValue={version.businessOutcomes} disabled={!editable} rows={3} />
      </label>
      <label>
        Decision scope
        <textarea name="decisionScope" defaultValue={version.decisionScope} disabled={!editable} rows={3} />
      </label>
      <label>
        Reporting level
        <select name="reportingLevel" defaultValue={version.reportingLevel} disabled={!editable}>
          {levels.map(([value, label]) => (
            <option key={value || "empty"} value={value}>
              {label}
            </option>
          ))}
        </select>
      </label>
      <label>
        Industry
        <input name="industry" defaultValue={version.industry} disabled={!editable} />
      </label>
      <label>
        Region
        <input name="region" defaultValue={version.region} disabled={!editable} />
      </label>
      <label>
        Team size
        <input name="teamSize" inputMode="numeric" defaultValue={version.teamSize ?? ""} disabled={!editable} />
      </label>
      {editable ? <PendingButton className="primary">Save context</PendingButton> : <p>Published and in-review versions are read here. Start the next draft from history to edit.</p>}
    </form>
  );
}
