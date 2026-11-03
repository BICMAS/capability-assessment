"use client";

import { Expand, Minimize2, Search, ZoomIn, ZoomOut } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useMemo, useRef, useState } from "react";

import type { Department, DirectoryPerson, Organogram, Position } from "@/lib/org-directory";

import { addPositionAction, removePositionAction, savePositionAction } from "./chart-actions";
import { childPositions, descendantIds, positionVisible, rootPositions } from "./chart-tree";

const minZoom = 0.7;
const maxZoom = 1.4;
const zoomStep = 0.1;

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
  const canvasRef = useRef<HTMLDivElement>(null);
  const [query, setQuery] = useState("");
  const [departmentFilter, setDepartmentFilter] = useState("");
  const [message, setMessage] = useState("");
  const [notice, setNotice] = useState("");
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({});
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [pendingDelete, setPendingDelete] = useState<Position | null>(null);
  const [zoom, setZoom] = useState(1);

  const filtered = useMemo(() => {
    return positions.filter((position) => {
      if (departmentFilter && position.departmentId !== departmentFilter) {
        return false;
      }
      return positionVisible(position, positions, query);
    });
  }, [departmentFilter, positions, query]);

  const roots = useMemo(
    () => rootPositions(positions).filter((position) => positionVisible(position, positions, query) && (!departmentFilter || matchesDepartmentBranch(position, positions, departmentFilter))),
    [departmentFilter, positions, query],
  );

  const selected = selectedId ? (positions.find((position) => position.id === selectedId) ?? null) : null;
  const assignedIds = new Set(positions.map((position) => position.personId).filter((id) => id !== ""));
  const availablePeople = people.filter((person) => !assignedIds.has(person.id) || person.id === selected?.personId);

  useEffect(() => {
    if (selectedId && !positions.some((position) => position.id === selectedId)) {
      setSelectedId(null);
    }
  }, [positions, selectedId]);

  async function refresh(error: string, success = "") {
    setMessage(error);
    setNotice(error === "" ? success : "");
    if (error === "") {
      router.refresh();
    }
  }

  function fitToScreen() {
    setZoom(1);
    canvasRef.current?.scrollTo({ top: 0, left: 0 });
  }

  function expandAll() {
    setCollapsed({});
  }

  function collapseAll() {
    const next: Record<string, boolean> = {};
    for (const position of positions) {
      if (childPositions(positions, position.id).length > 0) {
        next[position.id] = true;
      }
    }
    setCollapsed(next);
  }

  return (
    <div className="builder-shell">
      <header className="builder-toolbar">
        <div>
          <h4>Build online</h4>
          <p>
            <strong>{chart.name}</strong>{" "}
            <span className={chart.status === "published" ? "status-pill" : "role-status-draft"}>
              {chart.status === "published" ? "Published" : "Draft"}
            </span>
          </p>
        </div>
        <p className="portal-note">Edits save through the existing position APIs. Publish from Chart actions when the structure is ready.</p>
      </header>

      {message ? (
        <p role="alert" className="status-message is-error">
          {message}
        </p>
      ) : null}
      {notice ? (
        <p role="status" className="status-message is-success">
          {notice}
        </p>
      ) : null}

      <div className="builder-layout">
        <aside className="builder-panel builder-panel-left" aria-label="Position catalogue">
          <label className="builder-search">
            <Search size={16} aria-hidden="true" />
            <span className="app-clip">Search positions</span>
            <input value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search positions" />
          </label>

          <div className="builder-block">
            <h5>Departments</h5>
            <div className="builder-chips" role="group" aria-label="Filter by department">
              <button type="button" className={departmentFilter === "" ? "is-active" : undefined} onClick={() => setDepartmentFilter("")}>
                All
              </button>
              {departments.map((department) => (
                <button
                  key={department.id}
                  type="button"
                  className={departmentFilter === department.id ? "is-active" : undefined}
                  onClick={() => setDepartmentFilter(department.id)}
                >
                  {department.name}
                </button>
              ))}
            </div>
          </div>

          <div className="builder-block">
            <h5>Existing positions</h5>
            {filtered.length === 0 ? <p className="empty-state">No positions match this search.</p> : null}
            <ul className="builder-list">
              {filtered.map((position) => {
                const department = departments.find((item) => item.id === position.departmentId);
                const person = people.find((item) => item.id === position.personId);
                return (
                  <li key={position.id}>
                    <button
                      type="button"
                      className={selectedId === position.id ? "is-selected" : undefined}
                      onClick={() => setSelectedId(position.id)}
                    >
                      <strong>{position.title}</strong>
                      <span>{department?.name || "No department"}</span>
                      <span>{person?.name || "Vacant"}</span>
                    </button>
                  </li>
                );
              })}
            </ul>
          </div>

          <div className="builder-block">
            <h5>Available employees</h5>
            {availablePeople.length === 0 ? <p className="empty-state">Every listed person is already assigned, or none are recorded.</p> : null}
            <ul className="builder-people">
              {availablePeople.map((person) => (
                <li key={person.id}>
                  <strong>{person.name}</strong>
                  <span>{person.departmentName || "No department"}</span>
                </li>
              ))}
            </ul>
          </div>

          <div className="builder-block">
            <h5>Add new position</h5>
            <AddPosition
              chartId={chart.id}
              positions={positions}
              departments={departments}
              people={people}
              onDone={(error) => refresh(error, error === "" ? "Position added." : "")}
            />
          </div>
        </aside>

        <section className="builder-canvas-wrap" aria-label="Organisation chart">
          <div className="builder-canvas-controls">
            <button type="button" onClick={() => setZoom((value) => Math.min(maxZoom, Number((value + zoomStep).toFixed(2))))} aria-label="Zoom in">
              <ZoomIn size={16} aria-hidden="true" />
              Zoom in
            </button>
            <button type="button" onClick={() => setZoom((value) => Math.max(minZoom, Number((value - zoomStep).toFixed(2))))} aria-label="Zoom out">
              <ZoomOut size={16} aria-hidden="true" />
              Zoom out
            </button>
            <button type="button" onClick={fitToScreen} aria-label="Fit to screen">
              <Minimize2 size={16} aria-hidden="true" />
              Fit to screen
            </button>
            <button type="button" onClick={expandAll} aria-label="Expand branches">
              <Expand size={16} aria-hidden="true" />
              Expand
            </button>
            <button type="button" onClick={collapseAll}>
              Collapse
            </button>
            <span className="builder-zoom-label">{Math.round(zoom * 100)}%</span>
          </div>
          <div className="builder-canvas" ref={canvasRef}>
            <div className="builder-canvas-scale" style={{ transform: `scale(${zoom})` }}>
              {positions.length === 0 ? <p className="empty-state">No positions are recorded on this chart.</p> : null}
              {positions.length > 0 && roots.length === 0 ? <p className="empty-state">No positions match this search.</p> : null}
              <ul className="builder-tree">
                {roots.map((position) => (
                  <ChartBranch
                    key={position.id}
                    chartId={chart.id}
                    position={position}
                    positions={positions}
                    departments={departments}
                    people={people}
                    query={query}
                    departmentFilter={departmentFilter}
                    collapsed={collapsed}
                    selectedId={selectedId}
                    onSelect={setSelectedId}
                    onToggle={(id) => setCollapsed((current) => ({ ...current, [id]: !current[id] }))}
                    onDelete={setPendingDelete}
                  />
                ))}
              </ul>
            </div>
          </div>
        </section>

        <aside className="builder-panel builder-panel-right" aria-label="Position details">
          {selected ? (
            <PositionInspector
              chartId={chart.id}
              position={selected}
              positions={positions}
              departments={departments}
              people={people}
              onDelete={() => setPendingDelete(selected)}
              onDone={(error) => refresh(error, error === "" ? "Position saved." : "")}
            />
          ) : (
            <div className="builder-empty-selection">
              <h5>Position details</h5>
              <p className="portal-note">Select a position in the chart or list to edit its title, department, reporting line, and assignee.</p>
            </div>
          )}
        </aside>
      </div>

      {pendingDelete ? (
        <div className="chart-confirm" role="dialog" aria-modal="true" aria-labelledby="remove-position-title">
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
              void removePositionAction(chart.id, target.id).then((result) => {
                if (result.ok && selectedId === target.id) {
                  setSelectedId(null);
                }
                void refresh(result.ok ? "" : result.message, result.ok ? "Position removed." : "");
              });
            }}
          >
            Remove position
          </button>
        </div>
      ) : null}
    </div>
  );
}

