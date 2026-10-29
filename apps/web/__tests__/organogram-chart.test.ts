import { describe, expect, it } from "vitest";

import { descendantIds, rootPositions } from "@/features/role-profiles/chart-tree";
import { organogramUploadIssue } from "@/features/role-profiles/upload-rules";
import type { Position } from "@/lib/org-directory";

function position(id: string, title: string, reportsTo: string): Position {
  return {
    id,
    organisationId: "org",
    organogramId: "chart",
    departmentId: "",
    title,
    personId: "",
    reportsToPositionId: reportsTo,
    createdAt: "2026-10-29T00:00:00Z",
  };
}

describe("organogram chart", () => {
  it("keeps roots at the top and excludes a position from reporting to its own reports", () => {
    const positions = [position("director", "Managing Director", ""), position("manager", "Sales Manager", "director"), position("lead", "Team Lead", "manager")];
    expect(rootPositions(positions).map((item) => item.id)).toEqual(["director"]);
    const blocked = descendantIds(positions, "manager");
    blocked.add("manager");
    expect(blocked.has("lead")).toBe(true);
    expect(blocked.has("director")).toBe(false);
  });

  it("rejects an unsupported or oversized file before upload", () => {
    expect(organogramUploadIssue({ name: "notes.txt", size: 12 })).toBe("Upload a PNG, JPG, PDF, XLS, or XLSX file.");
    expect(organogramUploadIssue({ name: "chart.png", size: 5 * 1024 * 1024 + 1 })).toBe("The file must be 5 MB or smaller.");
    expect(organogramUploadIssue({ name: "chart.xlsx", size: 120 })).toBeNull();
  });
});
