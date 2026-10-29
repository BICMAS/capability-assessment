"use client";

import Link from "next/link";
import { useMemo, useState, type ReactNode } from "react";

import type { RoleProfile } from "@/lib/role-profile";

import { filterSkills, kindTotals, listSkills, proficiencyLabel, reportingLabel, sourceLabel, type SkillKind } from "./skills-view";

const tabs: Array<{ kind: SkillKind; label: string }> = [
  { kind: "technical", label: "Technical Skills" },
  { kind: "behavioural", label: "Behavioural Skills" },
  { kind: "knowledge", label: "Knowledge" },
];

const levels = [
  { value: "", label: "All levels" },
  { value: "1", label: "Basic" },
  { value: "2", label: "Intermediate" },
  { value: "3", label: "Advanced" },
  { value: "4", label: "Expert" },
  { value: "0", label: "Not set" },
];

const sources = [
  { value: "", label: "All sources" },
  { value: "ai_suggested", label: "AI suggestion" },
  { value: "hr_adjusted", label: "Adjusted by a person" },
  { value: "manually_added", label: "Added by a person" },
  { value: "imported", label: "Imported" },
];

export function SkillsWorkspace({
  profile,
  jobModel,
  children,
}: {
  profile: RoleProfile;
  jobModel: string;
  children?: ReactNode;
}) {
  const [kind, setKind] = useState<SkillKind>("technical");
  const [search, setSearch] = useState("");
  const [level, setLevel] = useState("");
  const [provenance, setProvenance] = useState("");
  const version = profile.version;
  const listed = useMemo(() => listSkills(version.skills, version.ai.draft), [version.skills, version.ai.draft]);
  const totals = kindTotals(listed);
  const visible = filterSkills(listed, { kind, search, level, provenance });
  const mock = jobModel === "mock-role-skills" || version.ai.model === "mock-role-skills";
  const suggested = listed.some((skill) => skill.provenance === "ai_suggested");
  const active = tabs.find((tab) => tab.kind === kind) ?? tabs[0];
  return (
    <section className="directory">
      <h2>{profile.roleName}</h2>
      <p>Status {version.status}. Recommendations on this page remain suggestions until a person reviews them.</p>
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
      {mock ? <p className="mock-banner">Development mock. These suggestions are not model output.</p> : null}
      {suggested ? <p className="skill-suggestion">These recommendations are suggestions pending human review.</p> : null}
      {children}
      <div className="skill-tabs" role="tablist" aria-label="Skill categories">
        {tabs.map((tab) => (
          <button key={tab.kind} type="button" role="tab" aria-selected={kind === tab.kind} onClick={() => setKind(tab.kind)}>
            {tab.label} ({totals[tab.kind]})
          </button>
        ))}
      </div>
      <div className="skill-filters">
        <label>
          Search
          <input value={search} onChange={(event) => setSearch(event.target.value)} type="search" />
        </label>
        <label>
          Required level
          <select value={level} onChange={(event) => setLevel(event.target.value)}>
            {levels.map((item) => (
              <option key={item.value || "all"} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
        <label>
          Source
          <select value={provenance} onChange={(event) => setProvenance(event.target.value)}>
            {sources.map((item) => (
              <option key={item.value || "all"} value={item.value}>
                {item.label}
              </option>
            ))}
          </select>
        </label>
      </div>
      <p>
        {active.label}: {visible.length} shown, {totals[kind]} in this category.
      </p>
      {visible.length === 0 ? <p>No skills match this view.</p> : null}
      {visible.length > 0 ? (
        <div className="skill-scroll">
          <table className="skills-table">
            <thead>
              <tr>
                <th scope="col">Number</th>
                <th scope="col">Skill</th>
                <th scope="col">Category</th>
                <th scope="col">Required level</th>
                <th scope="col">Source</th>
                <th scope="col">Actions</th>
              </tr>
            </thead>
            <tbody>
              {visible.map((skill) => (
                <tr key={skill.id}>
                  <td>{skill.position}</td>
                  <td>{skill.skillName}</td>
                  <td>{skill.category}</td>
                  <td>
                    {proficiencyLabel(skill.requiredProficiency)} ({skill.requiredProficiency})
                  </td>
                  <td>{sourceLabel(skill.provenance)}</td>
                  <td>
                    <details>
                      <summary>Inspect {skill.skillName}</summary>
                      <p>{skill.definition}</p>
                      <p>Rationale: {skill.rationale}</p>
                      <p>Assessment criteria: {skill.criteria || "Not recorded"}</p>
                      <p>Created: {skill.createdAt || "Not recorded"}</p>
                      <p>Modified: {skill.updatedAt || "Not recorded"}</p>
                    </details>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}
      <p className="chart-modes">
        <Link className="learn-link" href={`/admin/role-profiles/${profile.id}#assign-skill`}>
          Add a skill
        </Link>
        <Link className="learn-link" href={`/admin/role-profiles/${profile.id}`}>
          Review and customise
        </Link>
      </p>
    </section>
  );
}
