import { CapabilityInstrument, RequiredOnly } from "@/components/capability-instrument";
import { PathwayView } from "@/components/pathway";
import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { Settled } from "@/lib/safe";
import type { FrameworkVersion } from "@/lib/capability-framework";

import { saveSelfAssessment } from "@/app/(employee)/employee/assessment-actions";

import {
  bandChipClass,
  bandLabel,
  bandSections,
  groupByFramework,
  rowBand,
  type CapabilityRow,
  type CapabilityWorkbenchData,
} from "./capability-data";

export function EmployeeCapabilities({
  personId,
  hasRoleTitles,
  frameworks,
  data,
}: {
  personId: string;
  hasRoleTitles: boolean;
  frameworks: Settled<FrameworkVersion[]>;
  data: CapabilityWorkbenchData | null;
}) {
  return (
    <section className="employee-section employee-workbench" id="capabilities" aria-labelledby="employee-capabilities-heading">
      <header className="employee-section-head">
        <h2 id="employee-capabilities-heading">Capability workbench</h2>
        <p>Grouped by framework with the same strength, developing, and priority bands used for your team view.</p>
      </header>

      {!frameworks.ok ? <StatusMessage tone="error">{frameworks.error}</StatusMessage> : null}

      {frameworks.ok && !hasRoleTitles ? (
        <EmptyState>No role is recorded on your profile yet, so capability requirements cannot be matched.</EmptyState>
      ) : null}

      {frameworks.ok && hasRoleTitles && data && data.rows.length === 0 ? (
        <EmptyState>No capability requirements match your recorded role titles yet.</EmptyState>
      ) : null}

      {data && data.levelFailures > 0 ? (
        <StatusMessage tone="error">
          {data.levelFailures === 1
            ? "1 capability level could not be loaded."
            : `${data.levelFailures} capability levels could not be loaded.`}
        </StatusMessage>
      ) : null}

      {data && data.pathwayFailures > 0 ? (
        <StatusMessage tone="info">
          {data.pathwayFailures === 1
            ? "1 development pathway could not be loaded."
            : `${data.pathwayFailures} development pathways could not be loaded.`}
        </StatusMessage>
      ) : null}

      {data && data.rows.length > 0 ? (
        <div className="employee-stack">
          {groupByFramework(data.rows).map((group) => (
            <article key={group.key} className="employee-card employee-framework-card">
              <header className="employee-card-head">
                <div>
                  <h3>
                    {group.frameworkName} v{group.versionNumber}
                  </h3>
                  <p className="employee-meta">
                    {group.rows.length} {group.rows.length === 1 ? "capability" : "capabilities"} for your role titles
                  </p>
                </div>
              </header>

              <div className="employee-band-stack">
                {bandSections(group.rows).map((section) => (
                  <section key={section.key} className="employee-band-group" aria-labelledby={`${group.key}-${section.key}`}>
                    <h4 id={`${group.key}-${section.key}`} className={section.titleClass || undefined}>
                      {section.title}
                    </h4>
                    <ul className="employee-workbench-list">
                      {section.rows.map((row) => (
                        <li key={row.id}>
                          <CapabilityWorkbenchCard row={row} personId={personId} />
                        </li>
                      ))}
                    </ul>
                  </section>
                ))}
              </div>
            </article>
          ))}
        </div>
      ) : null}
    </section>
  );
}

function CapabilityWorkbenchCard({ row, personId }: { row: CapabilityRow; personId: string }) {
  const band = rowBand(row);

  return (
    <article className={`employee-workbench-card is-${band}`}>
      <header className="employee-workbench-card-head">
        <span className={bandChipClass(band)}>{bandLabel(band)}</span>
      </header>

      <div className="employee-workbench-grid">
        <div className="employee-workbench-main">
          {!row.level.ok ? <StatusMessage tone="error">{row.level.error}</StatusMessage> : null}
          {row.level.ok && row.explanation ? <CapabilityInstrument name={row.name} explanation={row.explanation} /> : null}
          {row.level.ok && !row.explanation ? <RequiredOnly name={row.name} requiredLevel={row.requiredLevel} /> : null}

          {!row.pathwayResult.ok ? <StatusMessage tone="error">{row.pathwayResult.error}</StatusMessage> : null}
          {row.pathway ? <PathwayView pathway={row.pathway} /> : null}
        </div>

        <aside className="employee-self-panel" aria-label={`Self-assessment for ${row.name}`}>
          <form action={saveSelfAssessment} className="framework-form employee-form employee-self-form">
            <input type="hidden" name="personId" value={personId} />
            <input type="hidden" name="frameworkVersionId" value={row.frameworkVersionId} />
            <input type="hidden" name="capabilityCode" value={row.capabilityCode} />
            <input type="hidden" name="requiredLevel" value={row.requiredLevel} />
            <div className="employee-self-panel-head">
              <h4>Self-assessment</h4>
              <p className="portal-note">Stored as a self-assessment. It is not included in the calculated level.</p>
            </div>
            <label>
              Proficiency (0–4)
              <input name="assessedProficiency" type="number" min={0} max={4} required inputMode="numeric" />
            </label>
            <label>
              Evidence
              <input name="evidence" type="text" required autoComplete="off" />
            </label>
            <button type="submit" className="primary">
              Save self-assessment
            </button>
          </form>
        </aside>
      </div>
    </article>
  );
}
