import { cleanup, render, screen } from "@testing-library/react";
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

import { OrganogramStage } from "@/features/role-profiles/chart";

afterEach(() => {
  cleanup();
});

describe("organogram workspace", () => {
  it("keeps chart management, builder entry points, and status in separate areas", () => {
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
        positions={{ ok: true, value: [] }}
        files={{ ok: true, value: [] }}
        departments={[{ id: "sales", organisationId: "org", name: "Sales" }]}
        people={[]}
        entry="build"
      />,
    );

    expect(screen.getByRole("heading", { name: "Organogram workspace" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Charts and uploads" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Chart status" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Online builder and positions" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Published chart status" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Position management" })).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Online organogram builder" })).toBeTruthy();
    expect(screen.getByRole("link", { name: "Build online" }).getAttribute("aria-current")).toBe("true");
    expect(screen.getByRole("link", { name: "Upload file" }).getAttribute("href")).toContain("entry=upload");
    expect(screen.getByRole("button", { name: "Save organogram" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Publish chart" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Add position" })).toBeTruthy();
  });
});
