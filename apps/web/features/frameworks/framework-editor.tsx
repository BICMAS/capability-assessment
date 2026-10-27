import { PendingButton } from "@/components/ui/pending-button";
import { StatusMessage } from "@/components/ui/status-message";
import type { FrameworkVersion } from "@/lib/capability-framework";

import {
  acceptProposal,
  addFrameworkCapability,
  addFrameworkRole,
  createFrameworkDraft,
  editPublishedFramework,
  approveFramework,
  importFrameworkCsv,
  publishFramework,
  rejectProposal,
  saveFrameworkIndicator,
  saveNegotiation,
  submitFrameworkForReview,
} from "./actions";

export function FrameworkEditor({
  versions,
  reviewBlocked,
  blockedAction,
  blockedVersionId,
  importError,
  saveError,
  frozenFramework,
  frozenVersion,
  frozenVersionId,
}: {
  versions: FrameworkVersion[];
  reviewBlocked: boolean;
  blockedAction: string;
  blockedVersionId: string;
  importError: string;
  saveError: string;
  frozenFramework: string;
  frozenVersion: string;
  frozenVersionId: string;
}) {
  const ordered = [...versions].sort(
    (left, right) =>
      left.frameworkName.localeCompare(right.frameworkName) || left.versionNumber - right.versionNumber,
  );
  const published = ordered.filter((version) => version.status === "published").at(-1);
  const hasDraft = ordered.some((version) => version.status === "draft");

  return (
    <>
      <div className="studio-tool">
        <form action={importFrameworkCsv} className="framework-form">
          <label>
            CSV
            <input name="file" type="file" accept=".csv,text/csv" required />
          </label>
          <StatusMessage tone="error">{importError}</StatusMessage>
          <PendingButton>Import CSV</PendingButton>
        </form>
        {ordered.length === 0 ? (
          <form action={createFrameworkDraft} className="framework-form">
            <label>
              Framework
              <input name="frameworkName" defaultValue="Sales capability framework" required />
            </label>
            <PendingButton className="primary">Create draft</PendingButton>
          </form>
        ) : null}
      </div>
      <div className="studio-versions">
        {ordered.map((version) => (
          <article key={version.id} className={version.status === "published" ? "studio-version studio-document" : "studio-version"}>
            <div className="studio-identity">
              <VersionSummary version={version} />
              {version.rowErrors.length > 0 ? (
                <>
                  <h4>Row errors</h4>
                  <ul>
                    {version.rowErrors.map((rowError) => (
                      <li key={`${rowError.row}-${rowError.message}`}>
                        Row {rowError.row}: {rowError.message}
                      </li>
                    ))}
                  </ul>
                </>
              ) : null}
            </div>
            <div className="studio-actions">
              {version.proposals.length > 0 ? (
                <div>
                  <h4>Proposals</h4>
                  <ul>
                    {version.proposals.map((proposal) => (
                      <li key={proposal.id}>
                        <p>
                          {proposal.message} · {proposal.status}
                        </p>
                        {proposal.status === "pending" ? (
                          <div className="proposal-actions">
                            <form action={rejectProposal}>
                              <input type="hidden" name="versionId" value={version.id} />
                              <input type="hidden" name="proposalId" value={proposal.id} />
                              <PendingButton>Reject {proposal.capabilityCode}</PendingButton>
                            </form>
                            <form action={acceptProposal}>
                              <input type="hidden" name="versionId" value={version.id} />
                              <input type="hidden" name="proposalId" value={proposal.id} />
                              <PendingButton className="primary">Accept {proposal.capabilityCode}</PendingButton>
                            </form>
                          </div>
                        ) : null}
                      </li>
                    ))}
                  </ul>
                </div>
              ) : null}
              <NegotiationEditor
                version={version}
                saveError={frozenMessage(version, saveError, frozenFramework, frozenVersion, frozenVersionId)}
              />
              {version.status === "draft" ? (
                <form action={submitFrameworkForReview}>
                  <input type="hidden" name="versionId" value={version.id} />
                  {blockedNote(reviewBlocked, blockedAction, "submit", version.id, blockedVersionId)}
                  <PendingButton className="primary">Submit {version.frameworkName} for review</PendingButton>
                </form>
              ) : null}
              {version.status === "draft" || version.status === "in_review" ? (
                <form action={approveFramework}>
                  <input type="hidden" name="versionId" value={version.id} />
                  {blockedNote(reviewBlocked, blockedAction, "approve", version.id, blockedVersionId)}
                  <PendingButton className="primary">Approve {version.frameworkName}</PendingButton>
                </form>
              ) : null}
              {version.status === "approved" ? (
                <form action={publishFramework}>
                  <input type="hidden" name="versionId" value={version.id} />
                  {blockedNote(reviewBlocked, blockedAction, "publish", version.id, blockedVersionId)}
                  <PendingButton className="primary">Publish {version.frameworkName}</PendingButton>
                </form>
              ) : null}
              <DraftSteps version={version} />
            </div>
          </article>
        ))}
      </div>
      {!hasDraft && published ? (
        <form action={editPublishedFramework} className="studio-follow">
          <input type="hidden" name="versionId" value={published.id} />
          <PendingButton>Edit as new draft</PendingButton>
        </form>
      ) : null}
    </>
  );
}

function frozenMessage(
  version: FrameworkVersion,
  saveError: string,
  framework: string,
  versionNumber: string,
  versionId: string,
): string {
  if (!saveError || version.frameworkName !== framework || String(version.versionNumber) !== versionNumber) {
    return "";
  }
  if (version.status !== "published" && version.status !== "retired") {
    return "";
  }
  if (versionId !== "" && version.id !== versionId) {
    return "";
  }
  return saveError;
}

