"use client";

import { useMemo, useState } from "react";

import type { AssignmentResult } from "@/lib/generated/assessment-plan/plans";

import { confirmAssignment } from "./assessment-actions";

export type AssignmentPerson = {
  id: string;
  name: string;
  departmentName: string;
  titles: string[];
};

export type AssignmentVersion = {
  id: string;
  frameworkName: string;
  versionNumber: number;
  roles: { id: string; name: string }[];
  capabilities: { id: string; name: string }[];
  requirements: { id: string; roleId: string; capabilityId: string; requiredLevel: number }[];
};

export function AssignmentStep({
  people,
  versions,
  results,
  error,
}: {
  people: AssignmentPerson[];
  versions: AssignmentVersion[];
  results: AssignmentResult[];
  error: string;
}) {
  const [query, setQuery] = useState("");
  const [selected, setSelected] = useState<string[]>([]);
  const [versionId, setVersionId] = useState(versions[0]?.id ?? "");
  const [roleId, setRoleId] = useState("");
  const version = versions.find((item) => item.id === versionId) ?? null;
  const role = version?.roles.find((item) => item.id === roleId) ?? null;
  const requirements = (version?.requirements ?? []).filter((requirement) => requirement.roleId === role?.id);
  const names = new Map(people.map((person) => [person.id, person.name]));
  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) {
      return people;
    }
    return people.filter((person) => person.name.toLowerCase().includes(needle) || person.departmentName.toLowerCase().includes(needle));
  }, [people, query]);
  const mismatched = role
    ? people.filter((person) => selected.includes(person.id) && !titlesMatch(person.titles, role.name))
    : [];

  function toggle(personId: string) {
    setSelected((current) => (current.includes(personId) ? current.filter((id) => id !== personId) : [...current, personId]));
  }

  return (
    <div className="studio-tool">
      <form action={confirmAssignment} className="framework-form">
        <label>
          Search people
          <input value={query} onChange={(event) => setQuery(event.target.value)} type="search" placeholder="Name or department" />
        </label>
        <ul className="person-choices">
          {visible.map((person) => (
            <li key={person.id}>
              <label>
                <input
                  type="checkbox"
                  name="personId"
                  value={person.id}
                  checked={selected.includes(person.id)}
                  onChange={() => toggle(person.id)}
                />
                <span>
                  {person.name}
                  <span className="person-title">{person.titles.length > 0 ? person.titles.join(", ") : "No directory role"}</span>
                </span>
              </label>
            </li>
          ))}
        </ul>
        <label>
          Published framework
          <select
            name="frameworkVersionId"
            value={versionId}
            onChange={(event) => {
              setVersionId(event.target.value);
              setRoleId("");
            }}
            required
          >
            {versions.length === 0 ? <option value="">No published version</option> : null}
            {versions.map((item) => (
              <option key={item.id} value={item.id}>
                {item.frameworkName} v{item.versionNumber}
              </option>
            ))}
          </select>
        </label>
        <label>
          Role
          <select name="roleId" value={roleId} onChange={(event) => setRoleId(event.target.value)} required>
            <option value="">Choose a role</option>
            {(version?.roles ?? []).map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
              </option>
            ))}
          </select>
        </label>
        {role ? (
          <div className="role-brief">
            <h3>{role.name}</h3>
            <ul className="role-levels">
              {requirements.map((requirement) => {
                const capability = version?.capabilities.find((item) => item.id === requirement.capabilityId);
                return (
                  <li key={requirement.id}>
                    <span>{capability?.name ?? "Capability"}</span>
                    <span className="role-level">{requirement.requiredLevel}</span>
                  </li>
                );
              })}
            </ul>
          </div>
        ) : null}
        {mismatched.length > 0 ? (
          <label>
            Exception reason
            <textarea name="exceptionReason" required minLength={1} maxLength={280} placeholder="Why this title can take the role" />
            <span className="person-title">
              {mismatched.map((person) => person.name).join(", ")} {mismatched.length === 1 ? "does" : "do"} not match {role?.name}.
            </span>
          </label>
        ) : null}
        {mismatched.map((person) => (
          <input key={person.id} type="hidden" name="mismatchPersonId" value={person.id} />
        ))}
        {error ? <p className="login-error">{error}</p> : null}
        <button className="primary" type="submit" disabled={selected.length === 0 || !role}>
          Confirm assignment
        </button>
      </form>
      {results.length > 0 ? (
        <ul className="assignment-results">
          {results.map((result) => (
            <li key={result.personId}>
              <span>{names.get(result.personId) ?? result.personId}</span>
              <span className="studio-lock">{result.outcome}</span>
              <span>{result.reason}</span>
              {result.assignmentId ? <a href={`/admin?assignment=${encodeURIComponent(result.assignmentId)}#plans`}>Open plan</a> : null}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}

function titlesMatch(titles: string[], roleName: string): boolean {
  return roleName.length > 0 && titles.some((title) => title.trim() === roleName);
}
