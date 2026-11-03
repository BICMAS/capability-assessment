import { cleanup, fireEvent, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
}));

vi.mock("@/features/role-profiles/chart-actions", () => ({
  addPositionAction: vi.fn(async () => ({ ok: true })),
  savePositionAction: vi.fn(async () => ({ ok: true })),
  removePositionAction: vi.fn(),
  publishOrganogramAction: vi.fn(),
}));

import { removePositionAction, savePositionAction } from "@/features/role-profiles/chart-actions";
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
    personId: "eli",
    reportsToPositionId: "director",
    createdAt: "2026-10-29T00:00:00Z",
  },
];

afterEach(() => {
  cleanup();
  vi.mocked(removePositionAction).mockReset();
  vi.mocked(savePositionAction).mockReset();
});

describe("organogram editor", () => {
  it("shows the hierarchy workspace and asks before removing a position", async () => {
    const user = userEvent.setup();
    render(
      <OrganogramEditor
        chart={{ id: "chart", organisationId: "org", name: "Sales chart", status: "draft", createdByPersonId: "ada", createdAt: "2026-10-29T00:00:00Z" }}
        positions={positions}
        departments={[{ id: "sales", organisationId: "org", name: "Sales" }]}
        people={[
          { id: "eli", organisationId: "org", name: "Eli Employee", departmentName: "Sales" },
          { id: "ada", organisationId: "org", name: "Ada Admin", departmentName: "" },
        ]}
      />,
    );

    expect(screen.getByRole("heading", { name: "Build online" })).toBeTruthy();
    expect(screen.getByLabelText("Organisation chart")).toBeTruthy();
    expect(screen.getByLabelText("Position catalogue")).toBeTruthy();
    expect(screen.getByLabelText("Position details")).toBeTruthy();
    expect(screen.getAllByText("Managing Director").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Sales Manager").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Ada Admin").length).toBeGreaterThan(0);
    expect(screen.getByRole("button", { name: "Zoom in" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Fit to screen" })).toBeTruthy();
    expect(screen.getAllByRole("link", { name: "Use for role profile" })[0].getAttribute("href")).toContain("title=Managing%20Director");

    const chart = screen.getByLabelText("Organisation chart");
    await user.click(within(chart).getAllByRole("button", { name: "Remove" })[0]);
    expect(screen.getByRole("dialog", { name: "Remove Managing Director?" })).toBeTruthy();
    expect(removePositionAction).not.toHaveBeenCalled();
    await user.click(screen.getByRole("button", { name: "Cancel" }));
    expect(removePositionAction).not.toHaveBeenCalled();
  });

  it("opens the inspector when a chart node is selected and saves through the existing action", async () => {
    const user = userEvent.setup();
    render(
      <OrganogramEditor
        chart={{ id: "chart", organisationId: "org", name: "Sales chart", status: "draft", createdByPersonId: "ada", createdAt: "2026-10-29T00:00:00Z" }}
        positions={positions}
        departments={[{ id: "sales", organisationId: "org", name: "Sales" }]}
        people={[{ id: "eli", organisationId: "org", name: "Eli Employee", departmentName: "Sales" }]}
      />,
    );

    const catalogue = screen.getByLabelText("Position catalogue");
    await user.click(within(catalogue).getByRole("button", { name: /Sales Manager/ }));
    const details = screen.getByLabelText("Position details");
    const title = within(details).getByLabelText("Position title");
    expect(title).toHaveValue("Sales Manager");
    fireEvent.change(title, { target: { value: "Area Sales Manager" } });
    await user.click(within(details).getByRole("button", { name: "Save changes" }));
    expect(savePositionAction).toHaveBeenCalledWith(
      "chart",
      "manager",
      expect.objectContaining({ title: "Area Sales Manager" }),
    );
  });
});
