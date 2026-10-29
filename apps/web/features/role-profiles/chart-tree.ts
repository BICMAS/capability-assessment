import type { Position } from "@/lib/org-directory";

export function rootPositions(positions: Position[]): Position[] {
  return positions.filter((position) => position.reportsToPositionId === "" || !positions.some((item) => item.id === position.reportsToPositionId));
}

export function childPositions(positions: Position[], parentId: string): Position[] {
  return positions.filter((position) => position.reportsToPositionId === parentId);
}

export function descendantIds(positions: Position[], positionId: string): Set<string> {
  const ids = new Set<string>();
  for (const child of childPositions(positions, positionId)) {
    ids.add(child.id);
    for (const nested of descendantIds(positions, child.id)) {
      ids.add(nested);
    }
  }
  return ids;
}

export function positionVisible(position: Position, positions: Position[], query: string): boolean {
  const needle = query.trim().toLowerCase();
  if (needle === "") {
    return true;
  }
  if (position.title.toLowerCase().includes(needle)) {
    return true;
  }
  return childPositions(positions, position.id).some((child) => positionVisible(child, positions, query));
}
