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

export function LevelSources({ explanation }: { explanation: Explanation | null }) {
  return (
    <div className="sources">
      {sources.map((source) => {
        const percentage = explanation ? source.read(explanation) : null;
        const reading = percentage === null ? "not yet assessed" : String(percentage);
        return (
          <label key={source.label}>
            {source.label}
            <input readOnly aria-label={source.label} value={percentage === null ? "" : reading} />
            <span className={percentage === null ? "source-reading source-empty" : "source-reading source-figure"}>
              {reading}
            </span>
          </label>
        );
      })}
    </div>
  );
}

export function LevelFacts({
  name,
  explanation,
}: {
  name: string;
  explanation: Explanation;
}) {
  const currentLabel = explanation.status === "provisional" ? "Provisional current level" : "Current level";
  return (
    <article>
      <h3>{name}</h3>
      <p className="level-sentence">
        {name} required {explanation.requiredLevel}, {currentLabel.toLowerCase()} {explanation.currentLevel}, gap{" "}
        {explanation.gap}
      </p>
      <dl className="metrics">
        <div>
          <dt>Required level</dt>
          <dd className="metric">{explanation.requiredLevel}</dd>
        </div>
        <div>
          <dt>{currentLabel}</dt>
          <dd className="metric">{explanation.currentLevel}</dd>
        </div>
        <div>
          <dt>Gap</dt>
          <dd className={`metric ${gapTone(explanation.gap)}`}>{explanation.gap}</dd>
        </div>
      </dl>
      <LevelSources explanation={explanation} />
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
