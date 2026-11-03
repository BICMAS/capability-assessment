"use client";

import { useState } from "react";

import { PendingButton } from "@/components/ui/pending-button";
import type { ApprovalPathStage } from "@/lib/role-profile";

import { saveApprovalPathForm } from "./actions";

export type ApprovalPreset = "standard" | "leader";

export function ApprovalPathConfig({
  path,
  pathError = "",
}: {
  path: ApprovalPathStage[] | null;
  pathError?: string;
}) {
  const current = matchPreset(path);
  const [pending, setPending] = useState<ApprovalPreset | null>(null);

  return (
    <section className="role-card approval-path-panel" aria-labelledby="approval-path-heading">
      <header className="role-card-head">
        <h2 id="approval-path-heading">Approval path</h2>
        <p>
          Changing the default path applies only to future submissions. Reviews already in progress keep their original
          stages. Approved profile versions stay immutable.
        </p>
      </header>

      {pathError ? <p className="status-message is-error">{pathError}</p> : null}

      <div className="approval-path-current" aria-live="polite">
        <h3>Current configuration</h3>
        {path && path.length > 0 ? (
          <ol className="approval-path-stages">
            {path.map((stage) => (
              <li key={`${stage.position}-${stage.stageKey}`}>
                <strong>{stage.label}</strong>
                <span>{roleLabel(stage.requiredRole)}</span>
              </li>
            ))}
          </ol>
        ) : (
          <p className="portal-note">No organisation approval path is recorded yet. Choose a configuration below.</p>
        )}
        <p className="approval-path-match">
          {current === "standard"
            ? "Matched preset: Standard approval"
            : current === "leader"
              ? "Matched preset: Executive approval"
              : current === "custom"
                ? "Custom path (not a named preset)"
                : "No preset selected yet"}
        </p>
      </div>

      <div className="approval-path-choices" role="group" aria-label="Approval path presets">
        <button
          type="button"
          className={current === "standard" ? "approval-path-card is-current" : "approval-path-card"}
          aria-pressed={current === "standard"}
          onClick={() => setPending("standard")}
        >
          <span className="ds-status">Recommended</span>
          <strong>Standard approval</strong>
          <span>HR / Learning &amp; Development → Line Manager → Business Leader</span>
        </button>
        <button
          type="button"
          className={current === "leader" ? "approval-path-card is-current" : "approval-path-card"}
          aria-pressed={current === "leader"}
          onClick={() => setPending("leader")}
        >
          <strong>Executive approval</strong>
          <span>Business Leader only</span>
        </button>
      </div>

      {pending ? (
        <div className="chart-confirm approval-path-confirm" role="dialog" aria-modal="true" aria-labelledby="approval-path-confirm-title">
          <h3 id="approval-path-confirm-title">Save {pending === "standard" ? "standard" : "executive"} approval path?</h3>
          <p>
            This updates the organisation default for future submissions only. Open reviews keep their current stages.
            Approved versions remain unchanged.
          </p>
          <div className="approval-path-confirm-actions">
            <button type="button" onClick={() => setPending(null)}>
              Cancel
            </button>
            <form action={saveApprovalPathForm}>
              <input type="hidden" name="preset" value={pending} />
              <PendingButton className="primary">Save configuration</PendingButton>
            </form>
          </div>
        </div>
      ) : null}
    </section>
  );
}

export function matchPreset(path: ApprovalPathStage[] | null): ApprovalPreset | "custom" | "none" {
  if (!path || path.length === 0) {
    return "none";
  }
  const roles = path
    .slice()
    .sort((left, right) => left.position - right.position)
    .map((stage) => stage.requiredRole);
  if (roles.length === 3 && roles[0] === "hr" && roles[1] === "manager" && roles[2] === "leader") {
    return "standard";
  }
  if (roles.length === 1 && roles[0] === "leader") {
    return "leader";
  }
  return "custom";
}

function roleLabel(role: string): string {
  if (role === "hr") {
    return "HR / L&D";
  }
  if (role === "manager") {
    return "Line manager";
  }
  if (role === "leader") {
    return "Business leader";
  }
  return role;
}