function blockedNote(
  reviewBlocked: boolean,
  blockedAction: string,
  action: string,
  versionId: string,
  blockedVersionId: string,
) {
  if (!reviewBlocked) {
    return null;
  }
  if (blockedAction !== "" && blockedAction !== action) {
    return null;
  }
  if (blockedVersionId !== "" && blockedVersionId !== versionId) {
    return null;
  }
  return <StatusMessage tone="error">A pending proposal is still open.</StatusMessage>;
}

function NegotiationEditor({ version, saveError }: { version: FrameworkVersion; saveError: string }) {
  const negotiation = version.capabilities.find((capability) => capability.name === "Negotiation");
  const indicator = negotiation?.levels.find((level) => level.level === 4)?.indicators[0];
  if (!negotiation || !indicator) {
    return null;
  }
  return (
    <form action={saveNegotiation} className="framework-form">
      <input type="hidden" name="versionId" value={version.id} />
      <input type="hidden" name="indicatorId" value={indicator.id} />
      <input type="hidden" name="frameworkName" value={version.frameworkName} />
      <input type="hidden" name="versionNumber" value={version.versionNumber} />
      <label>
        Negotiation level 4
        <textarea name="statement" required defaultValue={indicator.statement} />
      </label>
      <StatusMessage tone="error">{saveError}</StatusMessage>
      <PendingButton className="primary">Save Negotiation</PendingButton>
    </form>
  );
}

function DraftSteps({ version }: { version: FrameworkVersion }) {
  if (version.status !== "draft") {
    return null;
  }
  if (version.roles.length === 0) {
    return (
      <form action={addFrameworkRole} className="framework-form">
        <input type="hidden" name="versionId" value={version.id} />
        <label>
          Role
          <input name="roleName" defaultValue="Area Sales Manager" required />
        </label>
        <PendingButton className="primary">Add role</PendingButton>
      </form>
    );
  }
  if (version.requirements.length === 0) {
    return (
      <form action={addFrameworkCapability} className="framework-form">
        <input type="hidden" name="versionId" value={version.id} />
        <input type="hidden" name="roleId" value={version.roles[0]?.id ?? ""} />
        <label>
          Capability
          <input name="capabilityName" defaultValue="Negotiation" required />
        </label>
        <label>
          Required level
          <input name="requiredLevel" type="number" min={0} max={4} defaultValue={4} required />
        </label>
        <PendingButton className="primary">Add capability</PendingButton>
      </form>
    );
  }
  if (indicatorCount(version) === 0) {
    return (
      <form action={saveFrameworkIndicator} className="framework-form">
        <input type="hidden" name="versionId" value={version.id} />
        <input type="hidden" name="capabilityId" value={version.requirements[0]?.capabilityId ?? ""} />
        <input type="hidden" name="level" value={version.requirements[0]?.requiredLevel ?? 4} />
        <label>
          Behavioural indicator
          <textarea name="indicator" required defaultValue="Reaches a commercial agreement with the buyer." />
        </label>
        <PendingButton className="primary">Save draft</PendingButton>
      </form>
    );
  }
  return null;
}

function VersionSummary({ version }: { version: FrameworkVersion }) {
  const capabilityNames = new Map(version.capabilities.map((capability) => [capability.id, capability.name]));
  const roleNames = new Map(version.roles.map((role) => [role.id, role.name]));
  const indicators = version.capabilities.flatMap((capability) =>
    capability.levels.flatMap((level) =>
      level.indicators.map((indicator) => ({ ...indicator, level: level.level, capability: capability.name })),
    ),
  );

  return (
    <div>
      <p className="studio-kicker">version {version.versionNumber}</p>
      <h3>{version.frameworkName}</h3>
      <StatusMark status={version.status} />
      {version.roles.length > 0 ? (
        <>
          <h4>Roles</h4>
          <ul>
            {version.roles.map((role) => (
              <li key={role.id}>{role.name}</li>
            ))}
          </ul>
        </>
      ) : null}
      {version.requirements.length > 0 ? (
        <>
          <h4>Requirements</h4>
          <ul>
            {version.requirements.map((requirement) => (
              <li key={requirement.id}>
                {roleNames.get(requirement.roleId)} requires {capabilityNames.get(requirement.capabilityId)} at level{" "}
                {requirement.requiredLevel}
              </li>
            ))}
          </ul>
        </>
      ) : null}
      {indicators.length > 0 ? (
        <>
          <h4>Behavioural indicators</h4>
          <ul>
            {indicators.map((indicator) => (
              <li key={indicator.id}>
                {indicator.capability} level {indicator.level}: {indicator.statement}
              </li>
            ))}
          </ul>
        </>
      ) : null}
    </div>
  );
}

function StatusMark({ status }: { status: string }) {
  return (
    <p className={status === "published" ? "studio-status studio-locked" : "studio-status"}>
      <span>{statusLabel(status)}</span>
      {status === "published" ? <span className="studio-lock">Locked</span> : null}
    </p>
  );
}

function statusLabel(status: string): string {
  switch (status) {
    case "draft":
      return "draft";
    case "in_review":
      return "in review";
    case "approved":
      return "approved";
    case "published":
      return "published";
    case "retired":
      return "retired";
    default:
      return status;
  }
}

function indicatorCount(version: FrameworkVersion): number {
  return version.capabilities.reduce(
    (count, capability) =>
      count + capability.levels.reduce((levelCount, level) => levelCount + level.indicators.length, 0),
    0,
  );
}
