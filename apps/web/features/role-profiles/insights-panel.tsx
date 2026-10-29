import { EmptyState, StatusMessage } from "@/components/ui/status-message";
import type { InsightSummary, ProfileInsight } from "@/lib/profile-insights";
import type { RoleProfile } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

export function InsightsPanel({ profile, insight }: { profile: RoleProfile; insight: Settled<ProfileInsight> | null }) {
  if (profile.version.status !== "profile_published") {
    return <EmptyState>Insights are available after this version is published.</EmptyState>;
  }
  if (!insight || !insight.ok) {
    return <StatusMessage tone="error">{insight && !insight.ok ? insight.error : "Insights are unavailable right now."}</StatusMessage>;
  }
  const names = new Map(profile.version.skills.map((skill) => [skill.skillId, skill.skillName]));
  return (
    <section className="insight-panel" aria-labelledby="insight-heading">
      <h3 id="insight-heading">Published version {profile.version.versionNumber}</h3>
      {insight.value.employmentDecision === "not_an_employment_decision" ? (
        <p className="portal-note">This result is not an employment decision. A missing assessment is not recorded as proficiency zero.</p>
      ) : (
        <p className="portal-note">Service decision label: {insight.value.employmentDecision}</p>
      )}
      {insight.value.people.length === 0 ? <EmptyState>No assessed people are visible for this published version.</EmptyState> : null}
      {insight.value.summary ? <InsightSummaryView summary={insight.value.summary} names={names} /> : null}
      {insight.value.people.map((person) => (
        <article key={person.personId}>
          <h4>{person.personId}</h4>
          <table>
            <caption>Assessment states</caption>
            <thead>
              <tr>
                <th>Skill</th>
                <th>Required level</th>
                <th>State</th>
                <th>Recorded level</th>
              </tr>
            </thead>
            <tbody>
              {person.skills.map((skill) => (
                <tr key={skill.skillId}>
                  <td>{names.get(skill.skillId) ?? skill.skillId}</td>
                  <td>{skill.requiredProficiency}</td>
                  <td>{skill.state}</td>
                  <td>{skill.currentLevel ? skill.currentLevel : "Not assessed"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </article>
      ))}
    </section>
  );
}

function InsightSummaryView({ summary, names }: { summary: InsightSummary; names: Map<string, string> }) {
  return (
    <div>
      <h4>Capability progress</h4>
      <p className="portal-note">{summary.method}</p>
      <dl className="figures">
        <div>
          <dt>Required skill rows</dt>
          <dd>{summary.requiredRows}</dd>
        </div>
        <div>
          <dt>Recorded levels</dt>
          <dd>{summary.assessedRows}</dd>
        </div>
        <div>
          <dt>Missing assessments</dt>
          <dd>{summary.missingRows}</dd>
        </div>
        <div>
          <dt>Below the requirement</dt>
          <dd>{summary.belowRows}</dd>
        </div>
        <div>
          <dt>Meets or exceeds</dt>
          <dd>{summary.metRows}</dd>
        </div>
      </dl>
      <p className="portal-note">
        Coverage is {summary.assessedRows} recorded {summary.assessedRows === 1 ? "level" : "levels"} out of {summary.requiredRows} required skill {summary.requiredRows === 1 ? "row" : "rows"}.
      </p>
      <p className="portal-note">Gap closure over time is not shown because only the latest stored level is available.</p>
      {summary.criticalSkillIds.length === 0 ? (
        <EmptyState>No confirmed skill shortage is recorded. A missing assessment is not counted as a shortage.</EmptyState>
      ) : (
        <ul>
          {summary.criticalSkillIds.map((skillId) => (
            <li key={skillId}>Critical shortage: {names.get(skillId) ?? skillId}</li>
          ))}
        </ul>
      )}
      <h4>Capability by department</h4>
      {summary.departments.length === 0 ? <EmptyState>No department breakdown is available for the current filter.</EmptyState> : null}
      {summary.departments.length > 0 ? (
        <table>
          <caption>Department coverage</caption>
          <thead>
            <tr>
              <th>Department</th>
              <th>Required</th>
              <th>Missing assessments</th>
              <th>Below</th>
              <th>Met</th>
            </tr>
          </thead>
          <tbody>
            {summary.departments.map((department) => (
              <tr key={department.department}>
                <td>{department.department}</td>
                <td>{department.requiredRows}</td>
                <td>{department.missingRows}</td>
                <td>{department.belowRows}</td>
                <td>{department.metRows}</td>
              </tr>
            ))}
          </tbody>
        </table>
      ) : null}
      <h4>Workforce planning</h4>
      <p className="portal-note">
        Organisational priorities are the confirmed shortages above. People who meet the required level are the succession-planning inputs available from these records. No separate successor register is configured. Future development needs follow the confirmed gaps. Learning allocation uses saved recommendations, and a course is not marked enrolled, started, or completed here.
      </p>
    </div>
  );
}
