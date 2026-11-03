import type { Department, Position } from "@/lib/org-directory";

import { childPositions, rootPositions } from "./chart-tree";

export function ChartPreview({ positions, departments }: { positions: Position[]; departments: Department[] }) {
  if (positions.length === 0) {
    return <p className="empty-state">No positions are recorded on this chart yet.</p>;
  }
  const roots = rootPositions(positions);
  return (
    <ul className="organo-preview-tree">
      {roots.map((position) => (
        <PreviewNode key={position.id} position={position} positions={positions} departments={departments} />
      ))}
    </ul>
  );
}

function PreviewNode({
  position,
  positions,
  departments,
}: {
  position: Position;
  positions: Position[];
  departments: Department[];
}) {
  const children = childPositions(positions, position.id);
  const department = departments.find((item) => item.id === position.departmentId);
  return (
    <li>
      <div className="organo-preview-node">
        <strong>{position.title}</strong>
        <span>{department ? department.name : "No department"}</span>
      </div>
      {children.length > 0 ? (
        <ul>
          {children.map((child) => (
            <PreviewNode key={child.id} position={child} positions={positions} departments={departments} />
          ))}
        </ul>
      ) : null}
    </li>
  );
}
