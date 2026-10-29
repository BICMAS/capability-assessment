import { EmptyState } from "@/components/ui/status-message";
import { recommendForGap, saveLearningResource, saveProfileAssessment } from "@/features/role-profiles/insight-actions";
import { PracticeForm } from "@/features/role-profiles/practice-form";
import type { LearningRecommendation, LearningResource } from "@/lib/capability-insights";
import type { ProfileInsight } from "@/lib/profile-insights";
import type { RoleProfile } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

export function InsightWorkspace({
  profile,
  insight,
  resources,
  recommendations,
  filters,
}: {
  profile: RoleProfile;
  insight: Settled<ProfileInsight> | null;
  resources: Settled<LearningResource[]>;
  recommendations: Settled<LearningRecommendation[]>;
  filters: { person: string; department: string; role: string; q: string; from: string; to: string };
}) {
  if (profile.version.status !== "profile_published" || !insight?.ok) {
    return null;
  }
  const requiredSkills = profile.version.skills.filter((skill) => skill.priority === "required");
  const people = insight.value.people;
  const names = new Map(profile.version.skills.map((skill) => [skill.skillId, skill.displayName?.trim() || skill.skillName]));
  return (
    <div className="insight-panel">
      <form className="framework-form" method="get">
        <h4>Filter this published version</h4>
        <label>
          Search
          <input name="q" defaultValue={filters.q} />
        </label>
        <label>
          Employee
          <input name="person" defaultValue={filters.person} />
        </label>
        <label>
          Department
          <input name="department" defaultValue={filters.department} />
        </label>
        <label>
          Role
          <input name="role" defaultValue={filters.role} />
        </label>
        <label>
          Role assignment from
          <input name="from" type="text" defaultValue={filters.from} placeholder="2026-10-01" />
        </label>
        <label>
          Role assignment to
          <input name="to" type="text" defaultValue={filters.to} placeholder="2026-10-31" />
        </label>
        <button className="primary" type="submit">
          Apply filters
        </button>
      </form>

      <form className="framework-form" action={saveProfileAssessment}>
        <h4>Assessment</h4>
        <p className="portal-note">
          This records proficiency 0–4 against approved version {profile.version.versionNumber}. The stored percentage is the level multiplied by 25 so the gap service can recover the same level. The assessor is the signed-in person. The assessment date and provisional status are stored by the assessment service. A manager assessment is accepted for an administrator or the employee’s manager. A self-assessment is accepted for the employee or an administrator, and it is not used in the calculated level.
        </p>
        <input type="hidden" name="profileId" value={profile.id} />
        <input type="hidden" name="profileVersionId" value={profile.version.id} />
        {people.length === 0 || requiredSkills.length === 0 ? (
          <EmptyState>An assessment needs a person on this published version and a required skill.</EmptyState>
        ) : (
          <>
            <label>
              Employee
              <select name="personId" required defaultValue="">
                <option value="">Choose a person</option>
                {people.map((person) => (
                  <option key={person.personId} value={person.personId}>
                    {person.personId}
                    {person.roleName ? ` · ${person.roleName}` : ""}
                    {person.department ? ` · ${person.department}` : ""}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Skill and required proficiency
              <select name="skillId" required defaultValue="">
                <option value="">Choose a skill</option>
                {requiredSkills.map((skill) => (
                  <option key={skill.id} value={`${skill.skillId}:${skill.requiredProficiency}`}>
                    {names.get(skill.skillId) ?? skill.skillId} · required {skill.requiredProficiency}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Assessed proficiency
              <input name="assessedProficiency" type="number" min={0} max={4} required />
            </label>
            <label>
              Method
              <select name="method" required defaultValue="manager_assessment">
                <option value="manager_assessment">Manager assessment</option>
                <option value="self_assessment">Employee self-assessment</option>
              </select>
            </label>
            <label>
              Evidence or notes
              <textarea name="evidence" required maxLength={500} />
            </label>
            <button className="primary" type="submit">
              Save assessment
            </button>
          </>
        )}
      </form>

      <section>
        <h4>Learning recommendations</h4>
        <p className="portal-note">No learning system is confirming enrolment, start, or completion. A recommendation stays recommended until that confirmation exists. Voice practice is also unconfigured.</p>
        {!resources.ok ? <EmptyState>{resources.error}</EmptyState> : null}
        {resources.ok && resources.value.length === 0 ? <EmptyState>No learning resources are in the catalogue yet.</EmptyState> : null}
        {resources.ok && resources.value.length > 0 ? (
          <ul>
            {resources.value.map((resource) => (
              <li key={resource.id}>
                {resource.title} · {names.get(resource.skillId) ?? resource.skillId} · target {resource.targetLevel} · {resource.reference}
              </li>
            ))}
          </ul>
        ) : null}
        {!recommendations.ok ? <EmptyState>{recommendations.error}</EmptyState> : null}
        {recommendations.ok && recommendations.value.length === 0 ? <EmptyState>No recommendations are saved for the current access.</EmptyState> : null}
        {recommendations.ok
          ? recommendations.value.map((item) => (
              <p key={item.id}>
                {item.personId} · {names.get(item.skillId) ?? item.skillId} · intended level {item.intendedLevel} · {item.activity} · {item.rationale} · {item.reference} · {item.status}
              </p>
            ))
          : null}
        <form className="framework-form" action={saveLearningResource}>
          <input type="hidden" name="profileId" value={profile.id} />
          <label>
            Resource title
            <input name="title" required />
          </label>
          <label>
            Skill
            <select name="skillId" required defaultValue={requiredSkills[0]?.skillId ?? ""}>
              {requiredSkills.map((skill) => (
                <option key={skill.id} value={skill.skillId}>
                  {names.get(skill.skillId) ?? skill.skillId}
                </option>
              ))}
            </select>
          </label>
          <label>
            Target level
            <input name="targetLevel" type="number" min={0} max={4} required />
          </label>
          <label>
            Reference
            <input name="reference" required />
          </label>
          <label>
            Rationale
            <textarea name="rationale" required maxLength={500} />
          </label>
          <button type="submit">Save learning resource</button>
        </form>
        <form className="framework-form" action={recommendForGap}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="state" value="below" />
          <label>
            Confirmed gap
            <select name="gap" required defaultValue="">
              <option value="">Choose a person and skill below the requirement</option>
              {people.flatMap((person) =>
                person.skills
                  .filter((skill) => skill.state === "below")
                  .map((skill) => (
                    <option key={`${person.personId}-${skill.skillId}`} value={`${person.personId}:${skill.skillId}:${skill.requiredProficiency}`}>
                      {person.personId} · {names.get(skill.skillId) ?? skill.skillId} · required {skill.requiredProficiency}
                    </option>
                  )),
              )}
            </select>
          </label>
          <button type="submit">Recommend learning</button>
        </form>
      </section>
      <PracticeForm profileId={profile.id} people={people.map((person) => person.personId)} />
    </div>
  );
}
