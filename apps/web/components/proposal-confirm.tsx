import type { AssessmentSession } from "@/lib/generated/assessment-plan/plans";

const sourceLabels: Record<string, string> = {
  knowledge: "Knowledge",
  situational_judgement: "Situational judgement",
  ai_interview: "AI interview",
};

export function ProposalConfirm({
  personId,
  sessions,
  action,
  hidden,
}: {
  personId: string;
  sessions: AssessmentSession[];
  action: (formData: FormData) => Promise<void>;
  hidden?: { name: string; value: string }[];
}) {
  const pending = sessions.filter((session) => session.status === "submitted" && session.proposals && session.proposals.length > 0 && !session.confirmed);
  if (pending.length === 0) {
    return null;
  }
  return (
    <section className="team-group">
      <h3>Proposed results</h3>
      {pending.map((session) => (
        <article key={session.id} className="capability-figures">
          {(session.proposals ?? []).map((proposal) => (
            <p key={`${session.id}-${proposal.capabilityId}`} className="proposal-line">
              {sourceLabels[proposal.source] ?? proposal.source}{" "}
              <span className="proposal-percentage">{proposal.percentage}</span>
            </p>
          ))}
          <form action={action}>
            <input type="hidden" name="personId" value={personId} />
            <input type="hidden" name="sessionId" value={session.id} />
            {(hidden ?? []).map((field) => (
              <input key={field.name} type="hidden" name={field.name} value={field.value} />
            ))}
            <button className="primary" type="submit">Confirm</button>
          </form>
        </article>
      ))}
    </section>
  );
}
