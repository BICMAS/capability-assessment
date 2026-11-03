import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

vi.mock("react-dom", async () => {
  const actual = await vi.importActual<typeof import("react-dom")>("react-dom");
  return { ...actual, useFormStatus: () => ({ pending: false, data: null, method: null, action: null }) };
});

vi.mock("@/features/role-profiles/actions", () => ({
  saveApprovalPathForm: vi.fn(),
}));

import { ApprovalPathConfig } from "@/features/role-profiles/approval-path-config";
import { ReviewInbox } from "@/features/role-profiles/review-inbox";

afterEach(() => {
  cleanup();
});

describe("review inbox", () => {
  it("shows live overview counts, queue records, and view actions", async () => {
    const user = userEvent.setup();
    render(
      <ReviewInbox
        view="awaiting"
        awaiting={{ ok: true, value: 1 }}
        submitted={{ ok: true, value: 0 }}
        reviewed={{ ok: true, value: 2 }}
        queue={{
          ok: true,
          value: [
            {
              profileId: "profile-1",
              roleName: "Area Sales Manager",
              department: "Sales",
              versionId: "version-1",
              versionNumber: 1,
              revision: 1,
              status: "profile_in_review",
              workflowStage: "manager_review",
              stageLabel: "Line Manager Review",
              submittedAt: "2026-10-29T00:00:00Z",
              submittedByPersonId: "ada",
              assigneePersonId: "mgr-1",
              outstanding: "manager",
            },
          ],
        }}
        path={{
          ok: true,
          value: [
            { stageKey: "hr_review", label: "HR/L&D Review", requiredRole: "hr", position: 1 },
            { stageKey: "manager_review", label: "Line Manager Review", requiredRole: "manager", position: 2 },
            { stageKey: "leader_approval", label: "Business Leader Approval", requiredRole: "leader", position: 3 },
          ],
        }}
        personNames={
          new Map([
            ["ada", "Ada Admin"],
            ["mgr-1", "Morgan Manager"],
          ])
        }
      />,
    );

    expect(screen.getByRole("heading", { name: "Approval overview" })).toBeTruthy();
    const overview = screen.getByLabelText("Approval overview");
    expect(within(overview).getByRole("link", { name: /Awaiting my review/ }).textContent).toContain("1");
    expect(within(overview).getByRole("link", { name: /Submitted by me/ }).textContent).toContain("0");
    expect(within(overview).getByRole("link", { name: /Previously reviewed/ }).textContent).toContain("2");
    expect(screen.getByRole("heading", { name: "Approval management" })).toBeTruthy();
    expect(screen.getAllByText("Area Sales Manager").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Sales").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Line Manager Review").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Profile In Review").length).toBeGreaterThan(0);
    expect(screen.getByText("Ada Admin")).toBeTruthy();
    expect(screen.getByText("Morgan Manager")).toBeTruthy();
    expect(screen.getByRole("link", { name: "View" }).getAttribute("href")).toBe("/admin/role-profiles/profile-1/review");
    expect(screen.getByText("Matched preset: Standard approval")).toBeTruthy();

    await user.type(screen.getByRole("searchbox", { name: "Search approvals" }), "finance");
    expect(screen.getByText("No approval records match these filters.")).toBeTruthy();
  });

  it("asks for confirmation before saving an approval path", async () => {
    const user = userEvent.setup();
    render(
      <ApprovalPathConfig
        path={[{ stageKey: "leader_approval", label: "Business Leader Approval", requiredRole: "leader", position: 1 }]}
      />,
    );

    expect(screen.getByText("Matched preset: Executive approval")).toBeTruthy();
    await user.click(screen.getByRole("button", { name: /Standard approval/ }));
    const dialog = screen.getByRole("dialog", { name: "Save standard approval path?" });
    expect(dialog).toBeTruthy();
    expect(within(dialog).getByText(/future submissions only/i)).toBeTruthy();
    expect(within(dialog).getByRole("button", { name: "Save configuration" })).toBeTruthy();
    await user.click(within(dialog).getByRole("button", { name: "Cancel" }));
    expect(screen.queryByRole("dialog", { name: "Save standard approval path?" })).toBeNull();
  });
});
