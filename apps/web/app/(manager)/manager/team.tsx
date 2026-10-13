import type { FrameworkVersion } from "@/lib/capability-framework";
import { readLevel, type Explanation } from "@/lib/generated/assessment-gap/levels";

import { saveManagerAssessment } from "./actions";

export type TeamCapability = {
  name: string;
  capabilityCode: string;
  frameworkVersionId: string;
  requiredLevel: number;
  explanation: Explanation | null;
};

type AssessedCapability = TeamCapability & { explanation: Explanation };

export function TeamRow({
  personId,
  personName,
  roles,
  capabilities,
}: {
  personId: string;
  personName: string;
  roles: string[];
  capabilities: TeamCapability[];
}) {
  const strengths = capabilities.filter(isBand("strength"));
  const developing = capabilities.filter(isBand("developing"));
  const priority = capabilities.filter(isBand("priority"));

  return (
    <a className="team-row" href={`/manager/people/${personId}`}>
      <span className="team-name">{personName}</span>
      <span className="team-role">{roles.join(", ")}</span>
      <span className="team-gap">
        {priority.map((item) => (
          <span key={item.capabilityCode}>
            {item.name} {item.explanation.gap}
          </span>
        ))}
      </span>
      <span className="team-bands">
        <Band label="Strengths" tone="band-strength" items={strengths} />
        <Band label="Developing areas" items={developing} />
        <Band label="Priority gaps" tone="band-priority" items={priority} />
      </span>
    </a>
  );
}

export function PersonRecord({
  personId,
  personName,
  roles,
  capabilities,
}: {
  personId: string;
  personName: string;
  roles: string[];
  capabilities: TeamCapability[];
}) {
  const strengths = capabilities.filter(isBand("strength"));
  const developing = capabilities.filter(isBand("developing"));
  const priority = capabilities.filter(isBand("priority"));

  return (
    <>
      <p className="team-kicker">{roles.join(", ")}</p>
      <h2>{personName}</h2>
      <section className="team-group">
        <h3 className="band-strength">Strengths</h3>
        <FigureList items={strengths} />
      </section>
      <section className="team-group">
        <h3>Developing areas</h3>
        <FigureList items={developing} />
      </section>
      <section className="team-group">
        <h3 className="band-priority">Priority gaps</h3>
        {priority.length === 0 ? <p className="team-empty">None</p> : null}
        {priority.map((capability) => (
          <article key={capability.capabilityCode} className="capability-figures">
            <LevelFigures capability={capability} />
            <form className="framework-form" action={saveManagerAssessment}>
              <input type="hidden" name="personId" value={personId} />
              <input type="hidden" name="frameworkVersionId" value={capability.frameworkVersionId} />
              <input type="hidden" name="capabilityCode" value={capability.capabilityCode} />
              <input type="hidden" name="requiredLevel" value={capability.requiredLevel} />
              <label>
                Manager assessment for {capability.name}
                <input name="percentage" type="number" min={0} max={100} required defaultValue="" />
              </label>
              <label>
                Evidence
                <input name="evidence" type="text" required />
              </label>
              <button type="submit">Save manager assessment</button>
            </form>
          </article>
        ))}
      </section>
    </>
  );
}

function FigureList({ items }: { items: AssessedCapability[] }) {
  if (items.length === 0) {
    return <p className="team-empty">None</p>;
  }
  return items.map((capability) => (
    <article key={capability.capabilityCode} className="capability-figures">
      <LevelFigures capability={capability} />
    </article>
  ));
}

function LevelFigures({ capability }: { capability: AssessedCapability }) {
  const explanation = capability.explanation;
  return (
    <>
      <h3>{capability.name}</h3>
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
    </>
  );
}

function Band({
  label,
  tone,
  items,
}: {
  label: string;
  tone?: string;
  items: AssessedCapability[];
}) {
  return (
    <span className="team-band">
      <span className={tone ? `team-band-label ${tone}` : "team-band-label"}>{label}</span>
      <span className="team-band-names">{items.length === 0 ? "None" : items.map((item) => item.name).join(", ")}</span>
    </span>
  );
}

function isBand(band: "strength" | "developing" | "priority") {
  return (item: TeamCapability): item is AssessedCapability => item.explanation !== null && gapBand(item.explanation.gap) === band;
}

function gapTone(gap: string): string {
  const band = gapBand(gap);
  if (band === "strength") {
    return "tone-strength";
  }
  if (band === "priority") {
    return "tone-gap";
  }
  return "";
}

function gapBand(gap: string): "strength" | "developing" | "priority" {
  if (gap.startsWith("-") || gap === "0.0") {
    return "strength";
  }
  if (gap.startsWith("0.")) {
    return "developing";
  }
  return "priority";
}

export async function loadCapabilities(
  token: string,
  personId: string,
  roleTitles: string[],
  frameworks: FrameworkVersion[],
): Promise<TeamCapability[]> {
  const titles = new Set(roleTitles);
  const loaded = await Promise.all(
    frameworks.map(async (framework) => {
      const relevant = capabilitiesFor(framework, titles, new Map());
      const explanations = new Map<string, Explanation | null>(
        await Promise.all(
          relevant.map(async (capability) => {
            const explanation = await readLevel({
              token,
              personId,
              frameworkVersionId: capability.frameworkVersionId,
              capabilityCode: capability.capabilityCode,
            });
            return [capability.capabilityCode, explanation] as const;
          }),
        ),
      );
      return capabilitiesFor(framework, titles, explanations);
    }),
  );
  return loaded.flat();
}

export function capabilitiesFor(
  framework: FrameworkVersion,
  roleTitles: Set<string>,
  explanations: Map<string, Explanation | null>,
): TeamCapability[] {
  return framework.requirements.flatMap((requirement) => {
    const role = framework.roles.find((item) => item.id === requirement.roleId);
    const capability = framework.capabilities.find((item) => item.id === requirement.capabilityId);
    if (!role || !capability || !roleTitles.has(role.name)) {
      return [];
    }
    return [
      {
        name: capability.name,
        capabilityCode: capability.id,
        frameworkVersionId: framework.id,
        requiredLevel: requirement.requiredLevel,
        explanation: explanations.get(capability.id) ?? null,
      },
    ];
  });
}
