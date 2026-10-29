import Link from "next/link";

import { buildWorkflow, stageStateLabel, type StageId, type WorkflowEvidence } from "./workflow";
import type { RoleProfile } from "@/lib/role-profile";

export type WorkflowStep = StageId;

export function WorkflowStepper({
  profile,
  current,
  evidence,
}: {
  profile?: RoleProfile | null;
  current: StageId;
  evidence?: WorkflowEvidence | null;
}) {
  const stages = buildWorkflow({ profile, current, evidence });
  return (
    <nav className="workflow-nav" aria-label="Role skills workflow">
      <ol>
        {stages.map((stage) => {
          const label = stageStateLabel(stage);
          const body = (
            <>
              <span className="workflow-index">{stage.number}</span>
              <span className="workflow-copy">
                <span className="workflow-title">{stage.title}</span>
                <span className="workflow-summary">{stage.description}</span>
                <span className="workflow-state">{label}</span>
              </span>
            </>
          );
          return (
            <li key={stage.id} data-state={stage.status} data-tone={stage.tone}>
              {stage.href ? (
                <Link href={stage.href} aria-current={stage.current ? "step" : undefined}>
                  {body}
                </Link>
              ) : (
                <span aria-disabled="true" aria-current={stage.current ? "step" : undefined}>
                  {body}
                </span>
              )}
            </li>
          );
        })}
      </ol>
    </nav>
  );
}
