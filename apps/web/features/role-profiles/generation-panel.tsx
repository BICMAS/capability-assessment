import Link from "next/link";

import { StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { GenerationJob } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { retryGenerationForm, startGenerationForm } from "./actions";
import { JobRefresh } from "./job-refresh";

export function GenerationPanel({
  profileId,
  versionId,
  contextVersion,
  idempotencyKey,
  editable,
  job,
}: {
  profileId: string;
  versionId: string;
  contextVersion: number;
  idempotencyKey: string;
  editable: boolean;
  job: Settled<GenerationJob> | null;
}) {
  const current = job?.ok ? job.value : null;
  const active = current?.status === "queued" || current?.status === "running";
  return (
    <section className="directory">
      <h2>Generation</h2>
      <JobRefresh active={active} />
      {editable && contextVersion > 0 ? (
        <form action={startGenerationForm} className="framework-form">
          <input type="hidden" name="profileId" value={profileId} />
          <input type="hidden" name="versionId" value={versionId} />
          <input type="hidden" name="contextVersion" value={contextVersion} />
          <input type="hidden" name="idempotencyKey" value={idempotencyKey} />
          <p>Context version {contextVersion}. The request returns when the job is stored.</p>
          <PendingButton className="primary">Queue generation</PendingButton>
        </form>
      ) : (
        <p>Queue generation from a draft that has a saved context version.</p>
      )}
      {job && !job.ok ? <StatusMessage tone="error">{job.error}</StatusMessage> : null}
      {current ? (
        <article className="studio-version">
          <h3>
            Job {current.id} · <span data-testid="generation-status">{current.status}</span>
          </h3>
          <dl className="generation-meta">
            <div>
              <dt>Model</dt>
              <dd>{current.modelId || "Not recorded"}</dd>
            </div>
            <div>
              <dt>Prompt</dt>
              <dd>{current.promptVersion || "Not recorded"}</dd>
            </div>
            <div>
              <dt>Input hash</dt>
              <dd>{current.inputHash || "Not recorded"}</dd>
            </div>
            <div>
              <dt>Output hash</dt>
              <dd>{current.outputHash || "Not recorded"}</dd>
            </div>
            <div>
              <dt>Latency</dt>
              <dd>{current.latencyMs} ms</dd>
            </div>
            <div>
              <dt>Tokens</dt>
              <dd>
                {current.promptTokens} prompt / {current.completionTokens} completion
              </dd>
            </div>
            <div>
              <dt>Attempts</dt>
              <dd>{current.attempts}</dd>
            </div>
          </dl>
          {current.lastError === "timeout" ? <StatusMessage>The provider timed out. Retry the same job.</StatusMessage> : null}
          {current.lastError && current.lastError !== "timeout" ? <p>Last error: {current.lastError}</p> : null}
          {current.uncertainty.length > 0 ? <p>Uncertainty: {current.uncertainty.join(", ")}. This is not proof the draft is correct.</p> : null}
          {current.unsupported.length > 0 ? (
            <ul>
              {current.unsupported.map((item, index) => (
                <li key={`${item.skillId}-${item.reason}-${index}`}>
                  {item.skillId || "Suggestion"} {item.kind} · {item.reason}
                </li>
              ))}
            </ul>
          ) : null}
          {current.status === "succeeded" ? (
            <Link className="learn-link" href={`/admin/role-profiles/${profileId}`}>
              Open the draft
            </Link>
          ) : null}
          {current.status === "failed" || current.status === "stale" || current.status === "cancelled" ? (
            <form action={retryGenerationForm}>
              <input type="hidden" name="profileId" value={profileId} />
              <input type="hidden" name="jobId" value={current.id} />
              <PendingButton className="primary">Retry job</PendingButton>
            </form>
          ) : null}
        </article>
      ) : null}
    </section>
  );
}
