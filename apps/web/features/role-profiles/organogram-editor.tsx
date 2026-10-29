"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";

import type { Department, DirectoryPerson, Organogram, Position } from "@/lib/org-directory";

import { addPositionAction, publishOrganogramAction, removePositionAction, savePositionAction } from "./chart-actions";
import { childPositions, descendantIds, positionVisible, rootPositions } from "./chart-tree";

export function OrganogramEditor({
  chart,
  positions,
  departments,
  people,
}: {
  chart: Organogram;
  positions: Position[];
  departments: Department[];
  people: DirectoryPerson[];
}) {
  const router = useRouter();
  const [query, setQuery] = useState("");
  const [message, setMessage] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [pendingDelete, setPendingDelete] = useState<Position | null>(null);
  const roots = useMemo(() => rootPositions(positions).filter((position) => positionVisible(position, positions, query)), [positions, query]);

  async function refresh(error: string) {
    setMessage(error);
    if (error === "") {
      router.refresh();
    }
  }

  return (
    <div className="chart-editor">
      <div className="chart-editor-bar">
        <p>
          <strong>{chart.name}</strong> <span className="status-pill">{chart.status === "published" ? "Published" : "Draft"}</span>
        </p>
        {chart.status === "draft" ? (
          <button
            type="button"
            className="primary"
            onClick={() => {
              void publishOrganogramAction(chart.id).then((result) => refresh(result.ok ? "" : result.message));
            }}
          >
            Publish chart
          </button>
        ) : null}
      </div>
      <label>
        Search positions
        <input value={query} onChange={(event) => setQuery(event.target.value)} />
      </label>
      {message ? (
        <p role="alert" className="status-message is-error">
          {message}
        </p>
      ) : null}
      <AddPosition chartId={chart.id} positions={positions} departments={departments} people={people} onDone={refresh} />
      {positions.length === 0 ? <p className="empty-state">No positions are recorded on this chart.</p> : null}
      {positions.length > 0 && roots.length === 0 ? <p className="empty-state">No positions match this search.</p> : null}
      <ul className="chart-tree">
        {roots.map((position) => (
          <ChartNode
            key={position.id}
            chartId={chart.id}
            position={position}
            positions={positions}
            departments={departments}
            people={people}
            query={query}
            collapsed={collapsed}
            onToggle={(id) => setCollapsed((current) => ({ ...current, [id]: !current[id] }))}
            onDelete={setPendingDelete}
            onDone={refresh}
          />
        ))}
      </ul>
      {pendingDelete ? (
        <div className="chart-confirm" role="dialog" aria-labelledby="remove-position-title">
          <h3 id="remove-position-title">Remove {pendingDelete.title}?</h3>
          <p>Positions that still report to it must be moved first.</p>
          <button type="button" onClick={() => setPendingDelete(null)}>
            Cancel
          </button>
          <button
            type="button"
            className="primary"
            onClick={() => {
              const target = pendingDelete;
              setPendingDelete(null);
              void removePositionAction(chart.id, target.id).then((result) => refresh(result.ok ? "" : result.message));
            }}
          >
            Remove position
          </button>
        </div>
      ) : null}
    </div>
  );
}

function AddPosition({
  chartId,
  positions,
  departments,
  people,
  onDone,
}: {
  chartId: string;
  positions: Position[];
  departments: Department[];
  people: DirectoryPerson[];
  onDone: (error: string) => void;
}) {
  const [title, setTitle] = useState("");
  const [reportsTo, setReportsTo] = useState("");
  const [departmentId, setDepartmentId] = useState("");
  const [personId, setPersonId] = useState("");
  return (
    <form
      className="framework-form"
      onSubmit={(event) => {
        event.preventDefault();
        void addPositionAction(chartId, { title, departmentId, personId, reportsToPositionId: reportsTo }).then((result) => {
          if (result.ok) {
            setTitle("");
          }
          onDone(result.ok ? "" : result.message);
        });
      }}
    >
      <label>
        New position title
        <input value={title} onChange={(event) => setTitle(event.target.value)} required />
      </label>
      <ReportsField positions={positions} excluded={new Set()} value={reportsTo} onChange={setReportsTo} />
      <DepartmentField departments={departments} value={departmentId} onChange={setDepartmentId} />
      <PersonField people={people} value={personId} onChange={setPersonId} />
      <button type="submit" className="primary">
        Add position
      </button>
    </form>
  );
}

