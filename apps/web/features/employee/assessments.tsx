import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { AssessmentSession } from "@/lib/generated/assessment-plan/plans";
import type { Settled } from "@/lib/safe";

import { submitSessionAnswers } from "@/app/(employee)/employee/session-actions";

export function EmployeeAssessments({
  sessions,
  sessionError = "",
}: {
  sessions: Settled<AssessmentSession[]>;
  sessionError?: string;
}) {
  const waiting = sessions.ok ? sessions.value.filter((session) => session.status !== "submitted") : [];
  const submitted = sessions.ok ? sessions.value.filter((session) => session.status === "submitted") : [];

  return (
    <section className="employee-section" id="assessments" aria-labelledby="employee-assessments-heading">
      <header className="employee-section-head">
        <h2 id="employee-assessments-heading">Assessments</h2>
        <p>Complete assigned sessions and review submitted answers.</p>
      </header>

      {sessionError ? <StatusMessage tone="error">{sessionError}</StatusMessage> : null}
      {!sessions.ok ? <StatusMessage tone="error">{sessions.error}</StatusMessage> : null}

      {sessions.ok && sessions.value.length === 0 ? (
        <EmptyState>No assessment sessions are assigned to you yet.</EmptyState>
      ) : null}

      {sessions.ok && waiting.length > 0 ? (
        <div className="employee-stack">
          <h3 className="employee-subheading">Waiting for you</h3>
          {waiting.map((session) => (
            <AssessmentCard key={session.id} session={session} />
          ))}
        </div>
      ) : null}

      {sessions.ok && submitted.length > 0 ? (
        <div className="employee-stack">
          <h3 className="employee-subheading">Submitted</h3>
          {submitted.map((session) => (
            <AssessmentCard key={session.id} session={session} />
          ))}
        </div>
      ) : null}
    </section>
  );
}

function AssessmentCard({ session }: { session: AssessmentSession }) {
  const open = session.status !== "submitted";
  return (
    <article className="employee-card employee-assessment-card">
      <header className="employee-card-head">
        <div>
          <h3>Assessment session</h3>
          <p className="employee-meta">
            {session.skills.length} {session.skills.length === 1 ? "skill" : "skills"}
            {session.confirmed ? " · Confirmed" : ""}
          </p>
        </div>
        <span className={statusClass(session.status)}>{sessionLabel(session.status)}</span>
      </header>

      {session.status === "submitted" && session.confirmed
        ? (session.proposals ?? []).map((proposal) => (
            <p key={proposal.capabilityId} className="employee-proposal">
              {proposalLabel(proposal.source)} {proposal.percentage}% · Provisional
            </p>
          ))
        : null}

      {open ? (
        <form action={submitSessionAnswers} className="framework-form employee-form">
          <input type="hidden" name="sessionId" value={session.id} />
          {session.skills.map((skill, index) => (
            <div key={`${session.id}-${index}`} className="employee-skill-block">
              <p className="employee-instructions">{skill.instructions}</p>
              {skill.items.map((item) => (
                <label key={item.id}>
                  {item.prompt}
                  <textarea name={`answer.${item.id}`} required maxLength={8000} defaultValue={item.answer} rows={4} />
                </label>
              ))}
            </div>
          ))}
          <button className="primary" type="submit">
            Submit answers
          </button>
        </form>
      ) : (
        <div className="employee-stack">
          {session.skills.map((skill, index) => (
            <div key={`${session.id}-${index}`} className="employee-skill-block">
              <p className="employee-instructions">{skill.instructions}</p>
              {skill.items.map((item) => (
                <div key={item.id} className="employee-answer">
                  <p className="employee-prompt">{item.prompt}</p>
                  <p className="employee-answer-body">{item.answer || "No answer recorded."}</p>
                </div>
              ))}
            </div>
          ))}
        </div>
      )}
    </article>
  );
}

function sessionLabel(status: string): string {
  switch (status) {
    case "assigned":
      return "Assigned";
    case "started":
      return "In progress";
    case "submitted":
      return "Submitted";
    default:
      return status;
  }
}

function proposalLabel(source: string): string {
  switch (source) {
    case "knowledge":
      return "Knowledge";
    case "situational_judgement":
      return "Situational judgement";
    case "ai_interview":
      return "AI interview";
    default:
      return source;
  }
}

function statusClass(status: string): string {
  if (status === "submitted") {
    return "ds-status";
  }
  if (status === "started") {
    return "ds-status ds-status-attention";
  }
  return "ds-status ds-status-neutral";
}
