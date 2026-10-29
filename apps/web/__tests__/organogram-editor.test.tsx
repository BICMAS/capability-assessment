import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
}));

vi.mock("@/features/role-profiles/chart-actions", () => ({
  addPositionAction: vi.fn(),
  savePositionAction: vi.fn(),
  removePositionAction: vi.fn(),
  publishOrganogramAction: vi.fn(),
}));

import { removePositionAction } from "@/features/role-profiles/chart-actions";
import { OrganogramEditor } from "@/features/role-profiles/organogram-editor";
import type { Position } from "@/lib/org-directory";

const positions: Position[] = [
  {
    id: "director",
    organisationId: "org",
    organogramId: "chart",
    departmentId: "",
    title: "Managing Director",
    personId: "",
    reportsToPositionId: "",
    createdAt: "2026-10-29T00:00:00Z",
  },
  {
    id: "manager",
    organisationId: "org",
    organogramId: "chart",
    departmentId: "sales",
    title: "Sales Manager",
    personId: "",
    reportsToPositionId: "director",
    createdAt: "2026-10-29T00:00:00Z",
  },
];

afterEach(() => {
  cleanup();
});

describe("organogram editor", () => {
  it("shows the saved hierarchy and asks before removing a position", async () => {
    render(
      <OrganogramEditor
        chart={{ id: "chart", organisationId: "org", name: "Sales chart", status: "draft", createdByPersonId: "ada", createdAt: "2026-10-29T00:00:00Z" }}
        positions={positions}
        departments={[{ id: "sales", organisationId: "org", name: "Sales" }]}
        people={[]}
      />,
    );

    expect(screen.getAllByText("Managing Director").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Sales Manager").length).toBeGreaterThan(0);
    expect(screen.getAllByRole("link", { name: "Use for role profile" })[0].getAttribute("href")).toContain("title=Managing%20Director");
    await userEvent.click(screen.getAllByRole("button", { name: "Remove" })[0]);
    expect(screen.getByRole("dialog", { name: "Remove Managing Director?" })).toBeTruthy();
    expect(removePositionAction).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(removePositionAction).not.toHaveBeenCalled();
  });
});
