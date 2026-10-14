import { ProposalConfirm } from "@/components/proposal-confirm";
import type { FrameworkVersion } from "@/lib/capability-framework";
import type { AssessmentSession, Assignment, Plan, PlanSkill } from "@/lib/generated/assessment-plan/plans";

import { approveAssessmentPlan, confirmProposal, generatePlan, requestPlanChange, savePlanSkill, submitPlanForReview } from "./assessment-actions";

export function PlanStep({
  assignment,
  plan,
  session,
  version,
  personName,
  roleName,
  approverName,
  error,
}: {
  assignment: Assignment;
  plan: Plan | null;
  session: AssessmentSession | null;
  version: FrameworkVersion | null;
  personName: string;
  roleName: string;
  approverName: string;
  error: string;
}) {
  const editable = plan?.status === "generated" || plan?.status === "changes_requested";
  const approved = plan?.status === "approved";

  return (
    <div className="studio-tool">
      <p className="studio-kicker">
        {personName} · {roleName}
      </p>
      {error ? <p className="login-error">{error}</p> : null}
      {plan ? (
        <>
          <p className="studio-status">
            <span className="studio-lock">{plan.status.replaceAll("_", " ")}</span>
            <span>Version {plan.versionNumber}</span>
            {approved ? <span>Approved by {approverName}{plan.approvedAt ? ` · ${formatWhen(plan.approvedAt)}` : ""}</span> : null}
          </p>
          {plan.changeReason ? <p>Changes requested: {plan.changeReason}</p> : null}
          <div className="plan-skills">
            {plan.skills.map((skill) => (
              <SkillCard
                key={skill.id}
                skill={skill}
                capabilityName={capabilityName(version, skill.capabilityId)}
                editable={editable}
                assignmentId={assignment.id}
                planId={plan.id}
              />
            ))}
          </div>
          {editable ? (
            <form action={submitPlanForReview}>
              <input type="hidden" name="assignmentId" value={assignment.id} />
              <input type="hidden" name="planId" value={plan.id} />
              <button type="submit">Submit for review</button>
            </form>
          ) : null}
          {plan.status === "under_review" ? (
            <form action={requestPlanChange} className="framework-form">
              <input type="hidden" name="assignmentId" value={assignment.id} />
              <input type="hidden" name="planId" value={plan.id} />
              <label>
                Reason for changes
                <textarea name="reason" required maxLength={280} />
              </label>
              <button type="submit">Request changes</button>
            </form>
          ) : null}
          {session ? (
            <ProposalConfirm
              personId={assignment.personId}
              sessions={[session]}
              action={confirmProposal}
              hidden={[
                { name: "assignmentId", value: assignment.id },
                { name: "planId", value: plan.id },
              ]}
            />
          ) : null}
          {plan.status === "generated" || plan.status === "under_review" || plan.status === "changes_requested" ? (
            <form action={approveAssessmentPlan}>
              <input type="hidden" name="assignmentId" value={assignment.id} />
              <input type="hidden" name="planId" value={plan.id} />
              <button type="submit">Approve plan</button>
            </form>
          ) : null}
        </>
      ) : (
        <form action={generatePlan}>
          <input type="hidden" name="assignmentId" value={assignment.id} />
          <button type="submit">Generate plan</button>
        </form>
      )}
    </div>
  );
}

function SkillCard({
  skill,
  capabilityName,
  editable,
  assignmentId,
  planId,
}: {
  skill: PlanSkill;
  capabilityName: string;
  editable: boolean;
  assignmentId: string;
  planId: string;
}) {
  return (
    <article className="plan-skill">
      <h3>{capabilityName}</h3>
      <p className="studio-status">
        {skill.requiresSmeReview ? <span className="studio-lock">Requires SME review</span> : null}
        {skill.rubricNeedsReview ? <span className="studio-lock">Rubric needs review</span> : null}
      </p>
      <p>{skill.rationale}</p>
      {editable ? (
        <form action={savePlanSkill} className="framework-form">
          <input type="hidden" name="assignmentId" value={assignmentId} />
          <input type="hidden" name="planId" value={planId} />
          <input type="hidden" name="skillId" value={skill.id} />
          <label>
            Method
            <input name="recommendedMethod" defaultValue={skill.recommendedMethod} required maxLength={80} />
          </label>
          <label>
            Instructions
            <textarea name="instructions" defaultValue={skill.instructions} required maxLength={8000} />
          </label>
          <h4>Items</h4>
          {skill.assessmentItems.map((item, index) => (
            <label key={item.id}>
              Item {index + 1}
              <textarea name={`prompt.${index}`} defaultValue={item.prompt} required maxLength={2000} />
            </label>
          ))}
          <h4>Rubric</h4>
          {skill.rubric.length === 0 ? <p>The rubric needs review.</p> : null}
          {(skill.rubric.length > 0 ? skill.rubric : [{ level: skill.requiredLevel, descriptor: "" }]).map((band, index) => (
            <div key={`${skill.id}-${band.level}-${index}`} className="rubric-edit">
              <label>
                Level
                <input name={`level.${index}`} defaultValue={band.level} inputMode="numeric" required />
              </label>
              <label>
                Anchor
                <textarea name={`descriptor.${index}`} defaultValue={band.descriptor} required maxLength={2000} />
              </label>
            </div>
          ))}
          <button type="submit">Save skill</button>
        </form>
      ) : (
        <>
          <p>Method · {skill.recommendedMethod}</p>
          <p>{skill.instructions}</p>
          <h4>Items</h4>
          <ul>
            {skill.assessmentItems.map((item) => (
              <li key={item.id}>{item.prompt}</li>
            ))}
          </ul>
          <h4>Rubric</h4>
          {skill.rubric.length === 0 ? <p>The rubric needs review.</p> : null}
          <ul>
            {skill.rubric.map((band) => (
              <li key={`${skill.id}-${band.level}`}>
                Level {band.level} · {band.descriptor}
              </li>
            ))}
          </ul>
        </>
      )}
    </article>
  );
}

function capabilityName(version: FrameworkVersion | null, capabilityId: string): string {
  return version?.capabilities.find((capability) => capability.id === capabilityId)?.name ?? "Capability";
}

function formatWhen(value: string): string {
  const parsed = Date.parse(value);
  if (Number.isNaN(parsed)) {
    return value;
  }
  return new Intl.DateTimeFormat("en-GB", {
    day: "numeric",
    month: "short",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
    timeZone: "UTC",
  }).format(new Date(parsed));
}
