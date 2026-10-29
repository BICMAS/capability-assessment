"use client";

import Link from "next/link";
import { useEffect, useMemo, useState, useTransition } from "react";

import { EmptyState } from "@/components/ui/status-message";
import { PendingButton } from "@/components/ui/pending-button";
import type { RoleProfile, RoleProfileHistory, RoleProfileSkill } from "@/lib/role-profile";
import type { Settled } from "@/lib/safe";

import {
  addSkillForm,
  commentOnRoleProfileForm,
  createCanonicalSkillForm,
  mergeSkillsForm,
  removeSkillForm,
  saveSkillOrderForm,
  saveSkillReviewForm,
  submitRoleProfileForm,
} from "./actions";
import { recordedKind } from "./kind";
import { proficiencyLabel, reportingLabel, sourceLabel } from "./skills-view";
import { duplicatePairs, draftsDiffer, generatedSkills, moveItem, reviewCategories, reviewDraft, skillTitle, suggestionGaps, type ReviewDraft } from "./review-skills";

const priorities = ["required", "preferred", "optional"];

export function Workbench({
  profile,
  history,
  skillId = "",
  category = "",
}: {
  profile: RoleProfile;
  history?: Settled<RoleProfileHistory>;
  skillId?: string;
  category?: string;
}) {
  const version = profile.version;
  const editable = version.status === "profile_draft";
  const skills = useMemo(() => [...version.skills].sort((left, right) => left.position - right.position), [version.skills]);
  const initial = useMemo(() => Object.fromEntries(skills.map((skill) => [skill.id, reviewDraft(skill)])), [skills]);
  const [drafts, setDrafts] = useState<Record<string, ReviewDraft>>(initial);
  const [search, setSearch] = useState("");
  const [level, setLevel] = useState("");
  const [dragId, setDragId] = useState("");
  const [, startTransition] = useTransition();
  useEffect(() => {
    setDrafts(initial);
  }, [initial]);
  const dirtyIds = skills.filter((skill) => drafts[skill.id] && draftsDiffer(drafts[skill.id], initial[skill.id])).map((skill) => skill.id);
  const dirty = dirtyIds.length > 0;
  useEffect(() => {
    if (!dirty) {
      return;
    }
    const warn = (event: BeforeUnloadEvent) => {
      event.preventDefault();
    };
    window.addEventListener("beforeunload", warn);
    return () => window.removeEventListener("beforeunload", warn);
  }, [dirty]);
  const query = search.trim().toLowerCase();
  const visible = skills.filter((skill) => {
    const draft = drafts[skill.id] ?? initial[skill.id];
    if (!draft) {
      return false;
    }
    if (level && draft.requiredProficiency !== level) {
      return false;
    }
    if (!query) {
      return true;
    }
    return `${draft.displayName} ${draft.definition} ${draft.rationale}`.toLowerCase().includes(query);
  });
  const customised = skills.filter((skill) => skill.provenance !== "ai_suggested").length;
  const suggestions = generatedSkills(version.ai.draft);
  const removedSuggestions = suggestions.filter((item) => !skills.some((skill) => skill.skillId === item.skillId));
  const edits = dirtyIds.flatMap((id) => {
    const draft = drafts[id];
    const skill = skills.find((item) => item.id === id);
    if (!draft || !skill) {
      return [];
    }
    return [
      {
        skillAssignmentId: id,
        displayName: draft.displayName,
        category: draft.category,
        definition: draft.definition,
        requiredProficiency: Number(draft.requiredProficiency),
        priority: draft.priority,
        rationale: draft.rationale,
        criteria: draft.criteria,
      },
    ];
  });
  function updateDraft(id: string, patch: Partial<ReviewDraft>) {
    setDrafts((current) => ({ ...current, [id]: { ...(current[id] ?? initial[id]), ...patch } }));
  }
  function persistOrder(ids: string[]) {
    const data = new FormData();
    data.set("profileId", profile.id);
    data.set("versionId", version.id);
    data.set("expectedRevision", String(version.revision));
    ids.forEach((id) => data.append("skillAssignmentId", id));
    startTransition(() => {
      void saveSkillOrderForm(data);
    });
  }
  return (
    <section className="directory">
      <h2>{profile.roleName}</h2>
      <p>
        Version {version.versionNumber} · revision {version.revision} · status <strong>{version.status}</strong>
      </p>
      <dl className="generation-meta">
        <div>
          <dt>Purpose</dt>
          <dd>{version.purpose || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Industry</dt>
          <dd>{version.industry || "Not recorded"}</dd>
        </div>
        <div>
          <dt>Responsibility level</dt>
          <dd>{reportingLabel(version.reportingLevel)}</dd>
        </div>
        <div>
          <dt>Organogram position</dt>
          <dd>{version.positionId || "Not recorded"}</dd>
        </div>
      </dl>
      <p>
        Review progress: {customised} of {skills.length} skills have a human source. AI suggestion skills stay suggestions until a person changes them.
      </p>
      {dirty ? <p className="skill-suggestion">Unsaved changes</p> : <p>All saved changes are on this draft.</p>}
      <div className="skill-filters">
        <label>
          Search
          <input value={search} onChange={(event) => setSearch(event.target.value)} type="search" />
        </label>
        <label>
          Required level
          <select value={level} onChange={(event) => setLevel(event.target.value)}>
            <option value="">All levels</option>
            {[0, 1, 2, 3, 4].map((item) => (
              <option key={item} value={String(item)}>
                {proficiencyLabel(item)}
              </option>
            ))}
          </select>
        </label>
      </div>
      {editable ? (
        <form action={saveSkillReviewForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <input type="hidden" name="edits" value={JSON.stringify(edits)} />
          <PendingButton className="primary" disabled={!dirty}>
            Save changes
          </PendingButton>
        </form>
      ) : (
        <p>Skill editing is closed for this status. The approval step follows the existing review permissions.</p>
      )}
      {skills.length === 0 ? <EmptyState>No skills are on this draft.</EmptyState> : null}
      {query && visible.length === 0 ? <p>No skills match this view.</p> : null}
      {reviewCategories.map((group) => {
        const rows = visible.filter((skill) => (drafts[skill.id] ?? initial[skill.id])?.category === group);
        if (query && rows.length === 0) {
          return null;
        }
        return (
          <section key={group}>
            <h3>
              {group} ({rows.length})
            </h3>
            {rows.length === 0 ? <p>No skills in this category.</p> : null}
            {rows.map((skill) => (
              <SkillEditor
                key={skill.id}
                skill={skill}
                profile={profile}
                draft={drafts[skill.id] ?? initial[skill.id]}
                editable={editable}
                changed={suggestionGaps(skill, version.ai.draft).length > 0 || skill.provenance === "hr_adjusted"}
                unsaved={dirtyIds.includes(skill.id)}
                index={skills.findIndex((item) => item.id === skill.id)}
                total={skills.length}
                kind={recordedKind(version.ai.draft, skill.skillId)}
                onChange={(patch) => updateDraft(skill.id, patch)}
                onMove={(direction) => {
                  const index = skills.findIndex((item) => item.id === skill.id);
                  persistOrder(moveItem(skills.map((item) => item.id), skill.id, direction === "up" ? index - 1 : index + 1));
                }}
                onDragStart={() => setDragId(skill.id)}
                onDrop={() => {
                  const index = skills.findIndex((item) => item.id === skill.id);
                  if (dragId) {
                    persistOrder(moveItem(skills.map((item) => item.id), dragId, index));
                  }
                  setDragId("");
                }}
              />
            ))}
          </section>
        );
      })}
      {editable ? <DuplicateList profile={profile} skills={skills} /> : null}
      <details>
        <summary>Compare with the generated suggestions</summary>
        {suggestions.length === 0 ? <p>No generated suggestion is stored on this version.</p> : null}
        {skills.map((skill) => {
          const original = suggestions.find((item) => item.skillId === skill.skillId);
          const gaps = suggestionGaps(skill, version.ai.draft);
          if (!original) {
            return <p key={skill.id}>{skillTitle(skill)} was added after generation.</p>;
          }
          if (gaps.length === 0) {
            return <p key={skill.id}>{skillTitle(skill)} matches the generated suggestion.</p>;
          }
          return (
            <p key={skill.id}>
              {skillTitle(skill)} changed from the suggestion ({gaps.join(", ")}). Original definition: {original.definition || "Not recorded"}
            </p>
          );
        })}
        {removedSuggestions.map((item) => (
          <p key={item.skillId}>Generated skill {item.skillId} was removed after generation.</p>
        ))}
      </details>
      <h3>Change history</h3>
      {history && !history.ok ? <p>{history.error}</p> : null}
      {(!history || history.ok) && (history?.value.events.length ?? 0) === 0 && (history?.value.changes?.length ?? 0) === 0 ? (
        <EmptyState>No review history is stored.</EmptyState>
      ) : null}
      {history?.ok ? (
        <ul>
          {history.value.events.map((event) => (
            <li key={event.id}>
              {event.action} · {event.actorPersonId} · {event.occurredAt}
            </li>
          ))}
          {(history.value.changes ?? []).map((change) => (
            <li key={change.id}>
              {change.actorPersonId} changed {change.fields.join(", ")} · {change.occurredAt}
              {change.rationale ? ` · ${change.rationale}` : ""}
            </li>
          ))}
        </ul>
      ) : null}
      <h3>Reviewer comments</h3>
      {history?.ok && history.value.comments.length === 0 ? <EmptyState>No comments are stored.</EmptyState> : null}
      {history?.ok ? (
        <ul>
          {history.value.comments.map((comment) => (
            <li key={comment.id}>
              {comment.authorPersonId}: {comment.body}
            </li>
          ))}
        </ul>
      ) : null}
      {version.status === "profile_draft" || version.status === "profile_in_review" ? (
        <form action={commentOnRoleProfileForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="returnTo" value="skills" />
          <label>
            Comment
            <textarea name="body" required rows={3} />
          </label>
          <PendingButton>Save comment</PendingButton>
        </form>
      ) : null}
      {editable && dirty ? <p>Save changes before submitting for review.</p> : null}
      {editable && !dirty ? (
        <form action={submitRoleProfileForm}>
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <PendingButton className="primary">Submit for review</PendingButton>
        </form>
      ) : null}
      {version.status === "profile_in_review" ? (
        <p>
          <Link className="learn-link" href={`/admin/role-profiles/${profile.id}/review`}>
            Open the approval step
          </Link>
        </p>
      ) : null}
      {editable ? (
        <>
          <h3>Add a canonical skill</h3>
          <form action={createCanonicalSkillForm} className="framework-form">
            <input type="hidden" name="profileId" value={profile.id} />
            <label>
              Name
              <input name="name" required />
            </label>
            <label>
              Category
              <select name="category" defaultValue={category || "technical"}>
                {reviewCategories.map((item) => (
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
          <h3 id="assign-skill">Assign a skill</h3>
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
                {reviewCategories.map((item) => (
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
              <input name="requiredProficiency" inputMode="numeric" required defaultValue="2" />
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
              Assessment criteria
              <textarea name="criteria" rows={2} />
            </label>
            <input type="hidden" name="provenance" value="manually_added" />
            <PendingButton className="primary">Add skill</PendingButton>
          </form>
        </>
      ) : null}
    </section>
  );
}

function SkillEditor({
  skill,
  profile,
  draft,
  editable,
  changed,
  unsaved,
  index,
  total,
  kind,
  onChange,
  onMove,
  onDragStart,
  onDrop,
}: {
  skill: RoleProfileSkill;
  profile: RoleProfile;
  draft: ReviewDraft;
  editable: boolean;
  changed: boolean;
  unsaved: boolean;
  index: number;
  total: number;
  kind: string;
  onChange: (patch: Partial<ReviewDraft>) => void;
  onMove: (direction: "up" | "down") => void;
  onDragStart: () => void;
  onDrop: () => void;
}) {
  const version = profile.version;
  return (
    <article
      className={unsaved ? "skill-card unsaved" : "skill-card"}
      draggable={editable}
      onDragStart={onDragStart}
      onDragOver={(event) => event.preventDefault()}
      onDrop={onDrop}
    >
      <p>
        {skill.position}. {sourceLabel(skill.provenance)}
        {changed ? " · Changed" : ""}
        {unsaved ? " · Unsaved" : ""}
        {kind ? <span className="kind-note">Recorded kind: {kind}</span> : null}
      </p>
      {editable ? (
        <div className="framework-form">
          <label>
            Skill name for {skill.skillName}
            <input value={draft.displayName} onChange={(event) => onChange({ displayName: event.target.value })} required />
          </label>
          <label>
            Description
            <textarea value={draft.definition} onChange={(event) => onChange({ definition: event.target.value })} required rows={2} />
          </label>
          <label>
            Category
            <select value={draft.category} onChange={(event) => onChange({ category: event.target.value })}>
              {reviewCategories.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>
          <label>
            Required proficiency
            <select value={draft.requiredProficiency} onChange={(event) => onChange({ requiredProficiency: event.target.value })}>
              {[0, 1, 2, 3, 4].map((item) => (
                <option key={item} value={String(item)}>
                  {proficiencyLabel(item)}
                </option>
              ))}
            </select>
          </label>
          <label>
            Priority
            <select value={draft.priority} onChange={(event) => onChange({ priority: event.target.value })}>
              {priorities.map((item) => (
                <option key={item} value={item}>
                  {item}
                </option>
              ))}
            </select>
          </label>
          <label>
            Rationale
            <textarea value={draft.rationale} onChange={(event) => onChange({ rationale: event.target.value })} required rows={2} />
          </label>
          <label>
            Assessment criteria
            <textarea value={draft.criteria} onChange={(event) => onChange({ criteria: event.target.value })} rows={2} />
          </label>
          <button type="button" onClick={() => onMove("up")} disabled={index === 0}>
            Move {skillTitle(skill)} up
          </button>
          <button type="button" onClick={() => onMove("down")} disabled={index === total - 1}>
            Move {skillTitle(skill)} down
          </button>
        </div>
      ) : (
        <>
          <h4>{skillTitle(skill)}</h4>
          <p>{skill.definition}</p>
          <p>
            {skill.category} · {proficiencyLabel(skill.requiredProficiency)} · {skill.rationale}
          </p>
        </>
      )}
      {editable ? (
        <form action={removeSkillForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={version.id} />
          <input type="hidden" name="skillAssignmentId" value={skill.id} />
          <input type="hidden" name="expectedRevision" value={version.revision} />
          <label>
            <input type="checkbox" name="confirm" value="yes" required /> Confirm removal of {skillTitle(skill)}
          </label>
          <PendingButton>Remove skill</PendingButton>
        </form>
      ) : null}
    </article>
  );
}

function DuplicateList({ profile, skills }: { profile: RoleProfile; skills: RoleProfileSkill[] }) {
  const pairs = duplicatePairs(skills.map((skill) => ({ id: skill.id, name: skillTitle(skill) })));
  if (pairs.length === 0) {
    return null;
  }
  return (
    <section>
      <h3>Possible duplicates</h3>
      {pairs.map((pair) => (
        <form key={`${pair.keepId}-${pair.removeId}`} action={mergeSkillsForm} className="framework-form">
          <input type="hidden" name="profileId" value={profile.id} />
          <input type="hidden" name="versionId" value={profile.version.id} />
          <input type="hidden" name="expectedRevision" value={profile.version.revision} />
          <input type="hidden" name="keepId" value={pair.keepId} />
          <input type="hidden" name="removeId" value={pair.removeId} />
          <p>Keep the first {pair.name} and remove the later copy.</p>
          <label>
            <input type="checkbox" name="confirm" value="yes" required /> Confirm merge of {pair.name}
          </label>
          <PendingButton>Merge duplicate</PendingButton>
        </form>
      ))}
    </section>
  );
}
