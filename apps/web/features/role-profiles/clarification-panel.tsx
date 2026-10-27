import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { ClarificationSession } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import { answerClarificationForm, startClarificationForm } from "./actions";

export function ClarificationPanel({
  profileId,
  versionId,
  editable,
  session,
}: {
  profileId: string;
  versionId: string;
  editable: boolean;
  session: Settled<ClarificationSession>;
}) {
  return (
    <section className="directory">
      <h2>Clarification</h2>
      {!session.ok ? <StatusMessage tone="error">{session.error}</StatusMessage> : null}
      {editable ? (
        <form action={startClarificationForm}>
          <input type="hidden" name="profileId" value={profileId} />
          <input type="hidden" name="versionId" value={versionId} />
          <PendingButton className="primary">Check for questions</PendingButton>
        </form>
      ) : (
        <p>Clarification is available on a draft.</p>
      )}
      {session.ok ? (
        <>
          <p>
            Context version {session.value.contextVersion}. Round {session.value.round}. {session.value.complete ? "No critical field is unresolved." : "Critical information is still unresolved."}
          </p>
          {session.value.unresolved.length > 0 ? (
            <ul>
              {session.value.unresolved.map((field) => (
                <li key={field}>{field} is unresolved</li>
              ))}
            </ul>
          ) : null}
          {session.value.questions.length === 0 ? <EmptyState>No questions are stored for this version.</EmptyState> : null}
          {session.value.questions.map((question) => (
            <article key={question.id} className="studio-version">
              <h3>{question.question}</h3>
              <p>
                {question.field}
                {question.critical ? " · critical" : ""} · asked by {question.askedActorId}
              </p>
              {question.answeredAt ? (
                <p>Answer: {question.answer === "" ? "Unknown" : question.answer}</p>
              ) : editable ? (
                <>
                  <form action={answerClarificationForm} className="framework-form">
                    <input type="hidden" name="profileId" value={profileId} />
                    <input type="hidden" name="versionId" value={versionId} />
                    <input type="hidden" name="questionId" value={question.id} />
                    <label>
                      Answer
                      <textarea name="value" rows={3} />
                    </label>
                    <PendingButton className="primary">Save answer</PendingButton>
                  </form>
                  <form action={answerClarificationForm}>
                    <input type="hidden" name="profileId" value={profileId} />
                    <input type="hidden" name="versionId" value={versionId} />
                    <input type="hidden" name="questionId" value={question.id} />
                    <input type="hidden" name="value" value="" />
                    <PendingButton>Mark unknown</PendingButton>
                  </form>
                </>
              ) : (
                <p>Unanswered</p>
              )}
            </article>
          ))}
        </>
      ) : null}
    </section>
  );
}