function ChartBranch({
  chartId,
  position,
  positions,
  departments,
  people,
  query,
  departmentFilter,
  collapsed,
  selectedId,
  onSelect,
  onToggle,
  onDelete,
}: {
  chartId: string;
  position: Position;
  positions: Position[];
  departments: Department[];
  people: DirectoryPerson[];
  query: string;
  departmentFilter: string;
  collapsed: Record<string, boolean>;
  selectedId: string | null;
  onSelect: (id: string) => void;
  onToggle: (id: string) => void;
  onDelete: (position: Position) => void;
}) {
  const children = childPositions(positions, position.id).filter(
    (child) => positionVisible(child, positions, query) && (!departmentFilter || matchesDepartmentBranch(child, positions, departmentFilter)),
  );
  const department = departments.find((item) => item.id === position.departmentId);
  const person = people.find((item) => item.id === position.personId);
  const selected = selectedId === position.id;
  return (
    <li>
      <div className={selected ? "builder-node is-selected" : "builder-node"}>
        <button type="button" className="builder-node-main" onClick={() => onSelect(position.id)}>
          <strong>{position.title}</strong>
          <span>{department?.name || "No department"}</span>
          <span>{person?.name || "Vacant"}</span>
        </button>
        <div className="builder-node-actions">
          {children.length > 0 ? (
            <button type="button" aria-expanded={!collapsed[position.id]} onClick={() => onToggle(position.id)}>
              {collapsed[position.id] ? "Expand" : "Collapse"}
            </button>
          ) : null}
          <Link
            href={`/admin/role-profiles/new?title=${encodeURIComponent(position.title)}&position=${encodeURIComponent(position.id)}&organogram=${encodeURIComponent(chartId)}`}
          >
            Use for role profile
          </Link>
          <button type="button" onClick={() => onDelete(position)}>
            Remove
          </button>
        </div>
      </div>
      {children.length > 0 && !collapsed[position.id] ? (
        <ul>
          {children.map((child) => (
            <ChartBranch
              key={child.id}
              chartId={chartId}
              position={child}
              positions={positions}
              departments={departments}
              people={people}
              query={query}
              departmentFilter={departmentFilter}
              collapsed={collapsed}
              selectedId={selectedId}
              onSelect={onSelect}
              onToggle={onToggle}
              onDelete={onDelete}
            />
          ))}
        </ul>
      ) : null}
    </li>
  );
}

