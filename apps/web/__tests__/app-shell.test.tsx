import { cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

const pathname = vi.hoisted(() => ({ current: "/admin" }));
const push = vi.hoisted(() => vi.fn());

vi.mock("next/navigation", () => ({
  usePathname: () => pathname.current,
  useRouter: () => ({ push }),
}));

import { AppShell } from "@/components/app-shell";

const adminSession = {
  personId: "ada",
  organisationId: "bicmas",
  roles: ["admin"],
  managerOf: [],
  name: "Ada Admin",
};

afterEach(() => {
  pathname.current = "/admin";
  push.mockReset();
  cleanup();
});

function adminShell() {
  return (
    <AppShell
      area="admin"
      session={adminSession}
      notifications={[
        {
          id: "n1",
          organisationId: "bicmas",
          personId: "ada",
          title: "Profile submitted",
          body: "Area Sales Manager is waiting.",
          createdAt: "2026-10-30T00:00:00Z",
        },
      ]}
    >
      <p>Today</p>
    </AppShell>
  );
}

describe("application shell", () => {
  it("activates the existing admin routes and leaves future items inactive", () => {
    render(adminShell());

    const nav = screen.getByRole("navigation", { name: "Sections" });
    expect(within(nav).getByRole("link", { name: "Home" }).getAttribute("aria-current")).toBe("page");
    expect(within(nav).getByRole("link", { name: "Role skills profiles" }).getAttribute("href")).toBe("/admin/role-profiles");
    expect(within(nav).getByRole("link", { name: "Review & Approvals" }).getAttribute("href")).toBe("/admin/role-profiles/review");
    expect(within(nav).queryByRole("link", { name: /Settings/ })).toBeNull();
    expect(within(nav).queryByRole("link", { name: /Skills Insights/ })).toBeNull();
    expect(within(nav).queryByRole("link", { name: /Learning & Development/ })).toBeNull();
    expect(within(nav).queryByRole("link", { name: /Employees/ })).toBeNull();
    expect(within(nav).queryByRole("link", { name: /Organisational Structure/ })).toBeNull();
    expect(screen.getAllByText("Not available yet")).toHaveLength(5);
    expect(screen.getByRole("heading", { name: "Home" })).toBeTruthy();
    expect(screen.getByText("Admin")).toBeTruthy();
  });

  it("keeps review active when the inbox is open and signs out from the account menu", async () => {
    pathname.current = "/admin/role-profiles/review";
    render(adminShell());

    const nav = screen.getByRole("navigation", { name: "Sections" });
    expect(within(nav).getByRole("link", { name: "Review & Approvals" }).getAttribute("aria-current")).toBe("page");
    expect(within(nav).getByRole("link", { name: "Home" }).getAttribute("aria-current")).toBeNull();
    expect(screen.getByRole("heading", { name: "Review & Approvals" })).toBeTruthy();
    expect(screen.getByText("Manage submitted role skills profiles and configure approval workflows.")).toBeTruthy();

    await userEvent.click(screen.getAllByRole("button", { name: "Account menu for Ada Admin" })[0]);
    const signOut = screen.getByRole("button", { name: "Sign out" });
    expect(signOut.closest("form")?.getAttribute("action")).toBe("/session/sign-out");
  });

  it("searches only pages that already exist", async () => {
    render(adminShell());
    const search = screen.getByRole("searchbox", { name: "Search pages" });
    await userEvent.type(search, "review");
    const matches = screen.getByRole("listbox", { name: "Matching pages" });
    expect(within(matches).getByRole("link", { name: "Review & Approvals" })).toBeTruthy();
    await userEvent.clear(search);
    await userEvent.type(search, "settings");
    expect(screen.getByText("No matching page.")).toBeTruthy();
    expect(within(screen.getByRole("navigation", { name: "Sections" })).queryByRole("link", { name: "Settings" })).toBeNull();
  });

  it("shows a stored notification and says when help is unavailable", async () => {
    render(adminShell());
    await userEvent.click(screen.getByRole("button", { name: "Notifications, 1 unread" }));
    expect(screen.getByText("Profile submitted")).toBeTruthy();
    await userEvent.click(screen.getByRole("button", { name: "Help" }));
    expect(screen.getByText("Help is not available yet.")).toBeTruthy();
  });

  it("shows employee capability navigation and header without admin routes", () => {
    pathname.current = "/employee";
    render(
      <AppShell
        area="employee"
        session={{ personId: "eli", organisationId: "bicmas", roles: ["employee"], managerOf: [], name: "Eli Employee" }}
        notifications={[]}
      >
        <p>Capability body</p>
      </AppShell>,
    );

    const nav = screen.getByRole("navigation", { name: "Sections" });
    expect(within(nav).getByRole("link", { name: "My capability" }).getAttribute("href")).toBe("/employee");
    expect(within(nav).getByRole("link", { name: "My capability" }).getAttribute("aria-current")).toBe("page");
    expect(within(nav).queryByRole("link", { name: "Role skills profiles" })).toBeNull();
    expect(within(nav).queryByRole("link", { name: "Review & Approvals" })).toBeNull();
    expect(within(nav).queryByRole("link", { name: "My team" })).toBeNull();
    expect(screen.getByText("Capability")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "My capability" })).toBeTruthy();
    expect(screen.getByText(/Complete assigned assessments/i)).toBeTruthy();
  });

  it("shows manager team navigation and person header without admin routes", () => {
    pathname.current = "/manager/people/eli";
    render(
      <AppShell
        area="manager"
        session={{ personId: "maya", organisationId: "bicmas", roles: ["manager"], managerOf: ["eli"], name: "Maya Manager" }}
        notifications={[]}
      >
        <p>Person body</p>
      </AppShell>,
    );

    const nav = screen.getByRole("navigation", { name: "Sections" });
    expect(within(nav).getByRole("link", { name: "My team" }).getAttribute("href")).toBe("/manager");
    expect(within(nav).getByRole("link", { name: "My team" }).getAttribute("aria-current")).toBe("page");
    expect(within(nav).queryByRole("link", { name: "Role skills profiles" })).toBeNull();
    expect(within(nav).queryByRole("link", { name: "My capability" })).toBeNull();
    expect(screen.getByText("Team")).toBeTruthy();
    expect(screen.getByRole("heading", { name: "Team member" })).toBeTruthy();
    expect(screen.getByRole("navigation", { name: "Breadcrumb" }).textContent).toContain("My team");
    expect(screen.getByRole("navigation", { name: "Breadcrumb" }).textContent).toContain("Team member");
  });
});
