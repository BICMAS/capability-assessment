import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("next/navigation", () => ({
  useRouter: () => ({ refresh: vi.fn() }),
}));

vi.mock("@/features/role-profiles/chart-actions", () => ({
  createOrganogramForm: vi.fn(),
  addPositionAction: vi.fn(),
  savePositionAction: vi.fn(),
  removePositionAction: vi.fn(),
  publishOrganogramAction: vi.fn(),
}));

import { publishOrganogramAction } from "@/features/role-profiles/chart-actions";
import { OrganogramStage } from "@/features/role-profiles/chart";

afterEach(() => {
  cleanup();
  vi.mocked(publishOrganogramAction).mockReset();
});

describe("organogram workspace", () => {
  it("shows overview metrics, method cards, and guarded publish", async () => {
    render(
      <OrganogramStage
        charts={{
          ok: true,
          value: [
            {
              id: "chart-1",
              organisationId: "org",
              name: "Sample Chart",
              status: "draft",
              createdByPersonId: "ada",
              createdAt: "2026-10-29T00:00:00Z",
            },
          ],
        }}
        chart={{
          id: "chart-1",
          organisationId: "org",
          name: "Sample Chart",
          status: "draft",
          createdByPersonId: "ada",
          createdAt: "2026-10-29T00:00:00Z",
        }}
        positions={{
          ok: true,
          value: [
            {
              id: "p1",
              organisationId: "org",
              organogramId: "chart-1",
              departmentId: "sales",
              title: "Sales Manager",
              personId: "",
              reportsToPositionId: "",
              createdAt: "2026-10-30T00:00:00Z",
            },
          ],
        }}
        files={{ ok: true, value: [] }}
        departments={[{ id: "sales", organisationId: "org", name: "Sales" }]}
        people={[]}
        entry="upload"
      />,
    );

    expect(screen.getByRole("heading", { name: "Organogram onboarding" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Organogram overview" })).toBeTruthy();
    expect(screen.getAllByText("Sample Chart").length).toBeGreaterThan(0);
    expect(screen.getByRole("heading", { name: "Upload your organisation structure" })).toBeTruthy();
    expect(screen.getByText(/Accepted files: PNG, JPG, PDF, XLS, and XLSX/)).toBeTruthy();
    expect(screen.getByRole("link", { name: /Upload spreadsheet/ }).getAttribute("aria-current")).toBe("true");
    expect(screen.getByRole("link", { name: /Build online/ }).getAttribute("href")).toContain("entry=build");
    expect(screen.getByRole("link", { name: /Use template/ }).getAttribute("href")).toContain("entry=template");
    expect(screen.getByRole("link", { name: "Preview chart" }).getAttribute("href")).toContain("entry=build");
    expect(screen.getAllByRole("button", { name: "Save draft" }).length).toBeGreaterThan(0);
    expect(screen.getByText("Sales Manager")).toBeTruthy();

    await userEvent.click(screen.getByRole("button", { name: "Publish chart" }));
    expect(screen.getByRole("dialog", { name: "Publish this organogram?" })).toBeTruthy();
    expect(publishOrganogramAction).not.toHaveBeenCalled();
    await userEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(publishOrganogramAction).not.toHaveBeenCalled();
  });

  it("blocks publish when the chart has no positions", async () => {
    render(
      <OrganogramStage
        charts={{
          ok: true,
          value: [
            {
              id: "chart-1",
              organisationId: "org",
              name: "Empty Chart",
              status: "draft",
              createdByPersonId: "ada",
              createdAt: "2026-10-29T00:00:00Z",
            },
          ],
        }}
        chart={{
          id: "chart-1",
          organisationId: "org",
          name: "Empty Chart",
          status: "draft",
          createdByPersonId: "ada",
          createdAt: "2026-10-29T00:00:00Z",
        }}
        positions={{ ok: true, value: [] }}
        files={{ ok: true, value: [] }}
        departments={[]}
        people={[]}
        entry="build"
      />,
    );

    await userEvent.click(screen.getByRole("button", { name: "Publish chart" }));
    expect(screen.getByText("Add at least one position before publishing this chart.")).toBeTruthy();
    expect(screen.queryByRole("dialog", { name: "Publish this organogram?" })).toBeNull();
    expect(publishOrganogramAction).not.toHaveBeenCalled();
  });
});