function PositionInspector({
  chartId,
  position,
  positions,
  departments,
  people,
  onDelete,
  onDone,
}: {
  chartId: string;
  position: Position;
  positions: Position[];
  departments: Department[];
  people: DirectoryPerson[];
  onDelete: () => void;
  onDone: (error: string) => void;
}) {
  const [title, setTitle] = useState(position.title);
  const [reportsTo, setReportsTo] = useState(position.reportsToPositionId);
  const [departmentId, setDepartmentId] = useState(position.departmentId);
  const [personId, setPersonId] = useState(position.personId);
  const [pending, setPending] = useState(false);
  const excluded = descendantIds(positions, position.id);
  excluded.add(position.id);

  useEffect(() => {
    setTitle(position.title);
    setReportsTo(position.reportsToPositionId);
    setDepartmentId(position.departmentId);
    setPersonId(position.personId);
  }, [position]);

  return (
    <form
      className="framework-form builder-inspector"
      onSubmit={(event) => {
        event.preventDefault();
        if (title.trim() === "") {
          onDone("Enter a position title.");
          return;
        }
        setPending(true);
        void savePositionAction(chartId, position.id, {
          title,
          departmentId,
          personId,
          reportsToPositionId: reportsTo,
        }).then((result) => {
          setPending(false);
          onDone(result.ok ? "" : result.message);
        });
      }}
    >
      <h5>Position details</h5>
      <label>
        Position title
        <input value={title} onChange={(event) => setTitle(event.target.value)} required />
      </label>
      <ReportsField positions={positions} excluded={excluded} value={reportsTo} onChange={setReportsTo} />
      <DepartmentField departments={departments} value={departmentId} onChange={setDepartmentId} />
      <PersonField people={people} value={personId} onChange={setPersonId} />
      <div className="builder-inspector-actions">
        <button type="submit" className="primary" disabled={pending}>
          {pending ? "Saving…" : "Save changes"}
        </button>
        <button type="button" onClick={onDelete}>
          Remove
        </button>
        <Link
          href={`/admin/role-profiles/new?title=${encodeURIComponent(position.title)}&position=${encodeURIComponent(position.id)}&organogram=${encodeURIComponent(chartId)}`}
        >
          Use for role profile
        </Link>
      </div>
    </form>
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
  const [pending, setPending] = useState(false);
  return (
    <form
      className="framework-form"
      onSubmit={(event) => {
        event.preventDefault();
        if (title.trim() === "") {
          onDone("Enter a position title.");
          return;
        }
        setPending(true);
        void addPositionAction(chartId, { title, departmentId, personId, reportsToPositionId: reportsTo }).then((result) => {
          setPending(false);
          if (result.ok) {
            setTitle("");
            setReportsTo("");
            setDepartmentId("");
            setPersonId("");
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
      <button type="submit" className="primary" disabled={pending}>
        {pending ? "Saving…" : "Add position"}
      </button>
    </form>
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

function matchesDepartmentBranch(position: Position, positions: Position[], departmentId: string): boolean {
  if (position.departmentId === departmentId) {
    return true;
  }
  return childPositions(positions, position.id).some((child) => matchesDepartmentBranch(child, positions, departmentId));
}
