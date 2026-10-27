import type { Explanation } from "@/lib/generated/assessment-gap/levels";

const sources: Array<{
  label: string;
  read: (explanation: Explanation) => number | null;
}> = [
  { label: "Knowledge", read: (explanation) => explanation.knowledgePercentage },
  { label: "Situational judgement", read: (explanation) => explanation.situationalJudgementPercentage },
  { label: "AI interview", read: (explanation) => explanation.aiInterviewPercentage },
  { label: "Simulation", read: (explanation) => explanation.simulationPercentage },
  { label: "Manager assessment", read: (explanation) => explanation.managerAssessmentPercentage },
];

export function EvidenceRow({ explanation }: { explanation: Explanation | null }) {
  if (!explanation) {
    return null;
  }
  const scored = sources.flatMap((source) => {
    const percentage = source.read(explanation);
    return percentage === null ? [] : [{ label: source.label, percentage }];
  });
  const missing = sources.filter((source) => source.read(explanation) === null).map((source) => source.label);

  return (
    <>
      {scored.length > 0 ? (
        <div className="measures">
          {scored.map((source) => (
            <div key={source.label} className="measure">
              <span className="measure-label">{source.label}</span>
              <span className="measure-track" aria-hidden="true">
                <span className="measure-fill" style={{ width: `${source.percentage}%` }} />
              </span>
              <input readOnly aria-label={source.label} value={String(source.percentage)} />
              <span className="measure-reading">{source.percentage}</span>
            </div>
          ))}
        </div>
      ) : null}
      {missing.length > 0 ? <p className="portal-note">Still to assess: {missing.join(", ")}</p> : null}
    </>
  );
}

export function CapabilityInstrument({
  name,
  explanation,
}: {
  name: string;
  explanation: Explanation;
}) {
  return (
    <article className="instrument">
      <h3>{name}</h3>
      <dl className="figures">
        <div>
          <dt>Required</dt>
          <dd>{explanation.requiredLevel}</dd>
        </div>
        <div>
          <dt>Current</dt>
          <dd>{explanation.currentLevel}</dd>
          {explanation.status === "provisional" ? <p className="provisional">Provisional</p> : null}
        </div>
        <div>
          <dt>Gap</dt>
          <dd className={gapTone(explanation.gap)}>{explanation.gap}</dd>
        </div>
      </dl>
      <EvidenceRow explanation={explanation} />
    </article>
  );
}

function gapTone(gap: string): string {
  if (gap.startsWith("-") || gap === "0.0") {
    return "tone-strength";
  }
  if (gap.startsWith("0.")) {
    return "";
  }
  return "tone-gap";
}

export function RequiredOnly({ name, requiredLevel }: { name: string; requiredLevel: number }) {
  return (
    <article className="instrument instrument-quiet">
      <h3>{name}</h3>
      <p>Required level {requiredLevel}</p>
      <p className="portal-note">Not yet assessed</p>
    </article>
  );
}
