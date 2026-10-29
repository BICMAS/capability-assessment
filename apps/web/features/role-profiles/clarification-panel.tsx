import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { ClarificationQuestion, ClarificationSession } from "@/lib/role-profile";
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
            Context version {session.value.contextVersion}. Round {session.value.round}. {session.value.complete ? "No further clarification is required." : "Critical information is still unresolved."}
          </p>
          {followUp(session.value.questions) ? <p className={modelFailed(session.value.questions) ? "role-missing" : undefined}>{followUp(session.value.questions)}</p> : null}
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
              {question.reason ? <p>{question.reason}</p> : null}
              <p>
                {question.field}
                {question.critical ? " · critical" : ""} · {authorLabel(question.authoredBy)}
              </p>
              {question.answeredAt ? <p>Answer: {question.answer === "" ? "Unknown" : question.answer}</p> : null}
              {editable ? (
                <>
                  <form action={answerClarificationForm} className="framework-form">
                    <input type="hidden" name="profileId" value={profileId} />
                    <input type="hidden" name="versionId" value={versionId} />
                    <input type="hidden" name="questionId" value={question.id} />
                    {question.options && question.options.length > 0 ? (
                      <label>
                        {question.answeredAt ? "Correct the answer" : "Answer"}
                        <select name="value" defaultValue={question.answer}>
                          <option value="">Select a level</option>
                          {question.options.map((option) => (
                            <option key={option} value={option}>
                              {option.replaceAll("_", " ")}
                            </option>
                          ))}
                        </select>
                      </label>
                    ) : (
                      <label>
                        {question.answeredAt ? "Correct the answer" : "Answer"}
                        <textarea name="value" rows={3} defaultValue={question.answer} />
                      </label>
                    )}
                    <PendingButton className="primary">{question.answeredAt ? "Save correction" : "Save answer"}</PendingButton>
                  </form>
                  {!question.answeredAt ? (
                    <form action={answerClarificationForm}>
                      <input type="hidden" name="profileId" value={profileId} />
                      <input type="hidden" name="versionId" value={versionId} />
                      <input type="hidden" name="questionId" value={question.id} />
                      <input type="hidden" name="value" value="" />
                      <PendingButton>Mark unknown</PendingButton>
                    </form>
                  ) : null}
                </>
              ) : question.answeredAt ? null : (
                <p>Unanswered</p>
              )}
            </article>
          ))}
        </>
      ) : null}
    </section>
  );
}

function followUp(questions: ClarificationQuestion[]): string {
  if (questions.some((question) => question.authoredBy === "service_after_model_error")) {
    return "The model could not be reached. These questions were prepared from the recorded role information. They are not a live model response. Answer them to continue, then check again if you want another round.";
  }
  if (questions.some((question) => question.authoredBy === "model")) {
    return "The model suggested wording for the short fields. It did not fill in the role information.";
  }
  if (questions.length > 0) {
    return "These questions come from gaps in the recorded role information. A live model was not used.";
  }
  return "";
}

function modelFailed(questions: ClarificationQuestion[]): boolean {
  return questions.some((question) => question.authoredBy === "service_after_model_error");
}

function authorLabel(authoredBy: string | undefined): string {
  if (authoredBy === "model") {
    return "wording suggested by the model";
  }
  if (authoredBy === "service_after_model_error") {
    return "prepared after the model failed";
  }
  return "prepared from the recorded role";
}
