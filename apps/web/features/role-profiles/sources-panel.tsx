import { EmptyState } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile } from "@/lib/role-profile";

import { saveRoleSource } from "./actions";

export function SourcesPanel({ profile }: { profile: RoleProfile }) {
  const version = profile.version;
  const editable = version.status === "profile_draft";
  return (
    <section className="directory">
      <h2>Source documents</h2>
      <p>The capability service stores a source title and reference. It does not extract text from an uploaded file.</p>
      <p>
        Extraction status: {version.sources.length === 0 ? "no source recorded" : `${version.sources.length} source reference${version.sources.length === 1 ? "" : "s"} stored`}.
      </p>
      {version.sources.length === 0 ? <EmptyState>No authorised source is attached.</EmptyState> : (
        <ul>
          {version.sources.map((source) => (
            <li key={source.id}>
              {source.title} — {source.referenceUri}
            </li>
          ))}
        </ul>
      )}
      {editable ? (
        <form action={saveRoleSource} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <input type="hidden" name="purpose" value={version.purpose} />
          <input type="hidden" name="responsibilities" value={version.responsibilities} />
          <input type="hidden" name="businessOutcomes" value={version.businessOutcomes} />
          <input type="hidden" name="decisionScope" value={version.decisionScope} />
          <input type="hidden" name="industry" value={version.industry} />
          <input type="hidden" name="region" value={version.region} />
          <input type="hidden" name="teamSize" value={version.teamSize ?? ""} />
          <input type="hidden" name="reportingLevel" value={version.reportingLevel} />
          {version.sources.map((source) => (
            <span key={source.id}>
              <input type="hidden" name="sourceTitle" value={source.title} />
              <input type="hidden" name="sourceUri" value={source.referenceUri} />
            </span>
          ))}
          <label>
            Title
            <input name="sourceTitle" required />
          </label>
          <label>
            Reference URI
            <input name="sourceUri" required />
          </label>
          <PendingButton className="primary">Save source reference</PendingButton>
        </form>
      ) : null}
    </section>
  );
}
