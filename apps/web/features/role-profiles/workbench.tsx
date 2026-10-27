import { EmptyState } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile, RoleProfileSkill } from "@/lib/role-profile";

import { addSkillForm, createCanonicalSkillForm, removeSkillForm, reorderSkillForm, updateSkillForm } from "./actions";
import { recordedKind } from "./kind";

const categories = ["technical", "leadership", "domain", "communication", "compliance"];
const priorities = ["required", "preferred", "optional"];

export function Workbench({
  profile,
  skillId = "",
  category = "",
}: {
  profile: RoleProfile;
  skillId?: string;
  category?: string;
}) {
  const version = profile.version;
  const editable = version.status === "profile_draft";
  const skills = [...version.skills].sort((left, right) => left.position - right.position);
  return (
    <section className="directory">
      <h2>Skills</h2>
      <p>
        Status <strong>{version.status}</strong>. Revision {version.revision}. AI suggestions and HR adjustments use the provenance stored by the service.
      </p>
      {skills.length === 0 ? <EmptyState>No skills are assigned.</EmptyState> : null}
      <div className="skill-scroll">
        <table className="skill-table">
          <thead>
            <tr>
              <th scope="col">Skill</th>
              <th scope="col">Category</th>
              <th scope="col">Proficiency</th>
              <th scope="col">Provenance</th>
              <th scope="col">Priority</th>
              <th scope="col">Review status</th>
            </tr>
          </thead>
          <tbody>
            {skills.map((skill, index) => (
              <SkillRow key={skill.id} skill={skill} profile={profile} editable={editable} index={index} total={skills.length} ids={skills.map((item) => item.id)} kind={recordedKind(version.ai.draft, skill.skillId)} />
            ))}
          </tbody>
        </table>
      </div>
      {editable ? (
        <>
          <h3>Add a canonical skill</h3>
          <p>The service does not list the taxonomy. Create a skill, then assign the returned id.</p>
          <form action={createCanonicalSkillForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <label>
              Name
              <input name="name" required />
            </label>
            <label>
              Category
              <select name="category" defaultValue="technical">
                {categories.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Definition
              <textarea name="definition" required rows={2} />
            </label>
            <PendingButton>Create canonical skill</PendingButton>
          </form>
          <h3>Assign a skill</h3>
          <form action={addSkillForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <label>
              Canonical skill id
              <input name="skillId" defaultValue={skillId} required />
            </label>
            <label>
              Category
              <select name="category" defaultValue={category || "technical"}>
                {categories.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Definition
              <textarea name="definition" required rows={2} />
            </label>
            <label>
              Required proficiency
              <input name="requiredProficiency" inputMode="numeric" required />
            </label>
            <label>
              Priority
              <select name="priority" defaultValue="required">
                {priorities.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Rationale
              <textarea name="rationale" required rows={2} />
            </label>
            <label>
              Provenance
              <select name="provenance" defaultValue="manually_added">
                <option value="manually_added">manually_added</option>
                <option value="imported">imported</option>
              </select>
            </label>
            <PendingButton className="primary">Add skill</PendingButton>
          </form>
        </>
      ) : null}
    </section>
  );
}

function SkillRow({
  skill,
  profile,
  editable,
  index,
  total,
  ids,
  kind,
}: {
  skill: RoleProfileSkill;
  profile: RoleProfile;
  editable: boolean;
  index: number;
  total: number;
  ids: string[];
  kind: string;
}) {
  const version = profile.version;
  return (
    <tr>
      <td>
        <strong>{skill.skillName}</strong>
        <p>{skill.definition}</p>
        <p>Rationale: {skill.rationale || "None recorded"}</p>
        <p>Evidence indicator: none is stored on this assignment.</p>
        {editable ? (
          <form action={updateSkillForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="skillAssignmentId" value={skill.id} />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <label>
              Definition
              <textarea name="definition" defaultValue={skill.definition} required rows={2} />
            </label>
            <label>
              Required proficiency
              <input name="requiredProficiency" inputMode="numeric" defaultValue={skill.requiredProficiency} required />
            </label>
            <label>
              Priority
              <select name="priority" defaultValue={skill.priority}>
                {priorities.map((item) => (
                  <option key={item} value={item}>
                    {item}
                  </option>
                ))}
              </select>
            </label>
            <label>
              Rationale
              <textarea name="rationale" defaultValue={skill.rationale} required rows={2} />
            </label>
            <PendingButton>Save skill</PendingButton>
          </form>
        ) : null}
      </td>
      <td>
        {skill.category}
        {kind ? <span className="kind-note">Recorded kind: {kind}</span> : null}
      </td>
      <td>{skill.requiredProficiency}</td>
      <td>
        <span className={provenanceClass(skill.provenance)}>{skill.provenance}</span>
      </td>
      <td>
        {skill.priority}
        {editable ? (
          <form action={reorderSkillForm}>
            {ids.map((id) => (
              <input key={id} type="hidden" name="skillAssignmentId" value={id} />
            ))}
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <input type="hidden" name="moveId" value={skill.id} />
            <button type="submit" name="direction" value="up" disabled={index === 0}>
              Move {skill.skillName} up
            </button>
            <button type="submit" name="direction" value="down" disabled={index === total - 1}>
              Move {skill.skillName} down
            </button>
          </form>
        ) : null}
      </td>
      <td>
        {version.status}
        {editable ? (
          <form action={removeSkillForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <input type="hidden" name="versionId" value={version.id} />
            <input type="hidden" name="skillAssignmentId" value={skill.id} />
            <input type="hidden" name="expectedRevision" value={version.revision} />
            <label>
              <input type="checkbox" name="confirm" value="yes" required /> Confirm removal of {skill.skillName}
            </label>
            <PendingButton>Remove skill</PendingButton>
          </form>
        ) : null}
      </td>
    </tr>
  );
}

function provenanceClass(provenance: string): string {
  if (provenance === "ai_suggested") {
    return "provenance provenance-ai";
  }
  if (provenance === "hr_adjusted") {
    return "provenance provenance-hr";
  }
  return "provenance";
}