function ChartNode({
  chartId,
  position,
  positions,
  departments,
  people,
  query,
  collapsed,
  onToggle,
  onDelete,
  onDone,
}: {
  chartId: string;
  position: Position;
  positions: Position[];
  departments: Department[];
  people: DirectoryPerson[];
  query: string;
  collapsed: Record<string, boolean>;
  onToggle: (id: string) => void;
  onDelete: (position: Position) => void;
  onDone: (error: string) => void;
}) {
  const children = childPositions(positions, position.id).filter((child) => positionVisible(child, positions, query));
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState(position.title);
  const [reportsTo, setReportsTo] = useState(position.reportsToPositionId);
  const [departmentId, setDepartmentId] = useState(position.departmentId);
  const [personId, setPersonId] = useState(position.personId);
  const excluded = descendantIds(positions, position.id);
  excluded.add(position.id);
  const department = departments.find((item) => item.id === position.departmentId);
  return (
    <li>
      <div className="chart-node">
        {children.length > 0 ? (
          <button type="button" aria-expanded={!collapsed[position.id]} onClick={() => onToggle(position.id)}>
            {collapsed[position.id] ? "Expand" : "Collapse"}
          </button>
        ) : null}
        <strong>{position.title}</strong>
        <span>{department ? department.name : "No department"}</span>
        <Link href={`/admin/role-profiles/new?title=${encodeURIComponent(position.title)}&position=${encodeURIComponent(position.id)}&organogram=${encodeURIComponent(chartId)}`}>Use for role profile</Link>
        <button type="button" onClick={() => setEditing((current) => !current)}>
          Edit
        </button>
        <button type="button" onClick={() => onDelete(position)}>
          Remove
        </button>
      </div>
      {editing ? (
        <form
          className="framework-form"
          onSubmit={(event) => {
            event.preventDefault();
            void savePositionAction(chartId, position.id, { title, departmentId, personId, reportsToPositionId: reportsTo }).then((result) => {
              if (result.ok) {
                setEditing(false);
              }
              onDone(result.ok ? "" : result.message);
            });
          }}
        >
          <label>
            Job title
            <input value={title} onChange={(event) => setTitle(event.target.value)} required />
          </label>
          <ReportsField positions={positions} excluded={excluded} value={reportsTo} onChange={setReportsTo} />
          <DepartmentField departments={departments} value={departmentId} onChange={setDepartmentId} />
          <PersonField people={people} value={personId} onChange={setPersonId} />
          <button type="submit">Save position</button>
        </form>
      ) : null}
      {children.length > 0 && !collapsed[position.id] ? (
        <ul>
          {children.map((child) => (
            <ChartNode
              key={child.id}
              chartId={chartId}
              position={child}
              positions={positions}
              departments={departments}
              people={people}
              query={query}
              collapsed={collapsed}
              onToggle={onToggle}
              onDelete={onDelete}
              onDone={onDone}
            />
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function ReportsField({
  positions,
  excluded,
  value,
  onChange,
}: {
  positions: Position[];
  excluded: Set<string>;
  value: string;
  onChange: (value: string) => void;
}) {
  return (
    <label>
      Reports to
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">No manager on this chart</option>
        {positions
          .filter((position) => !excluded.has(position.id))
          .map((position) => (
            <option key={position.id} value={position.id}>
              {position.title}
            </option>
          ))}
      </select>
    </label>
  );
}

function DepartmentField({ departments, value, onChange }: { departments: Department[]; value: string; onChange: (value: string) => void }) {
  return (
    <label>
      Department
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">No department</option>
        {departments.map((department) => (
          <option key={department.id} value={department.id}>
            {department.name}
          </option>
        ))}
      </select>
    </label>
  );
}

function PersonField({ people, value, onChange }: { people: DirectoryPerson[]; value: string; onChange: (value: string) => void }) {
  return (
    <label>
      Person in this position
      <select value={value} onChange={(event) => onChange(event.target.value)}>
        <option value="">Vacant</option>
        {people.map((person) => (
          <option key={person.id} value={person.id}>
            {person.name}
          </option>
        ))}
      </select>
    </label>
  );
}
